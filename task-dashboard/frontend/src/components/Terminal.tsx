import React, { useEffect, useRef, useState } from 'react';
import { Terminal as XTerminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { WebLinksAddon } from '@xterm/addon-web-links';
import { StartTerminalSession } from '../../wailsjs/go/main/App';
import '@xterm/xterm/css/xterm.css';

interface TerminalProps {
  className?: string;
}

interface TerminalMessage {
  type: string;
  data: string;
}

// Global terminal ID that persists across component remounts
let globalTerminalId: string | null = null;

const Terminal: React.FC<TerminalProps> = ({ className = '' }) => {
  const terminalRef = useRef<HTMLDivElement>(null);
  const xtermRef = useRef<XTerminal | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const connectTimeoutRef = useRef<number | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [terminalId, setTerminalId] = useState<string>('');
  const [connectionState, setConnectionState] = useState<'disconnected' | 'connecting' | 'connected' | 'restoring'>('disconnected');

  // Debounce helper
  const debounce = (func: () => void, wait: number) => {
    let timeout: number;
    return () => {
      clearTimeout(timeout);
      timeout = window.setTimeout(func, wait);
    };
  };

  useEffect(() => {
    if (!terminalRef.current) return;

    let mounted = true;

    // Create terminal instance
    const xterm = new XTerminal({
      fontFamily: 'Monaco, Menlo, "Ubuntu Mono", monospace',
      fontSize: 14,
      theme: {
        background: '#1e1e1e',
        foreground: '#ffffff',
        cursor: '#ffffff',
        selectionBackground: '#ffffff40',
        black: '#000000',
        red: '#ff6b6b',
        green: '#51cf66',
        yellow: '#ffd93d',
        blue: '#74c0fc',
        magenta: '#d0bfff',
        cyan: '#5bc0de',
        white: '#ffffff',
        brightBlack: '#495057',
        brightRed: '#ff6b6b',
        brightGreen: '#51cf66',
        brightYellow: '#ffd93d',
        brightBlue: '#74c0fc',
        brightMagenta: '#d0bfff',
        brightCyan: '#5bc0de',
        brightWhite: '#ffffff',
      },
      cursorBlink: true,
      rows: 24,
      cols: 80,
    });

    // Add addons
    const fitAddon = new FitAddon();
    const webLinksAddon = new WebLinksAddon();
    
    xterm.loadAddon(fitAddon);
    xterm.loadAddon(webLinksAddon);
    
    // Open terminal in DOM
    xterm.open(terminalRef.current);
    
    // Focus the terminal immediately
    xterm.focus();
    
    // Initial fit with delay to ensure DOM is ready
    setTimeout(() => {
      try {
        fitAddon.fit();
        // Focus again after fit
        xterm.focus();
      } catch (error) {
        console.warn('Initial terminal fit failed:', error);
      }
    }, 100);
    
    // Store references
    xtermRef.current = xterm;
    fitAddonRef.current = fitAddon;

    // Start terminal session with delay to avoid React Strict Mode double-mounting
    if (mounted) {
      connectTimeoutRef.current = window.setTimeout(() => {
        if (mounted && connectionState === 'disconnected') {
          startTerminalSession();
        }
      }, 200);
    }

    // Handle input from terminal
    xterm.onData((data) => {
      if (!mounted) return;
      if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
        const message: TerminalMessage = {
          type: 'input',
          data: data,
        };
        wsRef.current.send(JSON.stringify(message));
      }
    });

    // Handle resize with debounce - less aggressive
    const handleResize = () => {
      if (fitAddonRef.current && xtermRef.current && terminalRef.current) {
        // Check if the terminal container is still visible
        const rect = terminalRef.current.getBoundingClientRect();
        if (rect.width > 0 && rect.height > 0) {
          setTimeout(() => {
            try {
              fitAddonRef.current?.fit();
            } catch (error) {
              console.warn('Terminal resize failed:', error);
            }
          }, 200);
        }
      }
    };

    const debouncedResize = debounce(handleResize, 300);
    window.addEventListener('resize', debouncedResize);

    // Cleanup
    return () => {
      mounted = false;
      
      // Clear any pending connection timeout
      if (connectTimeoutRef.current) {
        window.clearTimeout(connectTimeoutRef.current);
        connectTimeoutRef.current = null;
      }
      
      // Clean up event listeners
      window.removeEventListener('resize', debouncedResize);
      
      // Close WebSocket connection
      if (wsRef.current) {
        try {
          if (wsRef.current.readyState === WebSocket.OPEN || wsRef.current.readyState === WebSocket.CONNECTING) {
            wsRef.current.close(1000, 'Component unmounting');
          }
          wsRef.current = null;
        } catch (error) {
          console.warn('Error closing WebSocket:', error);
        }
      }
      
      // Dispose terminal
      if (xtermRef.current) {
        try {
          xtermRef.current.dispose();
          xtermRef.current = null;
        } catch (error) {
          console.warn('Error disposing terminal:', error);
        }
      }
      
      setConnectionState('disconnected');
    };
  }, []);

  const startTerminalSession = async () => {
    // Prevent multiple concurrent connections
    if (connectionState !== 'disconnected') {
      return;
    }
    
    try {
      setConnectionState('connecting');
      
      // Reuse existing terminal ID if available, otherwise create new one
      let termId: string;
      if (globalTerminalId) {
        termId = globalTerminalId;
      } else {
        termId = await StartTerminalSession();
        globalTerminalId = termId;
      }
      setTerminalId(termId);

      // Close any existing WebSocket before creating new one
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }

      // Connect to WebSocket server running on the Wails backend
      const wsUrl = `ws://localhost:8080/ws/terminal/${termId}`;
      const ws = new WebSocket(wsUrl);
      
      let isRestoring = false;
      
      ws.onopen = () => {
        setIsConnected(true);
        // Set to restoring if this is a reconnection, connected if new
        if (globalTerminalId === termId) {
          setConnectionState('restoring');
          isRestoring = true;
        } else {
          setConnectionState('connected');
        }
        if (xtermRef.current) {
          xtermRef.current.focus();
        }
      };
      
      ws.onmessage = (event) => {
        try {
          const message: TerminalMessage = JSON.parse(event.data);
          if (xtermRef.current) {
            if (message.type === 'output') {
              xtermRef.current.write(message.data);
              // If we were restoring and now getting new output, restoration is complete
              if (isRestoring) {
                setConnectionState('connected');
                isRestoring = false;
              }
            } else if (message.type === 'history') {
              xtermRef.current.write(message.data);
            }
          }
        } catch (error) {
          console.error('Error parsing WebSocket message:', error);
        }
      };

      ws.onclose = () => {
        setIsConnected(false);
        setConnectionState('disconnected');
      };

      ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        setIsConnected(false);
        setConnectionState('disconnected');
      };

      wsRef.current = ws;
    } catch (error) {
      console.error('Error starting terminal session:', error);
      setConnectionState('disconnected');
      setIsConnected(false);
    }
  };

  return (
    <div className={`terminal-container h-full flex flex-col ${className}`}>
      <div className="bg-gray-800 border border-gray-600 rounded-lg overflow-hidden flex flex-col h-full">
        <div className="bg-gray-700 px-4 py-2 border-b border-gray-600 flex-shrink-0">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-2">
              <div className="flex space-x-1">
                <div className="w-3 h-3 bg-red-500 rounded-full"></div>
                <div className="w-3 h-3 bg-yellow-500 rounded-full"></div>
                <div className="w-3 h-3 bg-green-500 rounded-full"></div>
              </div>
              <span className="text-gray-300 text-sm font-mono">Terminal</span>
            </div>
            <div className="flex items-center space-x-2">
              <div className={`w-2 h-2 rounded-full ${
                connectionState === 'connected' ? 'bg-green-500' : 
                connectionState === 'connecting' ? 'bg-yellow-500' :
                connectionState === 'restoring' ? 'bg-blue-500' : 'bg-red-500'
              }`}></div>
              <span className="text-gray-400 text-xs">
                {connectionState === 'connected' ? 'Connected' : 
                 connectionState === 'connecting' ? 'Connecting...' :
                 connectionState === 'restoring' ? 'Restoring...' : 'Disconnected'}
              </span>
            </div>
          </div>
        </div>
        <div 
          ref={terminalRef} 
          className="terminal-content flex-1 min-h-0"
          style={{ 
            padding: '8px',
          }}
          onClick={() => {
            if (xtermRef.current) {
              xtermRef.current.focus();
            }
          }}
        />
      </div>
    </div>
  );
};

export default Terminal;