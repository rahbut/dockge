// Package agent manages outbound Socket.io v4 connections to remote Dockge
// instances (agents). Since go-socket.io v1.7.0 is server-only, we implement
// the client side using gorilla/websocket with the Socket.io wire protocol.
//
// Socket.io v4 wire protocol over WebSocket (simplified):
//
//	Engine.io packet:  "4" + socketio_payload
//	Socket.io message: "42" + json_array   (event = element 0, args = rest)
//	Socket.io connect: "40"
//	Socket.io ack:     "43" + id + json_array
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/mod/semver"

	"github.com/rahbut/dockge/backend/models"
)

const minSupportedVersion = "v1.4.0"

// Upstream is the interface the agent manager uses to push events back to the
// browser client that triggered the connection.
type Upstream interface {
	Emit(event string, args ...interface{})
}

// agentConn is a single outbound Socket.io connection to a remote Dockge instance.
type agentConn struct {
	mu         sync.Mutex
	ws         *websocket.Conn
	loggedIn   bool
	ackID      uint64
	pendingAck map[uint64]func([]interface{})
}

// Manager manages outbound connections to remote Dockge instances.
type Manager struct {
	mu             sync.Mutex
	upstream       Upstream
	conns          map[string]*agentConn // keyed by endpoint (host:port)
	firstConnectAt time.Time
	isDev          bool
}

// New creates a Manager bound to the given upstream socket.
func New(upstream Upstream, isDev bool) *Manager {
	return &Manager{
		upstream:       upstream,
		conns:          make(map[string]*agentConn),
		firstConnectAt: time.Now(),
		isDev:          isDev,
	}
}

// endpointOf extracts host:port from a URL string.
func endpointOf(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("invalid Dockge URL: %q", rawURL)
	}
	return u.Host, nil
}

// ─── Connection management ────────────────────────────────────────────────────

// Connect opens a socket.io connection to the remote agent at rawURL.
func (m *Manager) Connect(rawURL, username, password string) error {
	endpoint, err := endpointOf(rawURL)
	if err != nil {
		return err
	}

	m.mu.Lock()
	if _, ok := m.conns[endpoint]; ok {
		m.mu.Unlock()
		return nil
	}
	m.conns[endpoint] = &agentConn{pendingAck: make(map[uint64]func([]interface{}))}
	m.mu.Unlock()

	m.upstream.Emit("agentStatus", map[string]interface{}{
		"endpoint": endpoint,
		"status":   "connecting",
	})

	go m.connectLoop(endpoint, rawURL, username, password)
	return nil
}

// connectLoop establishes and maintains the WebSocket/Socket.io connection.
func (m *Manager) connectLoop(endpoint, rawURL, username, password string) {
	// Convert http(s) URL → ws(s) URL for the Socket.io endpoint.
	wsURL := socketioWsURL(rawURL)

	dialer := websocket.Dialer{HandshakeTimeout: 15 * time.Second}
	headers := http.Header{"endpoint": {endpoint}}

	ws, _, err := dialer.Dial(wsURL, headers)
	if err != nil {
		m.upstream.Emit("agentStatus", map[string]interface{}{
			"endpoint": endpoint, "status": "offline",
		})
		return
	}

	m.mu.Lock()
	ac := m.conns[endpoint]
	if ac == nil {
		m.mu.Unlock()
		ws.Close()
		return
	}
	ac.ws = ws
	m.mu.Unlock()

	// Goroutine to read incoming messages.
	go m.readLoop(endpoint, ac, username, password)
}

// socketioWsURL converts an http(s) Dockge URL to a Socket.io WebSocket URL.
func socketioWsURL(rawURL string) string {
	u, _ := url.Parse(rawURL)
	scheme := "ws"
	if u.Scheme == "https" {
		scheme = "wss"
	}
	return fmt.Sprintf("%s://%s/socket.io/?EIO=4&transport=websocket", scheme, u.Host)
}

// readLoop processes incoming Socket.io frames.
func (m *Manager) readLoop(endpoint string, ac *agentConn, username, password string) {
	defer func() {
		ac.mu.Lock()
		ws := ac.ws
		ac.loggedIn = false
		ac.mu.Unlock()
		if ws != nil {
			ws.Close()
		}
		m.upstream.Emit("agentStatus", map[string]interface{}{
			"endpoint": endpoint, "status": "offline",
		})
	}()

	for {
		_, msg, err := ac.ws.ReadMessage()
		if err != nil {
			return
		}
		m.handleMessage(endpoint, ac, msg, username, password)
	}
}

// handleMessage processes a single Socket.io frame.
func (m *Manager) handleMessage(endpoint string, ac *agentConn, raw []byte, username, password string) {
	s := string(raw)

	// Engine.io open packet: "0{...json...}" — handshake complete, send connect.
	if strings.HasPrefix(s, "0") {
		ac.ws.WriteMessage(websocket.TextMessage, []byte("40"))
		return
	}

	// Engine.io ping: "2" → pong: "3"
	if s == "2" {
		ac.ws.WriteMessage(websocket.TextMessage, []byte("3"))
		return
	}

	// Socket.io connect ack: "40" → we are connected, now login.
	if s == "40" {
		m.emitEvent(ac, "login", map[string]interface{}{
			"username": username,
			"password": password,
		}, func(args []interface{}) {
			ok := false
			if len(args) > 0 {
				if res, ok2 := args[0].(map[string]interface{}); ok2 {
					ok, _ = res["ok"].(bool)
				}
			}
			ac.mu.Lock()
			ac.loggedIn = ok
			ac.mu.Unlock()
			status := "offline"
			if ok {
				status = "online"
			}
			m.upstream.Emit("agentStatus", map[string]interface{}{
				"endpoint": endpoint, "status": status,
			})
		})
		return
	}

	// Socket.io ack: "43<id>[...]"
	if strings.HasPrefix(s, "43") {
		rest := s[2:]
		idEnd := 0
		for idEnd < len(rest) && rest[idEnd] >= '0' && rest[idEnd] <= '9' {
			idEnd++
		}
		if idEnd > 0 {
			var ackID uint64
			fmt.Sscanf(rest[:idEnd], "%d", &ackID)
			var payload []interface{}
			json.Unmarshal([]byte(rest[idEnd:]), &payload)
			ac.mu.Lock()
			cb := ac.pendingAck[ackID]
			delete(ac.pendingAck, ackID)
			ac.mu.Unlock()
			if cb != nil {
				cb(payload)
			}
		}
		return
	}

	// Socket.io event: "42[...]"
	if strings.HasPrefix(s, "42") {
		var parts []interface{}
		if err := json.Unmarshal([]byte(s[2:]), &parts); err != nil || len(parts) < 1 {
			return
		}
		event, _ := parts[0].(string)
		args := parts[1:]

		switch event {
		case "info":
			if !m.isDev && len(args) > 0 {
				if res, ok := args[0].(map[string]interface{}); ok {
					ver, _ := res["version"].(string)
					if ver != "" && !strings.HasPrefix(ver, "v") {
						ver = "v" + ver
					}
					if semver.Compare(ver, minSupportedVersion) < 0 {
						m.upstream.Emit("agentStatus", map[string]interface{}{
							"endpoint": endpoint, "status": "offline",
							"msg": fmt.Sprintf("%s: Unsupported version: %s", endpoint, ver),
						})
						ac.ws.Close()
						return
					}
				}
			}
		case "agent":
			// Proxy agent events upstream.
			m.upstream.Emit("agent", args...)
		}
		return
	}
}

// emitEvent sends a Socket.io event with an optional ack callback.
func (m *Manager) emitEvent(ac *agentConn, event string, payload interface{}, ack func([]interface{})) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if ac.ws == nil {
		return
	}

	var msg string
	if ack != nil {
		id := atomic.AddUint64(&ac.ackID, 1)
		ac.pendingAck[id] = ack
		data := []interface{}{event, payload}
		b, _ := json.Marshal(data)
		msg = fmt.Sprintf("421%d%s", id, string(b))
	} else {
		data := []interface{}{event, payload}
		b, _ := json.Marshal(data)
		msg = "42" + string(b)
	}
	ac.ws.WriteMessage(websocket.TextMessage, []byte(msg))
}

// Disconnect closes the connection to the given endpoint.
func (m *Manager) Disconnect(endpoint string) {
	m.mu.Lock()
	ac, ok := m.conns[endpoint]
	m.mu.Unlock()
	if ok && ac != nil {
		ac.mu.Lock()
		ws := ac.ws
		ac.mu.Unlock()
		if ws != nil {
			ws.Close()
		}
	}
}

// DisconnectAll closes all outbound connections.
func (m *Manager) DisconnectAll() {
	m.mu.Lock()
	endpoints := make([]string, 0, len(m.conns))
	for ep := range m.conns {
		endpoints = append(endpoints, ep)
	}
	m.mu.Unlock()
	for _, ep := range endpoints {
		m.Disconnect(ep)
	}
}

// ConnectAll loads agents from the database and connects to each.
func (m *Manager) ConnectAll(ctx context.Context) error {
	m.firstConnectAt = time.Now()
	agents, err := models.GetAgentList(ctx)
	if err != nil {
		return err
	}
	for _, a := range agents {
		_ = m.Connect(a.URL, a.Username, a.Password)
	}
	return nil
}

// EmitToEndpoint forwards an event to a specific remote agent.
// Retries for up to 10 seconds to allow login to complete.
func (m *Manager) EmitToEndpoint(endpoint, event string, args ...interface{}) error {
	deadline := m.firstConnectAt.Add(10 * time.Second)

	for {
		m.mu.Lock()
		ac, ok := m.conns[endpoint]
		m.mu.Unlock()

		if !ok {
			return fmt.Errorf("no client for endpoint %q", endpoint)
		}

		ac.mu.Lock()
		connected := ac.ws != nil && ac.loggedIn
		ac.mu.Unlock()

		if connected {
			data := append([]interface{}{endpoint, event}, args...)
			b, _ := json.Marshal(append([]interface{}{"agent"}, data...))
			msg := "42" + string(b)
			ac.mu.Lock()
			err := ac.ws.WriteMessage(websocket.TextMessage, []byte(msg))
			ac.mu.Unlock()
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("socket client not connected for endpoint %q", endpoint)
		}
		time.Sleep(time.Second)
	}
}

// EmitToAllEndpoints forwards an event to all agents.
func (m *Manager) EmitToAllEndpoints(event string, args ...interface{}) {
	m.mu.Lock()
	endpoints := make([]string, 0, len(m.conns))
	for ep := range m.conns {
		endpoints = append(endpoints, ep)
	}
	m.mu.Unlock()
	for _, ep := range endpoints {
		_ = m.EmitToEndpoint(ep, event, args...)
	}
}

// SendAgentList emits the current agent list to the upstream browser socket.
func (m *Manager) SendAgentList(ctx context.Context) error {
	list, err := models.GetAgentList(ctx)
	if err != nil {
		return err
	}
	result := map[string]interface{}{
		"": map[string]interface{}{"url": "", "username": "", "endpoint": ""},
	}
	for ep, a := range list {
		result[ep] = a.ToJSON()
	}
	m.upstream.Emit("agentList", map[string]interface{}{
		"ok": true, "agentList": result,
	})
	return nil
}

// Add persists a new agent and connects.
func (m *Manager) Add(ctx context.Context, rawURL, username, password string) error {
	if _, err := models.AddAgent(ctx, rawURL, username, password); err != nil {
		return err
	}
	return m.Connect(rawURL, username, password)
}

// Remove disconnects and deletes an agent.
func (m *Manager) Remove(ctx context.Context, rawURL string) error {
	endpoint, err := endpointOf(rawURL)
	if err != nil {
		return err
	}
	if _, err := models.RemoveAgent(ctx, rawURL); err != nil {
		return err
	}
	m.Disconnect(endpoint)
	m.mu.Lock()
	delete(m.conns, endpoint)
	m.mu.Unlock()
	return m.SendAgentList(ctx)
}

// Test temporarily connects to verify credentials without persisting.
func (m *Manager) Test(rawURL, username, password string) error {
	wsURL := socketioWsURL(rawURL)
	endpoint, _ := endpointOf(rawURL)

	dialer := websocket.Dialer{HandshakeTimeout: 15 * time.Second}
	headers := http.Header{"endpoint": {endpoint}}
	ws, _, err := dialer.Dial(wsURL, headers)
	if err != nil {
		return fmt.Errorf("unable to connect to the Dockge instance")
	}
	defer ws.Close()

	// Simple blocking handshake for the test.
	result := make(chan error, 1)
	connected := false

	ws.SetReadDeadline(time.Now().Add(15 * time.Second))
	for {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			result <- fmt.Errorf("connection error: %w", err)
			break
		}
		s := string(msg)

		if strings.HasPrefix(s, "0") {
			ws.WriteMessage(websocket.TextMessage, []byte("40"))
			continue
		}
		if s == "2" {
			ws.WriteMessage(websocket.TextMessage, []byte("3"))
			continue
		}
		if s == "40" && !connected {
			connected = true
			data, _ := json.Marshal([]interface{}{"login", map[string]interface{}{
				"username": username, "password": password,
			}})
			ws.WriteMessage(websocket.TextMessage, []byte("421"+string(data)))
			continue
		}
		if strings.HasPrefix(s, "43") {
			var payload []interface{}
			json.Unmarshal([]byte(s[3:]), &payload) // strip "431"
			if len(payload) > 0 {
				if res, ok := payload[0].(map[string]interface{}); ok {
					if ok2, _ := res["ok"].(bool); ok2 {
						result <- nil
					} else {
						msg2, _ := res["msg"].(string)
						result <- fmt.Errorf("%s", msg2)
					}
				}
			}
			break
		}
	}

	select {
	case err := <-result:
		return err
	default:
		return fmt.Errorf("unexpected test flow")
	}
}
