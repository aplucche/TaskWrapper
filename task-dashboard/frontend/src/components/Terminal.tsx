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

const Terminal: React.FC<TerminalProps> = ({ className = '' }) => {
  const terminalRef = useRef<HTMLDivElement>(null);
  const xtermRef = useRef<XTerminal | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [terminalId, setTerminalId] = useState<string>('');

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

    // Start terminal session
    startTerminalSession();

    // Handle input from terminal
    xterm.onData((data) => {
      console.log('Terminal input data:', data);
      if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
        const message: TerminalMessage = {
          type: 'input',
          data: data,
        };
        console.log('Sending to WebSocket:', message);
        wsRef.current.send(JSON.stringify(message));
      } else {
        console.log('WebSocket not ready, state:', wsRef.current?.readyState);
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
      window.removeEventListener('resize', debouncedResize);
      if (wsRef.current) {
        wsRef.current.close();
      }
      if (xtermRef.current) {
        xtermRef.current.dispose();
      }
    };
  }, []);

  const startTerminalSession = async () => {
    try {
      // Get terminal ID from Wails backend
      const termId = await StartTerminalSession();
      console.log('Terminal ID received:', termId);
      setTerminalId(termId);

      // Connect to WebSocket server running on the Wails backend
      const wsUrl = `ws://localhost:8080/ws/terminal/${termId}`;
      console.log('Connecting to WebSocket:', wsUrl);
      const ws = new WebSocket(wsUrl);
      
      ws.onopen = () => {
        console.log('WebSocket connection opened');
        setIsConnected(true);
        if (xtermRef.current) {
          xtermRef.current.write('\r\n\x1b[32mTerminal connected!\x1b[0m\r\n');
          xtermRef.current.write('Welcome to the integrated terminal. You can run any command including Claude CLI.\r\n');
          // Focus the terminal when connected
          xtermRef.current.focus();
        }
      };

      ws.onmessage = (event) => {
        try {
          console.log('WebSocket message received:', event.data);
          const message: TerminalMessage = JSON.parse(event.data);
          if (message.type === 'output' && xtermRef.current) {
            xtermRef.current.write(message.data);
          }
        } catch (error) {
          console.error('Error parsing WebSocket message:', error);
        }
      };

      ws.onclose = () => {
        setIsConnected(false);
        if (xtermRef.current) {
          xtermRef.current.write('\r\n\x1b[31mTerminal disconnected!\x1b[0m\r\n');
        }
      };

      ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        setIsConnected(false);
        if (xtermRef.current) {
          xtermRef.current.write('\r\n\x1b[31mWebSocket connection failed. Terminal features will be limited.\x1b[0m\r\n');
          xtermRef.current.write('This is a demo terminal. WebSocket server error.\r\n');
        }
      };

      wsRef.current = ws;
    } catch (error) {
      console.error('Error starting terminal session:', error);
      if (xtermRef.current) {
        xtermRef.current.write('\r\n\x1b[31mFailed to start terminal session.\x1b[0m\r\n');
      }
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
              <div className={`w-2 h-2 rounded-full ${isConnected ? 'bg-green-500' : 'bg-red-500'}`}></div>
              <span className="text-gray-400 text-xs">
                {isConnected ? 'Connected' : 'Disconnected'}
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