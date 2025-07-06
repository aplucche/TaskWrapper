package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
	"github.com/google/uuid"
)

// TerminalService handles terminal session management and WebSocket connections
type TerminalService struct {
	terminals   map[string]*Terminal
	mu          sync.RWMutex
	wsStarted   sync.Once
	upgrader    websocket.Upgrader
	logger      Logger
	ctx         context.Context
}

// NewTerminalService creates a new terminal service
func NewTerminalService(logger Logger, allowedOrigins []string) *TerminalService {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			if len(allowedOrigins) == 0 {
				return true // Allow all origins if none specified (development mode)
			}
			origin := r.Header.Get("Origin")
			for _, allowed := range allowedOrigins {
				if origin == allowed {
					return true
				}
			}
			return false
		},
	}

	return &TerminalService{
		terminals: make(map[string]*Terminal),
		upgrader:  upgrader,
		logger:    logger,
	}
}

// SetContext sets the application context
func (ts *TerminalService) SetContext(ctx context.Context) {
	ts.ctx = ctx
}

// StartTerminalSession creates a new terminal session and returns its ID
func (ts *TerminalService) StartTerminalSession() string {
	terminalID := uuid.New().String()
	ts.logger.Info(fmt.Sprintf("Creating terminal session: %s", terminalID))
	
	// Start WebSocket server if not already running
	go ts.startWebSocketServer()
	
	return terminalID
}

// GetTerminal retrieves a terminal by ID
func (ts *TerminalService) GetTerminal(terminalID string) (*Terminal, bool) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	terminal, exists := ts.terminals[terminalID]
	return terminal, exists
}

// startWebSocketServer starts the WebSocket server for terminal sessions
func (ts *TerminalService) startWebSocketServer() {
	ts.wsStarted.Do(func() {
		http.HandleFunc("/ws/terminal/", ts.HandleWebSocket)
		
		go func() {
			ts.logger.Info("Starting WebSocket server on :8080")
			if err := http.ListenAndServe(":8080", nil); err != nil {
				ts.logger.Error("WebSocket server failed", err)
			}
		}()
	})
}

// HandleWebSocket handles WebSocket connections for terminal sessions
func (ts *TerminalService) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract terminal ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid terminal ID", http.StatusBadRequest)
		return
	}
	terminalID := pathParts[3]
	
	ts.logger.Info(fmt.Sprintf("WebSocket connection for terminal: %s", terminalID))
	
	// Upgrade connection to WebSocket
	conn, err := ts.upgrader.Upgrade(w, r, nil)
	if err != nil {
		ts.logger.Error("Failed to upgrade WebSocket connection", err)
		return
	}
	defer conn.Close()
	
	// Check if terminal already exists (reconnection)
	ts.mu.Lock()
	terminal, exists := ts.terminals[terminalID]
	if exists {
		// Reconnect to existing terminal
		terminal.Conn = conn
		ts.logger.Info(fmt.Sprintf("Reconnected to existing terminal: %s", terminalID))
		
		// Send terminal history to reconnecting client
		go ts.sendTerminalHistory(terminal)
	}
	ts.mu.Unlock()
	
	if !exists {
		// Create new terminal session
		var err error
		terminal, err = ts.createTerminal(terminalID, conn)
		if err != nil {
			ts.logger.Error("Failed to create terminal", err)
			return
		}
		
		// Store terminal session
		ts.mu.Lock()
		ts.terminals[terminalID] = terminal
		ts.mu.Unlock()
	}
	
	// Handle messages
	ts.handleTerminalMessages(terminal)
}

// createTerminal creates a new terminal process with PTY
func (ts *TerminalService) createTerminal(terminalID string, conn *websocket.Conn) (*Terminal, error) {
	// Create a new shell process
	cmd := exec.Command("/bin/bash")
	
	// Set environment variables
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
	)
	
	// Start the command with a PTY
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to start terminal with PTY: %v", err)
	}
	
	terminal := &Terminal{
		ID:     terminalID,
		Cmd:    cmd,
		Pty:    ptmx,
		Conn:   conn,
		Done:   make(chan bool),
		Buffer: NewTerminalBuffer(),
	}
	
	ts.logger.Info(fmt.Sprintf("Terminal process started for session %s (PID: %d)", terminalID, cmd.Process.Pid))
	
	// Start goroutine to read from PTY and send to WebSocket
	go ts.readFromPty(terminal)
	
	return terminal, nil
}

// handleTerminalMessages handles the message loop for a terminal session
func (ts *TerminalService) handleTerminalMessages(terminal *Terminal) {
	defer func() {
		// Only close the WebSocket connection, keep terminal running
		if terminal.Conn != nil {
			terminal.Conn.Close()
			terminal.Conn = nil
		}
		ts.logger.Info(fmt.Sprintf("WebSocket disconnected for terminal %s, terminal continues running", terminal.ID))
	}()

	// Handle WebSocket messages
	for {
		var message TerminalMessage
		err := terminal.Conn.ReadJSON(&message)
		if err != nil {
			ts.logger.Error("Failed to read WebSocket message", err)
			break
		}
		
		if message.Type == "input" {
			// Write input to PTY
			_, err := terminal.Pty.Write([]byte(message.Data))
			if err != nil {
				ts.logger.Error("Failed to write to PTY", err)
				break
			}
		}
	}
}

// readFromPty reads output from PTY and sends to WebSocket
func (ts *TerminalService) readFromPty(terminal *Terminal) {
	buffer := make([]byte, 1024)
	
	for {
		n, err := terminal.Pty.Read(buffer)
		if err != nil {
			if err == io.EOF {
				ts.logger.Info(fmt.Sprintf("Terminal %s process ended", terminal.ID))
				// Only now actually clean up the terminal since process ended
				ts.cleanupTerminal(terminal)
			} else {
				ts.logger.Error("Failed to read from PTY", err)
			}
			break
		}
		
		// Store output in buffer for reconnection
		outputData := string(buffer[:n])
		terminal.Buffer.AddLine(outputData)
		
		// Send output to WebSocket if still connected
		if terminal.Conn != nil {
			message := TerminalMessage{
				Type: "output",
				Data: outputData,
			}
			
			if err := terminal.Conn.WriteJSON(message); err != nil {
				ts.logger.Error("Failed to send terminal output to WebSocket", err)
				// Don't break here - just log and continue, WebSocket might reconnect
				terminal.Conn = nil
			}
		}
		// If no WebSocket connection, just continue reading (terminal keeps running)
	}
}

// CleanupTerminal properly cleans up terminal resources by ID
func (ts *TerminalService) CleanupTerminal(terminalID string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	terminal, exists := ts.terminals[terminalID]
	if !exists {
		return
	}
	
	ts.cleanupTerminal(terminal)
}

// cleanupTerminal properly cleans up terminal resources (internal method)
func (ts *TerminalService) cleanupTerminal(terminal *Terminal) {
	if terminal.Pty != nil {
		terminal.Pty.Close()
	}
	if terminal.Cmd != nil && terminal.Cmd.Process != nil {
		terminal.Cmd.Process.Kill()
	}
	if terminal.Conn != nil {
		terminal.Conn.Close()
	}
	
	// Remove from active terminals map
	delete(ts.terminals, terminal.ID)
	ts.logger.Info(fmt.Sprintf("Terminal %s cleaned up", terminal.ID))
}

// sendTerminalHistory sends stored terminal history to a reconnecting client
func (ts *TerminalService) sendTerminalHistory(terminal *Terminal) {
	if terminal.Conn == nil || terminal.Buffer == nil {
		return
	}
	
	history := terminal.Buffer.GetHistory()
	if len(history) == 0 {
		return
	}
	
	// Send history as a special message type
	for _, line := range history {
		// Check if connection is still valid before each send
		if terminal.Conn == nil {
			break
		}
		
		message := TerminalMessage{
			Type: "history",
			Data: line,
		}
		
		if err := terminal.Conn.WriteJSON(message); err != nil {
			ts.logger.Error("Failed to send terminal history", err)
			terminal.Conn = nil
			break
		}
	}
	
	ts.logger.Info(fmt.Sprintf("Sent %d lines of history to terminal %s", len(history), terminal.ID))
}