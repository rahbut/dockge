package handlers

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	sio "github.com/zishang520/socket.io/servers/socket/v3"

	"github.com/rahbut/dockge/backend/models"
	"github.com/rahbut/dockge/backend/stack"
	"github.com/rahbut/dockge/backend/terminal"
)

// RegisterDockerHandlers registers stack/docker socket events directly on the
// socket (in addition to the "agent" proxy handler which handles the same events
// when they arrive via the agent envelope).
func RegisterDockerHandlers(socket *sio.Socket, srv *Server) {

	socket.On("requestStackList", func(args ...any) {
		ack := extractAck(args)
		if err := requireLogin(socket); err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		srv.BroadcastStackList()
		if ack != nil {
			ack(map[string]any{"ok": true, "msg": "scanFolder", "msgi18n": true})
		}
	})

	socket.On("getStack", func(args ...any) {
		ack := extractAck(args)
		if err := requireLogin(socket); err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		stackName, _ := strArg(args, 0)
		st, err := stack.GetStack(srv.StacksDir, stackName, false)
		if err != nil {
			if ack != nil {
				ack(map[string]any{
					"ok": true,
					"stack": map[string]any{
						"name":              stackName,
						"isManagedByDockge": false,
						"composeYAML":       "",
						"composeENV":        "",
						"status":            0,
						"endpoint":          "",
					},
				})
			}
			return
		}
		st.Load()
		obj, err := st.ToJSON(context.Background(), "")
		if err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		st.JoinCombinedTerminal(terminal.NewSocketAdapter(socket, ""))
		if ack != nil {
			ack(map[string]any{"ok": true, "stack": obj})
		}
	})

	socket.On("deployStack", func(args ...any) {
		ack := extractAck(args)
		if err := requireLogin(socket); err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		name, _ := strArg(args, 0)
		composeYAML, _ := strArg(args, 1)
		composeENV, _ := strArg(args, 2)
		isAdd := boolArg(args, 3)
		go func() {
			st := stack.New(srv.StacksDir, name, composeYAML, composeENV, false)
			if err := st.Save(isAdd); err != nil {
				if ack != nil {
					ack(errResp(err.Error()))
				}
				return
			}
			if err := st.Deploy(terminal.NewSocketAdapter(socket, "")); err != nil {
				if ack != nil {
					ack(errResp(err.Error()))
				}
				return
			}
			if ack != nil {
				ack(map[string]any{"ok": true, "msg": "Deployed", "msgi18n": true})
			}
		}()
	})

	socket.On("saveStack", func(args ...any) {
		ack := extractAck(args)
		if err := requireLogin(socket); err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		name, _ := strArg(args, 0)
		composeYAML, _ := strArg(args, 1)
		composeENV, _ := strArg(args, 2)
		isAdd := boolArg(args, 3)
		st := stack.New(srv.StacksDir, name, composeYAML, composeENV, false)
		if err := st.Save(isAdd); err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		if ack != nil {
			ack(map[string]any{"ok": true, "msg": "Saved", "msgi18n": true})
		}
	})

	socket.On("deleteStack", func(args ...any) {
		ack := extractAck(args)
		if err := requireLogin(socket); err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		name, _ := strArg(args, 0)
		go func() {
			st, err := stack.GetStack(srv.StacksDir, name, false)
			if err != nil {
				if ack != nil {
					ack(errResp(err.Error()))
				}
				return
			}
			if err := st.Delete(terminal.NewSocketAdapter(socket, "")); err != nil {
				if ack != nil {
					ack(errResp(err.Error()))
				}
				return
			}
			if ack != nil {
				ack(map[string]any{"ok": true, "msg": "Deleted", "msgi18n": true})
			}
		}()
	})

	socket.On("startStack", makeStackOpHandler(socket, srv, "Started", func(st *stack.Stack, adapter terminal.Emitter) error {
		return st.Start(adapter)
	}))
	socket.On("stopStack", makeStackOpHandler(socket, srv, "Stopped", func(st *stack.Stack, adapter terminal.Emitter) error {
		return st.Stop(adapter)
	}))
	socket.On("restartStack", makeStackOpHandler(socket, srv, "Restarted", func(st *stack.Stack, adapter terminal.Emitter) error {
		return st.Restart(adapter)
	}))
	socket.On("downStack", makeStackOpHandler(socket, srv, "Stopped", func(st *stack.Stack, adapter terminal.Emitter) error {
		return st.Down(adapter)
	}))

	socket.On("updateStack", func(args ...any) {
		ack := extractAck(args)
		if err := requireLogin(socket); err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		name, _ := strArg(args, 0)
		go func() {
			st, err := stack.GetStack(srv.StacksDir, name, false)
			if err != nil {
				if ack != nil {
					ack(errResp(err.Error()))
				}
				return
			}
			st.Load()

			// Self-update path: when updating the stack that dockge itself is
			// running inside, docker compose up -d will stop this process.
			// Pull first (blocking, with terminal output), clear the update
			// badge, send the ack, then spawn the detached up -d. The socket.io
			// client will reconnect automatically once the new container is up.
			if name == stack.SelfStackName() {
				if err := st.SelfUpdate(terminal.NewSocketAdapter(socket, "")); err != nil {
					if ack != nil {
						ack(errResp(err.Error()))
					}
					return
				}
				if err := models.ClearStackUpdateResult(context.Background(), name); err != nil {
					log.Warn().Err(err).Str("stack", name).Msg("Failed to clear update result cache")
				}
				if ack != nil {
					ack(map[string]any{"ok": true, "msg": "Updated", "msgi18n": true})
				}
				// Small delay to allow the ack to reach the client before
				// the container is replaced.
				time.Sleep(500 * time.Millisecond)
				return
			}

			if err := st.Update(terminal.NewSocketAdapter(socket, "")); err != nil {
				if ack != nil {
					ack(errResp(err.Error()))
				}
				return
			}
			// Clear the persisted update badge so the next BroadcastStackList
			// call does not re-overlay the stale "updateAvailable: true" value
			// from the DB cache.
			if err := models.ClearStackUpdateResult(context.Background(), name); err != nil {
				log.Warn().Err(err).Str("stack", name).Msg("Failed to clear update result cache")
			}
			go srv.BroadcastStackListAfter(500 * time.Millisecond)
			if ack != nil {
				ack(map[string]any{"ok": true, "msg": "Updated", "msgi18n": true})
			}
		}()
	})

	socket.On("serviceStatusList", func(args ...any) {
		ack := extractAck(args)
		if err := requireLogin(socket); err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		name, _ := strArg(args, 0)
		st, err := stack.GetStack(srv.StacksDir, name, false)
		if err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		statuses, err := st.GetServiceStatusList()
		if err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		out := make(map[string]any, len(statuses))
		for svcName, ss := range statuses {
			out[svcName] = map[string]any{"state": ss.State, "ports": ss.Ports}
		}
		if ack != nil {
			ack(map[string]any{"ok": true, "serviceStatusList": out})
		}
	})

	socket.On("checkAllStacksUpdates", func(args ...any) {
		ack := extractAck(args)
		if err := requireLogin(socket); err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		go func() {
			allResults, err := srv.RunUpdateCheck()
			if err != nil {
				if ack != nil {
					ack(errResp(err.Error()))
				}
				return
			}
			// Persist so late-connecting clients and future scheduled checks
			// can use these results (same as the scheduled check does).
			ctx := context.Background()
			if perr := models.SetLastUpdateResults(ctx, allResults); perr != nil {
				log.Warn().Err(perr).Msg("checkAllStacksUpdates: failed to persist results")
			}
			if ack != nil {
				ack(map[string]any{"ok": true, "allResults": allResults})
			}
		}()
	})

	socket.On("pruneImages", func(args ...any) {
		ack := extractAck(args)
		if err := requireLogin(socket); err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		aggressive := boolArg(args, 0)
		go func() {
			pruneArgs := []string{"image", "prune", "-f"}
			if aggressive {
				pruneArgs = append(pruneArgs, "-a")
			}
			out, err := exec.Command("docker", pruneArgs...).Output()
			if err != nil {
				if ack != nil {
					ack(errResp(fmt.Sprintf("docker image prune: %v", err)))
				}
				return
			}
			if ack != nil {
				ack(parsePruneOutput(string(out)))
			}
		}()
	})

	socket.On("getDockerNetworkList", func(args ...any) {
		ack := extractAck(args)
		if err := requireLogin(socket); err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		out, err := exec.Command("docker", "network", "ls", "--format", "{{.Name}}").Output()
		if err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		var networks []string
		for _, line := range strings.Split(string(out), "\n") {
			if line = strings.TrimSpace(line); line != "" {
				networks = append(networks, line)
			}
		}
		if ack != nil {
			ack(map[string]any{"ok": true, "dockerNetworkList": networks})
		}
	})
}

// makeStackOpHandler builds an async handler for start/stop/restart/down operations.
func makeStackOpHandler(
	socket *sio.Socket,
	srv *Server,
	successMsg string,
	op func(*stack.Stack, terminal.Emitter) error,
) func(...any) {
	return func(args ...any) {
		ack := extractAck(args)
		if err := requireLogin(socket); err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		name, _ := strArg(args, 0)
		go func() {
			st, err := stack.GetStack(srv.StacksDir, name, false)
			if err != nil {
				if ack != nil {
					ack(errResp(err.Error()))
				}
				return
			}
			if err := op(st, terminal.NewSocketAdapter(socket, "")); err != nil {
				if ack != nil {
					ack(errResp(err.Error()))
				}
				return
			}
			if ack != nil {
				ack(map[string]any{"ok": true, "msg": successMsg, "msgi18n": true})
			}
		}()
	}
}
