package terminal

import (
	"errors"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/creack/pty"
	sio "github.com/zishang520/socket.io/servers/socket/v3"
)

const (
	// TerminalCols matches TERMINAL_COLS in common/util-common.ts — the width
	// xterm.js initialises at before FitAddon resizes to the actual browser width.
	// Using the same value means docker compose TUI output is formatted correctly
	// and ESC[nA cursor-up sequences land on the right lines.
	TerminalRows         = 30
	TerminalCols         = 105
	ProgressTerminalRows = 10
	CombinedTerminalRows = 100
	CombinedTerminalCols = 58 // matches COMBINED_TERMINAL_COLS in util-common.ts
)

// Emitter is the interface the terminal needs from a connected socket.
type Emitter interface {
	ID() string
	Connected() bool
	// EmitAgent sends an event wrapped in the "agent" envelope the frontend expects.
	EmitAgent(event string, args ...any)
}

// ─── Terminal registry ────────────────────────────────────────────────────────

var (
	terminalMapMu sync.Mutex
	terminalMap   = make(map[string]*Terminal)
)

func registerTerminal(name string, t *Terminal) {
	terminalMapMu.Lock()
	terminalMap[name] = t
	terminalMapMu.Unlock()
}

func unregisterTerminal(name string) {
	terminalMapMu.Lock()
	delete(terminalMap, name)
	terminalMapMu.Unlock()
}

// GetTerminal returns a running terminal by name, or nil.
func GetTerminal(name string) *Terminal {
	terminalMapMu.Lock()
	defer terminalMapMu.Unlock()
	return terminalMap[name]
}

// GetOrCreateTerminal returns an existing terminal or creates a new one.
func GetOrCreateTerminal(name, file string, args []string, cwd string) *Terminal {
	if t := GetTerminal(name); t != nil {
		return t
	}
	return NewTerminal(name, file, args, cwd)
}

// ─── Terminal ─────────────────────────────────────────────────────────────────

// Terminal runs a command in a PTY and broadcasts output to subscribed sockets.
type Terminal struct {
	mu   sync.Mutex
	name string
	file string
	args []string
	cwd  string
	rows uint16
	cols uint16

	ptyFile *os.File
	started bool

	buffer  *LimitQueue
	sockets map[string]Emitter

	EnableKeepAlive bool

	exitCh   chan int
	callback func(exitCode int)

	stopKeepAlive chan struct{}
	stopKicker    chan struct{}
}

// NewTerminal creates a Terminal but does not start the PTY process.
func NewTerminal(name, file string, args []string, cwd string) *Terminal {
	t := &Terminal{
		name:          name,
		file:          file,
		args:          args,
		cwd:           cwd,
		rows:          TerminalRows,
		cols:          TerminalCols,
		buffer:        NewLimitQueue(100),
		sockets:       make(map[string]Emitter),
		exitCh:        make(chan int, 1),
		stopKeepAlive: make(chan struct{}),
		stopKicker:    make(chan struct{}),
	}
	registerTerminal(name, t)
	return t
}

func (t *Terminal) Name() string { return t.name }

func (t *Terminal) SetRows(rows uint16) {
	t.mu.Lock()
	t.rows = rows
	f := t.ptyFile
	t.mu.Unlock()
	if f != nil {
		_ = pty.Setsize(f, &pty.Winsize{Rows: rows, Cols: t.cols})
	}
}

func (t *Terminal) SetCols(cols uint16) {
	t.mu.Lock()
	t.cols = cols
	f := t.ptyFile
	t.mu.Unlock()
	if f != nil {
		_ = pty.Setsize(f, &pty.Winsize{Rows: t.rows, Cols: cols})
	}
}

func (t *Terminal) Resize(rows, cols uint16) {
	t.mu.Lock()
	t.rows = rows
	t.cols = cols
	f := t.ptyFile
	t.mu.Unlock()
	if f != nil {
		_ = pty.Setsize(f, &pty.Winsize{Rows: rows, Cols: cols})
	}
}

func (t *Terminal) GetBuffer() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.buffer.Join()
}

func (t *Terminal) Join(socket Emitter) {
	t.mu.Lock()
	t.sockets[socket.ID()] = socket
	t.mu.Unlock()
}

func (t *Terminal) Leave(socket Emitter) {
	t.mu.Lock()
	delete(t.sockets, socket.ID())
	t.mu.Unlock()
}

func (t *Terminal) OnExit(cb func(exitCode int)) {
	t.mu.Lock()
	t.callback = cb
	t.mu.Unlock()
}

func (t *Terminal) Start() {
	t.mu.Lock()
	if t.started {
		t.mu.Unlock()
		return
	}
	t.started = true
	t.mu.Unlock()

	go t.kickDisconnectedClients()
	if t.EnableKeepAlive {
		go t.keepAliveLoop()
	}
	go t.run()
}

func (t *Terminal) run() {
	cmd := buildCmd(t.file, t.args, t.cwd)

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{
		Rows: t.rows,
		Cols: t.cols,
	})
	if err != nil {
		t.fireExit(1)
		return
	}

	t.mu.Lock()
	t.ptyFile = ptmx
	t.mu.Unlock()

	buf := make([]byte, 4096)
	for {
		n, err := ptmx.Read(buf)
		if n > 0 {
			data := string(buf[:n])
			t.mu.Lock()
			t.buffer.Push(data)
			sockets := t.snapshotSockets()
			t.mu.Unlock()
			for _, s := range sockets {
				s.EmitAgent("terminalWrite", t.name, data)
			}
		}
		if err != nil {
			break
		}
	}

	exitCode := 0
	if err := cmd.Wait(); err != nil {
		var exitErr interface{ ExitCode() int }
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}
	ptmx.Close()
	t.fireExit(exitCode)
}

func (t *Terminal) fireExit(code int) {
	t.mu.Lock()
	sockets := t.snapshotSockets()
	t.sockets = make(map[string]Emitter)
	cb := t.callback
	keepAlive := t.EnableKeepAlive
	t.mu.Unlock()

	for _, s := range sockets {
		s.EmitAgent("terminalExit", t.name, code)
	}

	// For combined log terminals (EnableKeepAlive=true) keep the terminal
	// registered after exit so that clients who join shortly after can still
	// read the buffered history via GetBuffer(). The terminal will be
	// unregistered by the keepAlive loop once no clients remain, or after
	// the grace period.
	if !keepAlive {
		unregisterTerminal(t.name)
	}

	close(t.stopKicker)
	if keepAlive {
		// Start a grace-period goroutine that unregisters after 60s if no
		// client re-joins.
		go func() {
			time.Sleep(60 * time.Second)
			unregisterTerminal(t.name)
			close(t.stopKeepAlive)
		}()
	}

	select {
	case t.exitCh <- code:
	default:
	}
	if cb != nil {
		cb(code)
	}
}

func (t *Terminal) snapshotSockets() []Emitter {
	out := make([]Emitter, 0, len(t.sockets))
	for _, s := range t.sockets {
		out = append(out, s)
	}
	return out
}

func (t *Terminal) Close() {
	t.mu.Lock()
	f := t.ptyFile
	t.mu.Unlock()
	if f != nil {
		f.Write([]byte{0x03})
	}
}

func (t *Terminal) kickDisconnectedClients() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			t.mu.Lock()
			for id, s := range t.sockets {
				if !s.Connected() {
					delete(t.sockets, id)
				}
			}
			t.mu.Unlock()
		case <-t.stopKicker:
			return
		}
	}
}

func (t *Terminal) keepAliveLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			t.mu.Lock()
			n := len(t.sockets)
			t.mu.Unlock()
			if n == 0 {
				t.Close()
				return
			}
		case <-t.stopKeepAlive:
			return
		}
	}
}

// Exec creates a terminal, starts it, blocks until exit, returns exit code.
//
// The terminal is registered (and the socket subscribed) before the PTY starts,
// so that a terminalResize event from the frontend can arrive and set the correct
// column width before docker compose begins formatting its TUI output.
// A brief pause gives the frontend's FitAddon time to measure the container and
// send terminalResize with the actual browser width.
func Exec(socket Emitter, name, file string, args []string, cwd string) (int, error) {
	if GetTerminal(name) != nil {
		return -1, errors.New("another operation is already running, please try again later")
	}
	t := NewTerminal(name, file, args, cwd)
	t.rows = ProgressTerminalRows
	if socket != nil {
		t.Join(socket)
	}
	// Allow ~150 ms for the frontend's terminalResize to arrive and resize
	// the PTY to the actual browser column width before the process starts.
	// Without this, docker compose formats its TUI output for the default
	// 105-col width, and ESC[nA cursor-up sequences may not land correctly
	// if the browser is wider.
	time.Sleep(150 * time.Millisecond)
	t.Start()
	code := <-t.exitCh
	return code, nil
}

// ─── InteractiveTerminal ──────────────────────────────────────────────────────

type InteractiveTerminal struct {
	*Terminal
}

func NewInteractiveTerminal(name, file string, args []string, cwd string) *InteractiveTerminal {
	return &InteractiveTerminal{Terminal: NewTerminal(name, file, args, cwd)}
}

func (it *InteractiveTerminal) Write(input string) {
	it.mu.Lock()
	f := it.ptyFile
	it.mu.Unlock()
	if f != nil {
		f.WriteString(input)
	}
}

// ─── MainTerminal ─────────────────────────────────────────────────────────────

type MainTerminal struct {
	*InteractiveTerminal
}

func NewMainTerminal(name, stacksDir string, enableConsole bool) (*MainTerminal, error) {
	if !enableConsole {
		return nil, errors.New("console is not enabled")
	}
	shell := "bash"
	if runtime.GOOS == "windows" {
		shell = detectWindowsShell()
	}
	it := NewInteractiveTerminal(name, shell, []string{}, stacksDir)
	return &MainTerminal{InteractiveTerminal: it}, nil
}

func detectWindowsShell() string {
	if _, err := os.Stat(`C:\Program Files\PowerShell\7\pwsh.exe`); err == nil {
		return "pwsh.exe"
	}
	return "powershell.exe"
}

// ─── SocketAdapter ────────────────────────────────────────────────────────────

// SocketAdapter adapts a zishang520/socket.io v3 Socket to the Emitter interface.
type SocketAdapter struct {
	conn     *sio.Socket
	endpoint string
}

func NewSocketAdapter(conn *sio.Socket, endpoint string) *SocketAdapter {
	return &SocketAdapter{conn: conn, endpoint: endpoint}
}

func (s *SocketAdapter) ID() string { return string(s.conn.Id()) }

func (s *SocketAdapter) Connected() bool { return s.conn.Connected() }

// EmitAgent sends a server-push "agent" event to the frontend.
// Server→client wire format: emit("agent", eventName, arg1, arg2, ...)
// The endpoint is NOT a leading argument here — it differs from the
// client→server direction where the client sends emit("agent", endpoint, eventName, ...).
func (s *SocketAdapter) EmitAgent(event string, args ...any) {
	payload := append([]any{event}, args...)
	s.conn.Emit("agent", payload...)
}
