package runtime

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/anthropics/slop/internal/ast"
	"github.com/anthropics/slop/internal/builtin"
	"github.com/anthropics/slop/internal/evaluator"
	"github.com/anthropics/slop/internal/lexer"
	"github.com/anthropics/slop/internal/parser"
)

// =============================================================================
// Enhanced LLM Mock
// =============================================================================

// LLMCall represents a recorded LLM call for inspection.
type LLMCall struct {
	Index     int
	Request   *LLMRequest
	Response  *LLMResponse
	Error     error
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration

	// Token metrics (simulated)
	InputTokens  int
	OutputTokens int
}

// ResponseMatcher defines how to match and respond to LLM requests.
type ResponseMatcher struct {
	// Match conditions (all non-nil conditions must match)
	PromptContains    string
	PromptRegex       *regexp.Regexp
	SystemContains    string
	ModelEquals       string
	SchemaHasField    string
	CallIndex         *int // Match specific call index (0-based)

	// Response to return when matched
	Response *LLMResponse
	Error    error

	// Dynamic response generator (takes precedence over Response)
	Handler func(req *LLMRequest) (*LLMResponse, error)
}

// Matches returns true if the request matches this matcher's conditions.
func (m *ResponseMatcher) Matches(req *LLMRequest, callIndex int) bool {
	if m.PromptContains != "" && !strings.Contains(req.Prompt, m.PromptContains) {
		return false
	}
	if m.PromptRegex != nil && !m.PromptRegex.MatchString(req.Prompt) {
		return false
	}
	if m.SystemContains != "" && !strings.Contains(req.System, m.SystemContains) {
		return false
	}
	if m.ModelEquals != "" && req.Model != m.ModelEquals {
		return false
	}
	if m.SchemaHasField != "" {
		if req.Schema == nil || req.Schema.Properties == nil {
			return false
		}
		if _, ok := req.Schema.Properties[m.SchemaHasField]; !ok {
			return false
		}
	}
	if m.CallIndex != nil && *m.CallIndex != callIndex {
		return false
	}
	return true
}

// TestLLMClient is an enhanced mock LLM client for testing.
type TestLLMClient struct {
	mu sync.Mutex

	// Call history
	Calls     []LLMCall
	callIndex int64 // Atomic counter for unique call indices

	// Response matchers (evaluated in order, first match wins)
	Matchers []*ResponseMatcher

	// Default response generator (used when no matcher matches)
	DefaultHandler func(req *LLMRequest) (*LLMResponse, error)

	// Failure simulation
	FailAfterCalls  int   // Fail after N successful calls (0 = never)
	FailureError    error
	FailureCount    int   // Number of times to fail before recovering (0 = fail forever)
	currentFailures int

	// Latency simulation
	MinLatency time.Duration // Minimum response latency
	MaxLatency time.Duration // Maximum response latency (for random range)
	Latency    time.Duration // Fixed latency (use if MinLatency and MaxLatency are 0)

	// Token tracking
	TokensPerChar     float64 // Tokens per character for estimation (default: 0.25)
	TotalInputTokens  int64   // Total input tokens across all calls
	TotalOutputTokens int64   // Total output tokens across all calls

	// Concurrency tracking
	ConcurrentCalls int32 // Current number of concurrent calls
	MaxConcurrent   int32 // Maximum concurrent calls observed

	// Streaming simulation
	StreamChunkSize    int           // Characters per stream chunk
	StreamChunkDelay   time.Duration // Delay between stream chunks
	StreamingEnabled   bool          // Whether to simulate streaming
	OnStreamChunk      func(chunk string) // Callback for each stream chunk
}

// NewTestLLMClient creates a new test LLM client with schema-based defaults.
func NewTestLLMClient() *TestLLMClient {
	return &TestLLMClient{
		DefaultHandler: func(req *LLMRequest) (*LLMResponse, error) {
			return &LLMResponse{
				Parsed: generateMockResponse(req.Schema),
			}, nil
		},
		TokensPerChar:   0.25, // ~4 chars per token on average
		StreamChunkSize: 20,   // Default chunk size for streaming
	}
}

// WithLatency sets a fixed latency for all LLM calls.
func (c *TestLLMClient) WithLatency(d time.Duration) *TestLLMClient {
	c.Latency = d
	return c
}

// WithLatencyRange sets a random latency range for LLM calls.
func (c *TestLLMClient) WithLatencyRange(min, max time.Duration) *TestLLMClient {
	c.MinLatency = min
	c.MaxLatency = max
	return c
}

// WithStreaming enables streaming simulation.
func (c *TestLLMClient) WithStreaming(chunkSize int, chunkDelay time.Duration) *TestLLMClient {
	c.StreamingEnabled = true
	c.StreamChunkSize = chunkSize
	c.StreamChunkDelay = chunkDelay
	return c
}

// estimateTokens estimates token count based on text length.
func (c *TestLLMClient) estimateTokens(text string) int {
	return int(float64(len(text)) * c.TokensPerChar)
}

// Complete implements LLMClient.
func (c *TestLLMClient) Complete(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	startTime := time.Now()

	// Get unique call index atomically (before any locks)
	callIndex := int(atomic.AddInt64(&c.callIndex, 1) - 1)

	// Track concurrency (atomic operations outside lock)
	current := atomic.AddInt32(&c.ConcurrentCalls, 1)
	defer atomic.AddInt32(&c.ConcurrentCalls, -1)

	// Update max concurrent if this is a new high
	for {
		max := atomic.LoadInt32(&c.MaxConcurrent)
		if current <= max || atomic.CompareAndSwapInt32(&c.MaxConcurrent, max, current) {
			break
		}
	}

	c.mu.Lock()

	// Check for simulated failure
	if c.FailAfterCalls > 0 && callIndex >= c.FailAfterCalls {
		if c.FailureCount == 0 || c.currentFailures < c.FailureCount {
			c.currentFailures++
			call := LLMCall{
				Index:     callIndex,
				Request:   req,
				Error:     c.FailureError,
				StartTime: startTime,
				EndTime:   time.Now(),
				Duration:  time.Since(startTime),
			}
			c.Calls = append(c.Calls, call)
			c.mu.Unlock()
			return nil, c.FailureError
		}
	}

	// Find matching response
	var resp *LLMResponse
	var err error

	for _, matcher := range c.Matchers {
		if matcher.Matches(req, callIndex) {
			if matcher.Handler != nil {
				resp, err = matcher.Handler(req)
			} else if matcher.Error != nil {
				err = matcher.Error
			} else {
				resp = matcher.Response
			}
			break
		}
	}

	// Use default handler if no match
	if resp == nil && err == nil && c.DefaultHandler != nil {
		resp, err = c.DefaultHandler(req)
	}

	// Calculate token counts
	inputTokens := c.estimateTokens(req.Prompt)
	if req.System != "" {
		inputTokens += c.estimateTokens(req.System)
	}
	outputTokens := 0
	if resp != nil && resp.Content != "" {
		outputTokens = c.estimateTokens(resp.Content)
	} else if resp != nil && resp.Parsed != nil {
		// Estimate based on parsed content (rough approximation)
		outputTokens = 50 // Default for parsed responses
	}

	// Update total token counts
	atomic.AddInt64(&c.TotalInputTokens, int64(inputTokens))
	atomic.AddInt64(&c.TotalOutputTokens, int64(outputTokens))

	// Capture latency setting before unlock for simulation
	latency := c.Latency
	minLatency := c.MinLatency
	maxLatency := c.MaxLatency
	streamingEnabled := c.StreamingEnabled
	streamChunkDelay := c.StreamChunkDelay
	streamChunkSize := c.StreamChunkSize
	onStreamChunk := c.OnStreamChunk

	c.mu.Unlock()

	// Simulate latency (outside lock to allow concurrency)
	if latency > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(latency):
		}
	} else if minLatency > 0 && maxLatency > minLatency {
		// Random latency between min and max
		jitter := time.Duration(time.Now().UnixNano() % int64(maxLatency-minLatency))
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(minLatency + jitter):
		}
	}

	// Simulate streaming if enabled
	if streamingEnabled && resp != nil && resp.Content != "" && onStreamChunk != nil {
		content := resp.Content
		for i := 0; i < len(content); i += streamChunkSize {
			end := i + streamChunkSize
			if end > len(content) {
				end = len(content)
			}
			onStreamChunk(content[i:end])
			if streamChunkDelay > 0 {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(streamChunkDelay):
				}
			}
		}
	}

	endTime := time.Now()

	// Record the call
	c.mu.Lock()
	call := LLMCall{
		Index:        callIndex,
		Request:      req,
		Response:     resp,
		Error:        err,
		StartTime:    startTime,
		EndTime:      endTime,
		Duration:     endTime.Sub(startTime),
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
	}
	c.Calls = append(c.Calls, call)
	c.mu.Unlock()

	return resp, err
}

// OnPromptContaining adds a matcher for prompts containing the given text.
func (c *TestLLMClient) OnPromptContaining(text string) *MatcherBuilder {
	matcher := &ResponseMatcher{PromptContains: text}
	c.Matchers = append(c.Matchers, matcher)
	return &MatcherBuilder{client: c, matcher: matcher}
}

// OnPromptMatching adds a matcher for prompts matching the given regex.
func (c *TestLLMClient) OnPromptMatching(pattern string) *MatcherBuilder {
	matcher := &ResponseMatcher{PromptRegex: regexp.MustCompile(pattern)}
	c.Matchers = append(c.Matchers, matcher)
	return &MatcherBuilder{client: c, matcher: matcher}
}

// OnCallIndex adds a matcher for a specific call index.
func (c *TestLLMClient) OnCallIndex(index int) *MatcherBuilder {
	matcher := &ResponseMatcher{CallIndex: &index}
	c.Matchers = append(c.Matchers, matcher)
	return &MatcherBuilder{client: c, matcher: matcher}
}

// OnSchemaHasField adds a matcher for schemas containing a specific field.
func (c *TestLLMClient) OnSchemaHasField(field string) *MatcherBuilder {
	matcher := &ResponseMatcher{SchemaHasField: field}
	c.Matchers = append(c.Matchers, matcher)
	return &MatcherBuilder{client: c, matcher: matcher}
}

// MatcherBuilder provides a fluent API for building response matchers.
type MatcherBuilder struct {
	client  *TestLLMClient
	matcher *ResponseMatcher
}

// AndPromptContains adds an additional prompt contains condition.
func (b *MatcherBuilder) AndPromptContains(text string) *MatcherBuilder {
	b.matcher.PromptContains = text
	return b
}

// AndSystemContains adds a system prompt contains condition.
func (b *MatcherBuilder) AndSystemContains(text string) *MatcherBuilder {
	b.matcher.SystemContains = text
	return b
}

// AndModel adds a model equals condition.
func (b *MatcherBuilder) AndModel(model string) *MatcherBuilder {
	b.matcher.ModelEquals = model
	return b
}

// RespondWith sets a static response.
func (b *MatcherBuilder) RespondWith(parsed any) *MatcherBuilder {
	b.matcher.Response = &LLMResponse{Parsed: parsed}
	return b
}

// RespondWithContent sets a response with raw content.
func (b *MatcherBuilder) RespondWithContent(content string) *MatcherBuilder {
	b.matcher.Response = &LLMResponse{Content: content}
	return b
}

// RespondWithHandler sets a dynamic response handler.
func (b *MatcherBuilder) RespondWithHandler(handler func(req *LLMRequest) (*LLMResponse, error)) *MatcherBuilder {
	b.matcher.Handler = handler
	return b
}

// RespondWithError makes this matcher return an error.
func (b *MatcherBuilder) RespondWithError(err error) *MatcherBuilder {
	b.matcher.Error = err
	return b
}

// Assertion helpers for TestLLMClient

// CallCount returns the number of LLM calls made.
func (c *TestLLMClient) CallCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.Calls)
}

// GetCall returns a specific call by index.
func (c *TestLLMClient) GetCall(index int) (*LLMCall, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if index < 0 || index >= len(c.Calls) {
		return nil, false
	}
	return &c.Calls[index], true
}

// LastCall returns the most recent call.
func (c *TestLLMClient) LastCall() (*LLMCall, bool) {
	return c.GetCall(c.CallCount() - 1)
}

// AssertCallCount checks that the expected number of calls were made.
func (c *TestLLMClient) AssertCallCount(expected int) error {
	actual := c.CallCount()
	if actual != expected {
		return fmt.Errorf("expected %d LLM calls, got %d", expected, actual)
	}
	return nil
}

// AssertPromptContains checks that a specific call's prompt contains text.
func (c *TestLLMClient) AssertPromptContains(callIndex int, text string) error {
	call, ok := c.GetCall(callIndex)
	if !ok {
		return fmt.Errorf("call index %d out of range (total calls: %d)", callIndex, c.CallCount())
	}
	if !strings.Contains(call.Request.Prompt, text) {
		return fmt.Errorf("call %d prompt does not contain %q\nPrompt: %s", callIndex, text, call.Request.Prompt)
	}
	return nil
}

// AssertAnyPromptContains checks that any call's prompt contains text.
func (c *TestLLMClient) AssertAnyPromptContains(text string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, call := range c.Calls {
		if strings.Contains(call.Request.Prompt, text) {
			return nil
		}
	}
	return fmt.Errorf("no LLM call prompt contains %q", text)
}

// Reset clears all recorded calls and resets metrics.
func (c *TestLLMClient) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Calls = nil
	c.currentFailures = 0
	atomic.StoreInt64(&c.callIndex, 0)
	atomic.StoreInt64(&c.TotalInputTokens, 0)
	atomic.StoreInt64(&c.TotalOutputTokens, 0)
	atomic.StoreInt32(&c.MaxConcurrent, 0)
}

// TotalTokens returns total input and output tokens across all calls.
func (c *TestLLMClient) TotalTokens() (input int64, output int64) {
	return atomic.LoadInt64(&c.TotalInputTokens), atomic.LoadInt64(&c.TotalOutputTokens)
}

// GetMaxConcurrency returns the maximum number of concurrent calls observed.
func (c *TestLLMClient) GetMaxConcurrency() int32 {
	return atomic.LoadInt32(&c.MaxConcurrent)
}

// TotalDuration returns the sum of all call durations.
func (c *TestLLMClient) TotalDuration() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	var total time.Duration
	for _, call := range c.Calls {
		total += call.Duration
	}
	return total
}

// AverageDuration returns the average call duration.
func (c *TestLLMClient) AverageDuration() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.Calls) == 0 {
		return 0
	}
	var total time.Duration
	for _, call := range c.Calls {
		total += call.Duration
	}
	return total / time.Duration(len(c.Calls))
}

// TokenCost calculates estimated cost based on token counts.
// Prices are per 1M tokens (inputPrice for input, outputPrice for output).
func (c *TestLLMClient) TokenCost(inputPrice, outputPrice float64) float64 {
	input, output := c.TotalTokens()
	return (float64(input)/1_000_000)*inputPrice + (float64(output)/1_000_000)*outputPrice
}

// =============================================================================
// Enhanced Service Mock (for MCP tools)
// =============================================================================

// ServiceCall represents a recorded service call.
type ServiceCall struct {
	Index  int
	Method string
	Args   []evaluator.Value
	Kwargs map[string]evaluator.Value
	Result evaluator.Value
	Error  error
}

// ServiceMatcher defines how to match and respond to service calls.
type ServiceMatcher struct {
	// Match conditions
	MethodEquals   string
	MethodRegex    *regexp.Regexp
	HasKwarg       string
	KwargEquals    map[string]any
	CallIndex      *int

	// Response
	Result  evaluator.Value
	Error   error
	Handler func(method string, args []evaluator.Value, kwargs map[string]evaluator.Value) (evaluator.Value, error)
}

// Matches returns true if the call matches this matcher.
func (m *ServiceMatcher) Matches(method string, kwargs map[string]evaluator.Value, callIndex int) bool {
	if m.MethodEquals != "" && method != m.MethodEquals {
		return false
	}
	if m.MethodRegex != nil && !m.MethodRegex.MatchString(method) {
		return false
	}
	if m.HasKwarg != "" {
		if _, ok := kwargs[m.HasKwarg]; !ok {
			return false
		}
	}
	if m.CallIndex != nil && *m.CallIndex != callIndex {
		return false
	}
	return true
}

// TestService is an enhanced mock service for testing MCP-like tools.
type TestService struct {
	mu sync.Mutex

	name     string
	methods  []string
	Calls    []ServiceCall
	Matchers []*ServiceMatcher

	// Default behavior
	DefaultResult  evaluator.Value
	DefaultHandler func(method string, args []evaluator.Value, kwargs map[string]evaluator.Value) (evaluator.Value, error)

	// Failure simulation
	FailAfterCalls int
	FailureError   error
	FailureMethods map[string]error // Per-method failures
}

// NewTestService creates a new test service.
func NewTestService(name string) *TestService {
	return &TestService{
		name:           name,
		methods:        []string{},
		DefaultResult:  evaluator.NONE,
		FailureMethods: make(map[string]error),
	}
}

// Name implements Service.
func (s *TestService) Name() string {
	return s.name
}

// Methods implements Service.
func (s *TestService) Methods() []string {
	return s.methods
}

// SetMethods sets the available methods.
func (s *TestService) SetMethods(methods ...string) *TestService {
	s.methods = methods
	return s
}

// Call implements Service.
func (s *TestService) Call(method string, args []evaluator.Value, kwargs map[string]evaluator.Value) (evaluator.Value, error) {
	return s.CallWithContext(context.Background(), method, args, kwargs)
}

// CallWithContext implements ServiceWithContext.
func (s *TestService) CallWithContext(_ context.Context, method string, args []evaluator.Value, kwargs map[string]evaluator.Value) (evaluator.Value, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	callIndex := len(s.Calls)

	// Check for per-method failure
	if err, ok := s.FailureMethods[method]; ok {
		call := ServiceCall{Index: callIndex, Method: method, Args: args, Kwargs: kwargs, Error: err}
		s.Calls = append(s.Calls, call)
		return nil, err
	}

	// Check for general failure
	if s.FailAfterCalls > 0 && callIndex >= s.FailAfterCalls {
		call := ServiceCall{Index: callIndex, Method: method, Args: args, Kwargs: kwargs, Error: s.FailureError}
		s.Calls = append(s.Calls, call)
		return nil, s.FailureError
	}

	// Find matching response
	var result evaluator.Value
	var err error

	for _, matcher := range s.Matchers {
		if matcher.Matches(method, kwargs, callIndex) {
			if matcher.Handler != nil {
				result, err = matcher.Handler(method, args, kwargs)
			} else if matcher.Error != nil {
				err = matcher.Error
			} else {
				result = matcher.Result
			}
			break
		}
	}

	// Use default if no match
	if result == nil && err == nil {
		if s.DefaultHandler != nil {
			result, err = s.DefaultHandler(method, args, kwargs)
		} else {
			result = s.DefaultResult
		}
	}

	// Record the call
	call := ServiceCall{Index: callIndex, Method: method, Args: args, Kwargs: kwargs, Result: result, Error: err}
	s.Calls = append(s.Calls, call)

	return result, err
}

// Close implements Service.
func (s *TestService) Close() error {
	return nil
}

// OnMethod adds a matcher for a specific method.
func (s *TestService) OnMethod(method string) *ServiceMatcherBuilder {
	matcher := &ServiceMatcher{MethodEquals: method}
	s.Matchers = append(s.Matchers, matcher)
	return &ServiceMatcherBuilder{service: s, matcher: matcher}
}

// OnCallIndex adds a matcher for a specific call index.
func (s *TestService) OnCallIndex(index int) *ServiceMatcherBuilder {
	matcher := &ServiceMatcher{CallIndex: &index}
	s.Matchers = append(s.Matchers, matcher)
	return &ServiceMatcherBuilder{service: s, matcher: matcher}
}

// ServiceMatcherBuilder provides a fluent API for service matchers.
type ServiceMatcherBuilder struct {
	service *TestService
	matcher *ServiceMatcher
}

// AndMethod adds a method condition.
func (b *ServiceMatcherBuilder) AndMethod(method string) *ServiceMatcherBuilder {
	b.matcher.MethodEquals = method
	return b
}

// AndHasKwarg adds a kwarg presence condition.
func (b *ServiceMatcherBuilder) AndHasKwarg(key string) *ServiceMatcherBuilder {
	b.matcher.HasKwarg = key
	return b
}

// Return sets the return value.
func (b *ServiceMatcherBuilder) Return(result evaluator.Value) *ServiceMatcherBuilder {
	b.matcher.Result = result
	return b
}

// ReturnMap sets a map return value.
func (b *ServiceMatcherBuilder) ReturnMap(pairs map[string]any) *ServiceMatcherBuilder {
	m := evaluator.NewMapValue()
	for k, v := range pairs {
		m.Set(k, anyToValue(v))
	}
	b.matcher.Result = m
	return b
}

// ReturnString sets a string return value.
func (b *ServiceMatcherBuilder) ReturnString(s string) *ServiceMatcherBuilder {
	b.matcher.Result = &evaluator.StringValue{Value: s}
	return b
}

// ReturnError sets an error response.
func (b *ServiceMatcherBuilder) ReturnError(err error) *ServiceMatcherBuilder {
	b.matcher.Error = err
	return b
}

// ReturnHandler sets a dynamic handler.
func (b *ServiceMatcherBuilder) ReturnHandler(handler func(method string, args []evaluator.Value, kwargs map[string]evaluator.Value) (evaluator.Value, error)) *ServiceMatcherBuilder {
	b.matcher.Handler = handler
	return b
}

// Assertion helpers

// CallCount returns the number of calls.
func (s *TestService) CallCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.Calls)
}

// MethodCallCount returns calls to a specific method.
func (s *TestService) MethodCallCount(method string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for _, call := range s.Calls {
		if call.Method == method {
			count++
		}
	}
	return count
}

// GetCall returns a specific call.
func (s *TestService) GetCall(index int) (*ServiceCall, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.Calls) {
		return nil, false
	}
	return &s.Calls[index], true
}

// GetMethodCalls returns all calls to a specific method.
func (s *TestService) GetMethodCalls(method string) []ServiceCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	var calls []ServiceCall
	for _, call := range s.Calls {
		if call.Method == method {
			calls = append(calls, call)
		}
	}
	return calls
}

// AssertCalled checks that a method was called.
func (s *TestService) AssertCalled(method string) error {
	if s.MethodCallCount(method) == 0 {
		return fmt.Errorf("method %q was never called", method)
	}
	return nil
}

// AssertNotCalled checks that a method was not called.
func (s *TestService) AssertNotCalled(method string) error {
	if s.MethodCallCount(method) > 0 {
		return fmt.Errorf("method %q was called %d times", method, s.MethodCallCount(method))
	}
	return nil
}

// AssertCallOrder checks that methods were called in a specific order.
func (s *TestService) AssertCallOrder(methods ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(methods) > len(s.Calls) {
		return fmt.Errorf("expected %d calls in order %v, but only %d calls were made", len(methods), methods, len(s.Calls))
	}

	methodIndex := 0
	for _, call := range s.Calls {
		if methodIndex < len(methods) && call.Method == methods[methodIndex] {
			methodIndex++
		}
	}

	if methodIndex != len(methods) {
		actualOrder := make([]string, len(s.Calls))
		for i, call := range s.Calls {
			actualOrder[i] = call.Method
		}
		return fmt.Errorf("expected call order %v, got %v", methods, actualOrder)
	}
	return nil
}

// Reset clears all recorded calls.
func (s *TestService) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Calls = nil
}

// =============================================================================
// Test Runtime Helper
// =============================================================================

// TestRuntime wraps a SLOP runtime with test utilities.
type TestRuntime struct {
	LLM      *TestLLMClient
	Services map[string]*TestService
	ctx      *evaluator.Context
	eval     *evaluator.Evaluator
}

// NewTestRuntime creates a new test runtime with enhanced mocks.
func NewTestRuntime() *TestRuntime {
	llm := NewTestLLMClient()
	ctx := evaluator.NewContext()
	eval := evaluator.NewWithContext(ctx)

	// Register all built-in functions (required for schema types like string, number, etc.)
	registry := builtin.NewRegistry()
	ctx.RegisterBuiltins(registry)

	// Register LLM service
	llmService := NewLLMService(llm)
	ctx.RegisterService("llm", llmService)

	return &TestRuntime{
		LLM:      llm,
		Services: make(map[string]*TestService),
		ctx:      ctx,
		eval:     eval,
	}
}

// AddService adds a test service.
func (r *TestRuntime) AddService(name string) *TestService {
	svc := NewTestService(name)
	r.Services[name] = svc
	r.ctx.RegisterService(name, svc)
	return svc
}

// SetInput sets an input variable.
func (r *TestRuntime) SetInput(key string, value any) {
	input, ok := r.ctx.Scope.Get("input")
	if !ok {
		input = evaluator.NewMapValue()
		r.ctx.Scope.Set("input", input)
	}
	if m, ok := input.(*evaluator.MapValue); ok {
		m.Set(key, anyToValue(value))
	}
}

// Execute runs SLOP source code.
func (r *TestRuntime) Execute(source string) (evaluator.Value, error) {
	// Ensure input exists
	if _, ok := r.ctx.Scope.Get("input"); !ok {
		r.ctx.Scope.Set("input", evaluator.NewMapValue())
	}

	l := newLexer(source)
	p := newParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		return nil, fmt.Errorf("parse errors: %v", p.Errors())
	}

	return r.eval.Eval(program)
}

// Emitted returns all emitted values.
func (r *TestRuntime) Emitted() []evaluator.Value {
	return r.ctx.Emitted
}

// ClearEmitted clears emitted values.
func (r *TestRuntime) ClearEmitted() {
	r.ctx.Emitted = nil
}

// Reset resets all mocks and state.
func (r *TestRuntime) Reset() {
	r.LLM.Reset()
	for _, svc := range r.Services {
		svc.Reset()
	}
	r.ctx.Emitted = nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// parserAdapter wraps a parser.Parser to work with TestRuntime.
type parserAdapter struct {
	p *parser.Parser
}

func (a *parserAdapter) ParseProgram() *ast.Program {
	return a.p.ParseProgram()
}

func (a *parserAdapter) Errors() []string {
	errs := a.p.Errors()
	result := make([]string, len(errs))
	for i, e := range errs {
		result[i] = e.Error()
	}
	return result
}

func newLexer(input string) *lexer.Lexer {
	return lexer.New(input)
}

func newParser(l *lexer.Lexer) *parserAdapter {
	return &parserAdapter{p: parser.New(l)}
}
