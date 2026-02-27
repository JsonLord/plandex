package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"sync"
)

// Transport defines the interface for communicating with an MCP server.
type Transport interface {
	Send(msg interface{}) error
	Receive() (interface{}, error)
	Close() error
}

// StdioTransport implements MCP over stdin/stdout of a subprocess.
type StdioTransport struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
	mu     sync.Mutex
	reader *bufio.Scanner
}

// NewStdioTransport starts a command and wraps its stdin/stdout.
func NewStdioTransport(command string, args ...string) (*StdioTransport, error) {
	cmd := exec.Command(command, args...)

	// Set environment variables if needed, e.g., PATH
	cmd.Env = os.Environ()

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	return &StdioTransport{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
		reader: bufio.NewScanner(stdout),
	}, nil
}

// Send writes a JSON-RPC message to the subprocess's stdin.
func (t *StdioTransport) Send(msg interface{}) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Append newline as required by JSON-RPC over stdio usually (or just good practice for line-based readers)
	if _, err := t.stdin.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write to stdin: %w", err)
	}

	return nil
}

// Receive reads the next JSON object from the subprocess's stdout.
func (t *StdioTransport) Receive() (interface{}, error) {
	if !t.reader.Scan() {
		if err := t.reader.Err(); err != nil {
			return nil, fmt.Errorf("error reading from stdout: %w", err)
		}
		return nil, io.EOF
	}

	line := t.reader.Bytes()

	// Try to unmarshal into a generic map to determine type later
	var msg map[string]interface{}
	if err := json.Unmarshal(line, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	return msg, nil
}

// Close terminates the subprocess and closes pipes.
func (t *StdioTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.stdin != nil {
		t.stdin.Close()
	}
	if t.stdout != nil {
		t.stdout.Close()
	}
	if t.stderr != nil {
		t.stderr.Close()
	}

	if t.cmd != nil && t.cmd.Process != nil {
		// Try to kill if still running
		t.cmd.Process.Kill()
		return t.cmd.Wait()
	}
	return nil
}

// TcpTransport implements MCP over a TCP connection.
type TcpTransport struct {
	conn   net.Conn
	mu     sync.Mutex
	reader *bufio.Scanner
}

// NewTcpTransport creates a new TCP transport connecting to the given address.
func NewTcpTransport(address string) (*TcpTransport, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", address, err)
	}

	return &TcpTransport{
		conn:   conn,
		reader: bufio.NewScanner(conn),
	}, nil
}

// Send writes a JSON-RPC message to the TCP connection.
func (t *TcpTransport) Send(msg interface{}) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Append newline as required by JSON-RPC usually (or just good practice for line-based readers)
	if _, err := t.conn.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write to connection: %w", err)
	}

	return nil
}

// Receive reads the next JSON object from the TCP connection.
func (t *TcpTransport) Receive() (interface{}, error) {
	if !t.reader.Scan() {
		if err := t.reader.Err(); err != nil {
			return nil, fmt.Errorf("error reading from connection: %w", err)
		}
		return nil, io.EOF
	}

	line := t.reader.Bytes()

	// Try to unmarshal into a generic map to determine type later
	var msg map[string]interface{}
	if err := json.Unmarshal(line, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	return msg, nil
}

// Close closes the TCP connection.
func (t *TcpTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.conn.Close()
}
