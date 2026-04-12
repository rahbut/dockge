package handlers

import (
	"context"
	"time"

	sio "github.com/zishang520/socket.io/servers/socket/v3"

	"github.com/rahbut/dockge/backend/models"
	"github.com/rahbut/dockge/backend/stack"
)

// Server holds application-wide state accessible from all socket handlers.
type Server struct {
	SIOServer     *sio.Server
	StacksDir     string
	DataDir       string
	Version       string
	IsDev         bool
	IsContainer   bool
	EnableConsole bool
}

// BroadcastStackListAfter broadcasts the stack list after a short delay to allow
// docker's internal state to settle after an operation completes.
// Called as a goroutine from state-changing handlers.
func (srv *Server) BroadcastStackListAfter(d time.Duration) {
	time.Sleep(d)
	srv.BroadcastStackList()
}

// BroadcastStackList builds the stack list and broadcasts it to all connected
// sockets as an "agent" event with the local endpoint ("").
// stackList is a map[stackName]stackObject — the frontend iterates with for...in.
func (srv *Server) BroadcastStackList() {
	stacks, err := stack.GetStackList(srv.StacksDir, false)
	if err != nil {
		return
	}
	list := make(map[string]any, len(stacks))
	for name, st := range stacks {
		list[name] = st.ToSimpleJSON()
	}
	// Server → client "agent" events use: emit("agent", eventName, payload)
	// The endpoint is embedded inside the payload, NOT as a leading argument.
	// (Client → server is different: emit("agent", endpoint, eventName, ...args))
	srv.SIOServer.Sockets().Emit("agent", "stackList", map[string]any{
		"ok":        true,
		"stackList": list,
		"endpoint":  "",
	})
}

// SendInfo emits server version and configuration info to a single connection.
func (srv *Server) SendInfo(socket *sio.Socket) {
	ctx := context.Background()
	primary, _ := models.GetPrimaryHostname(ctx)
	socket.Emit("info", map[string]any{
		"version":         srv.Version,
		"isContainer":     srv.IsContainer,
		"primaryHostname": primary,
	})
}
