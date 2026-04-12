package handlers

import (
	"context"

	sio "github.com/zishang520/socket.io/servers/socket/v3"

	agentpkg "github.com/rahbut/dockge/backend/agent"
)

// connContext stores per-socket state: authenticated username + agent manager.
type connContext struct {
	Username string
	Manager  *agentpkg.Manager
}

func RegisterAgentHandlers(socket *sio.Socket, srv *Server) {
	socket.On("addAgent", func(args ...any) {
		ack := extractAck(args)
		if err := requireLogin(socket); err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		req, _ := mapArg(args, 0)
		agentURL, _ := req["url"].(string)
		username, _ := req["username"].(string)
		password, _ := req["password"].(string)

		ctx := context.Background()
		mgr := getOrCreateManager(socket, srv)
		if err := mgr.Test(agentURL, username, password); err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		if err := mgr.Add(ctx, agentURL, username, password); err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		mgr.SendAgentList(ctx)
		if ack != nil {
			ack(map[string]any{"ok": true, "msg": "agentAddedSuccessfully", "msgi18n": true})
		}
	})

	socket.On("removeAgent", func(args ...any) {
		ack := extractAck(args)
		if err := requireLogin(socket); err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		agentURL, _ := strArg(args, 0)
		ctx := context.Background()
		mgr := getOrCreateManager(socket, srv)
		if err := mgr.Remove(ctx, agentURL); err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		if ack != nil {
			ack(map[string]any{"ok": true, "msg": "agentRemovedSuccessfully", "msgi18n": true})
		}
	})
}

// getOrCreateManager retrieves or creates the AgentManager for this connection.
func getOrCreateManager(socket *sio.Socket, srv *Server) *agentpkg.Manager {
	if cc, ok := socket.Data().(*connContext); ok && cc.Manager != nil {
		return cc.Manager
	}

	upstream := &connUpstream{socket: socket}
	mgr := agentpkg.New(upstream, srv.IsDev)

	username := ""
	if cc, ok := socket.Data().(*connContext); ok {
		username = cc.Username
	} else if s, ok := socket.Data().(string); ok {
		username = s
	}

	socket.SetData(&connContext{Username: username, Manager: mgr})
	go mgr.ConnectAll(context.Background())
	return mgr
}

// connUpstream wraps a *sio.Socket as the agent.Upstream interface.
type connUpstream struct {
	socket *sio.Socket
}

func (u *connUpstream) Emit(event string, args ...any) {
	u.socket.Emit(event, args...)
}
