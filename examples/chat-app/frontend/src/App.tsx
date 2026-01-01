import { useState, useEffect } from 'react';
import { useChat } from 'ai/react';
import './App.css';

interface Agent {
  id: string;
  name: string;
  description: string;
}

function App() {
  const [agents, setAgents] = useState<Agent[]>([]);
  const [selectedAgent, setSelectedAgent] = useState<string>('assistant');
  const [apiUrl] = useState(import.meta.env.VITE_API_URL || 'http://localhost:8080');

  // Fetch available agents
  useEffect(() => {
    fetch(`${apiUrl}/api/agents`)
      .then(res => res.json())
      .then(data => {
        setAgents(data);
        if (data.length > 0 && !selectedAgent) {
          setSelectedAgent(data[0].id);
        }
      })
      .catch(err => console.error('Failed to fetch agents:', err));
  }, [apiUrl]);

  // Use Vercel AI SDK's useChat hook with custom API
  const { messages, input, handleInputChange, handleSubmit, isLoading, error } = useChat({
    api: `${apiUrl}/api/chat`,
    body: {
      agentId: selectedAgent,
    },
    streamMode: 'text',
    onError: (error) => {
      console.error('Chat error:', error);
    },
  });

  return (
    <div className="app">
      <header className="app-header">
        <h1>🤖 SLOP Chat Agent</h1>
        <div className="agent-selector">
          <label htmlFor="agent-select">Agent: </label>
          <select
            id="agent-select"
            value={selectedAgent}
            onChange={(e) => setSelectedAgent(e.target.value)}
            disabled={isLoading}
          >
            {agents.map(agent => (
              <option key={agent.id} value={agent.id}>
                {agent.name}
              </option>
            ))}
          </select>
          {agents.find(a => a.id === selectedAgent) && (
            <span className="agent-description">
              {agents.find(a => a.id === selectedAgent)?.description}
            </span>
          )}
        </div>
      </header>

      <main className="chat-container">
        <div className="messages">
          {messages.length === 0 && (
            <div className="welcome-message">
              <h2>Welcome to SLOP Chat! 👋</h2>
              <p>Select an agent and start chatting. Your messages will be processed by SLOP scripts on the backend.</p>
            </div>
          )}

          {messages.map((message) => (
            <div
              key={message.id}
              className={`message ${message.role}`}
            >
              <div className="message-role">
                {message.role === 'user' ? '👤 You' : '🤖 Agent'}
              </div>
              <div className="message-content">
                {message.content}
              </div>
            </div>
          ))}

          {isLoading && (
            <div className="message assistant loading">
              <div className="message-role">🤖 Agent</div>
              <div className="message-content">
                <div className="typing-indicator">
                  <span></span>
                  <span></span>
                  <span></span>
                </div>
              </div>
            </div>
          )}

          {error && (
            <div className="error-message">
              Error: {error.message}
            </div>
          )}
        </div>

        <form onSubmit={handleSubmit} className="input-form">
          <input
            type="text"
            value={input}
            onChange={handleInputChange}
            placeholder="Type your message..."
            disabled={isLoading}
            className="message-input"
          />
          <button
            type="submit"
            disabled={isLoading || !input.trim()}
            className="send-button"
          >
            {isLoading ? 'Sending...' : 'Send'}
          </button>
        </form>
      </main>
    </div>
  );
}

export default App;
