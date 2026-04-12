package handlers

import (
	sio "github.com/zishang520/socket.io/servers/socket/v3"

	"github.com/rahbut/dockge/backend/stack"
	"github.com/rahbut/dockge/backend/terminal"
)

func RegisterTerminalHandlers(socket *sio.Socket, srv *Server) {
	socket.On("terminalInput", func(args ...any) {
		ack := extractAck(args)
		if err := requireLogin(socket); err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		termName, _ := strArg(args, 0)
		cmd, _ := strArg(args, 1)
		t := terminal.GetTerminal(termName)
		if t == nil {
			if ack != nil {
				ack(errResp("terminal not found"))
			}
			return
		}
		type writer interface{ Write(string) }
		if wt, ok := any(t).(writer); ok {
			wt.Write(cmd)
		}
		if ack != nil {
			ack(okResp())
		}
	})

	socket.On("mainTerminal", func(args ...any) {
		ack := extractAck(args)
		if err := requireLogin(socket); err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		termName, _ := strArg(args, 0)
		adapter := terminal.NewSocketAdapter(socket, "")
		t := terminal.GetTerminal(termName)
		if t == nil {
			mt, err := terminal.NewMainTerminal(termName, srv.StacksDir, srv.EnableConsole)
			if err != nil {
				if ack != nil {
					ack(errResp(err.Error()))
				}
				return
			}
			mt.Join(adapter)
			mt.Start()
		} else {
			t.Join(adapter)
		}
		if ack != nil {
			ack(okResp())
		}
	})

	socket.On("checkMainTerminal", func(args ...any) {
		ack := extractAck(args)
		if err := requireLogin(socket); err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		if ack != nil {
			ack(map[string]any{"ok": srv.EnableConsole})
		}
	})

	socket.On("interactiveTerminal", func(args ...any) {
		ack := extractAck(args)
		if err := requireLogin(socket); err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		stackName, _ := strArg(args, 0)
		serviceName, _ := strArg(args, 1)
		shellName, _ := strArg(args, 2)
		st, err := stack.GetStack(srv.StacksDir, stackName, false)
		if err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		st.JoinContainerTerminal(terminal.NewSocketAdapter(socket, ""), serviceName, shellName, 0)
		if ack != nil {
			ack(okResp())
		}
	})

	socket.On("terminalJoin", func(args ...any) {
		ack := extractAck(args)
		if err := requireLogin(socket); err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		termName, _ := strArg(args, 0)
		t := terminal.GetTerminal(termName)
		adapter := terminal.NewSocketAdapter(socket, "")
		if t == nil {
			if ack != nil {
				ack(map[string]any{"ok": true, "buffer": ""})
			}
			return
		}
		t.Join(adapter)
		if ack != nil {
			ack(map[string]any{"ok": true, "buffer": t.GetBuffer()})
		}
	})

	socket.On("leaveCombinedTerminal", func(args ...any) {
		ack := extractAck(args)
		if err := requireLogin(socket); err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		stackName, _ := strArg(args, 0)
		st, err := stack.GetStack(srv.StacksDir, stackName, true)
		if err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		st.LeaveCombinedTerminal(terminal.NewSocketAdapter(socket, ""))
		if ack != nil {
			ack(okResp())
		}
	})

	socket.On("terminalResize", func(args ...any) {
		termName, _ := strArg(args, 0)
		var rows, cols uint16
		if v, ok := args[1].(float64); ok {
			rows = uint16(v)
		}
		if v, ok := args[2].(float64); ok {
			cols = uint16(v)
		}
		if t := terminal.GetTerminal(termName); t != nil {
			t.Resize(rows, cols)
		}
	})
}
