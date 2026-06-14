// Package proxy implements protocol proxies for remote sessions.
// Shell provides an interactive PTY-based terminal proxy.
package proxy

import (
	"io"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/creack/pty"
)

// ShellSession represents an interactive shell session over a tunnel.
type ShellSession struct {
	cmd     *exec.Cmd
	ptmx    *os.File
	mu      sync.Mutex
	closed  bool
	onClose func()
}

// NewShellSession starts a PTY-based shell session.
func NewShellSession() (*ShellSession, error) {
	shell := defaultShell()

	cmd := exec.Command(shell)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}

	s := &ShellSession{
		cmd:  cmd,
		ptmx: ptmx,
	}

	slog.Info("shell session started", "shell", shell)
	return s, nil
}

// Read reads from the PTY output (stdout + stderr of the shell).
func (s *ShellSession) Read(p []byte) (int, error) {
	return s.ptmx.Read(p)
}

// Write writes to the PTY input (stdin of the shell).
func (s *ShellSession) Write(p []byte) (int, error) {
	return s.ptmx.Write(p)
}

// Resize changes the PTY window size.
func (s *ShellSession) Resize(rows, cols uint16) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	return pty.Setsize(s.ptmx, &pty.Winsize{
		Rows: rows,
		Cols: cols,
	})
}

// Close ends the shell session.
func (s *ShellSession) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true

	// Send SIGHUP to the process group
	if s.cmd.Process != nil {
		s.cmd.Process.Signal(syscall.SIGHUP)
	}

	err := s.ptmx.Close()

	// Wait for the process to exit
	if s.cmd.Process != nil {
		s.cmd.Wait()
	}

	if s.onClose != nil {
		s.onClose()
	}

	slog.Info("shell session closed")
	return err
}

// OnClose registers a callback for when the session closes.
func (s *ShellSession) OnClose(fn func()) {
	s.onClose = fn
}

// Pipe bidirectionally copies data between the shell and a tunnel stream.
// Blocks until one side closes or an error occurs.
func (s *ShellSession) Pipe(stream io.ReadWriter) error {
	var wg sync.WaitGroup
	wg.Add(2)

	errCh := make(chan error, 2)

	// Shell output -> tunnel stream
	go func() {
		defer wg.Done()
		_, err := io.Copy(stream, s)
		errCh <- err
	}()

	// Tunnel stream -> shell input
	go func() {
		defer wg.Done()
		_, err := io.Copy(s, stream)
		errCh <- err
	}()

	// Wait for both directions to finish
	wg.Wait()
	close(errCh)

	// Return the first error encountered
	for err := range errCh {
		if err != nil {
			return err
		}
	}
	return nil
}

// defaultShell returns the user's preferred shell or a fallback.
func defaultShell() string {
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}
	// Check for common shells
	for _, shell := range []string{"/bin/bash", "/bin/zsh", "/bin/sh"} {
		if _, err := os.Stat(shell); err == nil {
			return shell
		}
	}
	return "/bin/sh"
}