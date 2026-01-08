package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/standardbeagle/slop/internal/evaluator"
	"github.com/standardbeagle/slop/internal/lexer"
	"github.com/standardbeagle/slop/internal/parser"
	"github.com/standardbeagle/slop/internal/runtime"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Messages []ChatMessage `json:"messages"`
	AgentID  string        `json:"agentId,omitempty"`
}

type ChatResponse struct {
	Content string `json:"content"`
	Done    bool   `json:"done"`
}

// SLOPAgent wraps a SLOP script and provides streaming execution
type SLOPAgent struct {
	scriptPath string
	env        *runtime.Environment
}

func NewSLOPAgent(scriptPath string) (*SLOPAgent, error) {
	// Verify script exists
	if _, err := os.Stat(scriptPath); err != nil {
		return nil, fmt.Errorf("script not found: %w", err)
	}

	// Create runtime environment
	env := runtime.NewEnvironment()

	// Register custom built-in functions if needed
	// For example, you might want to add API access, database functions, etc.

	return &SLOPAgent{
		scriptPath: scriptPath,
		env:        env,
	}, nil
}

func (a *SLOPAgent) Execute(ctx context.Context, messages []ChatMessage) (<-chan string, <-chan error) {
	outputChan := make(chan string, 10)
	errChan := make(chan error, 1)

	go func() {
		defer close(outputChan)
		defer close(errChan)

		// Read SLOP script
		source, err := os.ReadFile(a.scriptPath)
		if err != nil {
			errChan <- fmt.Errorf("failed to read script: %w", err)
			return
		}

		// Parse SLOP script
		l := lexer.New(string(source))
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) > 0 {
			errChan <- fmt.Errorf("parse errors: %v", p.Errors())
			return
		}

		// Convert messages to JSON for SLOP
		messagesJSON, _ := json.Marshal(messages)

		// Set input variables in environment
		a.env.Set("messages", string(messagesJSON))
		a.env.Set("user_message", messages[len(messages)-1].Content)

		// Create evaluator with streaming output
		eval := evaluator.New(a.env)

		// Capture emit statements for streaming
		a.env.OnEmit(func(val runtime.Value) {
			select {
			case <-ctx.Done():
				return
			case outputChan <- val.String():
			}
		})

		// Execute the program
		result := eval.Eval(program)

		if errVal, ok := result.(*runtime.Error); ok {
			errChan <- fmt.Errorf("runtime error: %s", errVal.Message)
			return
		}

		// Send final result if it's not null
		if result.Type() != runtime.NONE_OBJ {
			select {
			case <-ctx.Done():
				return
			case outputChan <- result.String():
			}
		}
	}()

	return outputChan, errChan
}

// ChatHandler handles streaming chat requests
func ChatHandler(agentsDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse request
		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if len(req.Messages) == 0 {
			http.Error(w, "No messages provided", http.StatusBadRequest)
			return
		}

		// Default agent
		agentID := req.AgentID
		if agentID == "" {
			agentID = "assistant"
		}

		// Load SLOP agent
		scriptPath := filepath.Join(agentsDir, agentID+".slop")
		agent, err := NewSLOPAgent(scriptPath)
		if err != nil {
			http.Error(w, fmt.Sprintf("Agent not found: %v", err), http.StatusNotFound)
			return
		}

		// Set up SSE streaming
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		// Execute agent with context
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		outputChan, errChan := agent.Execute(ctx, req.Messages)

		// Stream responses
		for {
			select {
			case <-ctx.Done():
				return
			case err, ok := <-errChan:
				if ok && err != nil {
					data, _ := json.Marshal(ChatResponse{
						Content: fmt.Sprintf("Error: %v", err),
						Done:    true,
					})
					fmt.Fprintf(w, "data: %s\n\n", data)
					flusher.Flush()
				}
				return
			case chunk, ok := <-outputChan:
				if !ok {
					// Stream finished
					data, _ := json.Marshal(ChatResponse{
						Content: "",
						Done:    true,
					})
					fmt.Fprintf(w, "data: %s\n\n", data)
					flusher.Flush()
					return
				}

				// Send chunk
				data, _ := json.Marshal(ChatResponse{
					Content: chunk,
					Done:    false,
				})
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			}
		}
	}
}

// ListAgentsHandler returns available agents
func ListAgentsHandler(agentsDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		files, err := os.ReadDir(agentsDir)
		if err != nil {
			http.Error(w, "Failed to read agents directory", http.StatusInternalServerError)
			return
		}

		agents := []map[string]string{}
		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".slop") {
				agentID := strings.TrimSuffix(file.Name(), ".slop")

				// Try to read description from script
				description := "SLOP Agent"
				if content, err := os.ReadFile(filepath.Join(agentsDir, file.Name())); err == nil {
					scanner := bufio.NewScanner(strings.NewReader(string(content)))
					if scanner.Scan() {
						line := scanner.Text()
						if strings.HasPrefix(line, "# ") {
							description = strings.TrimPrefix(line, "# ")
						}
					}
				}

				agents = append(agents, map[string]string{
					"id":          agentID,
					"name":        agentID,
					"description": description,
				})
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(agents)
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	agentsDir := os.Getenv("AGENTS_DIR")
	if agentsDir == "" {
		agentsDir = "./slop-agents"
	}

	// Ensure agents directory exists
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		log.Fatalf("Failed to create agents directory: %v", err)
	}

	// Routes
	http.HandleFunc("/api/chat", ChatHandler(agentsDir))
	http.HandleFunc("/api/agents", ListAgentsHandler(agentsDir))

	// Health check
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "OK")
	})

	// CORS preflight
	http.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}
	})

	log.Printf("Starting SLOP Chat Server on port %s", port)
	log.Printf("Agents directory: %s", agentsDir)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
