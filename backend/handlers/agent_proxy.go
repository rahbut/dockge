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

const allEndpoints = "##ALL_DOCKGE_ENDPOINTS##"

// RegisterAgentProxyHandler registers the "agent" event envelope handler.
// The frontend wraps every stack/terminal/docker call as:
//
//	socket.emit("agent", endpoint, eventName, ...args)
//
// endpoint == ""              → handle locally
// endpoint == ALL_ENDPOINTS   → handle locally + forward to all remote agents
// endpoint == "host:port"     → forward to that specific remote agent only
func RegisterAgentProxyHandler(socket *sio.Socket, srv *Server) {
	socket.On("agent", func(args ...any) {
		if len(args) < 2 {
			return
		}
		endpoint, _ := args[0].(string)
		eventName, _ := args[1].(string)
		innerArgs := args[2:]

		handleLocal := endpoint == "" || endpoint == allEndpoints
		proxyRemote := endpoint != ""

		// IMPORTANT: some events (deployStack, startStack, stopStack, etc.) block
		// for the duration of the docker compose operation — potentially minutes.
		// zishang520/socket.io runs event handlers through a sequential task queue,
		// so blocking here would freeze the socket (no pings, no other events).
		// We always dispatch into a goroutine and call the ack when done.
		ack := extractAck(innerArgs)

		go func() {
			if proxyRemote {
				mgr := getOrCreateManager(socket, srv)
				if endpoint == allEndpoints {
					mgr.EmitToAllEndpoints(eventName, innerArgs...)
				} else {
					if err := mgr.EmitToEndpoint(endpoint, eventName, innerArgs...); err != nil {
						if ack != nil {
							ack(errResp(fmt.Sprintf("agent proxy: %v", err)))
						}
						return
					}
				}
			}

			if handleLocal {
				result := dispatchLocalAgentEvent(socket, srv, eventName, innerArgs...)
				if result != nil && ack != nil {
					ack(result)
				}
			}
		}()
	})
}

func dispatchLocalAgentEvent(socket *sio.Socket, srv *Server, eventName string, args ...any) any {
	if err := requireLogin(socket); err != nil {
		return errResp(err.Error())
	}

	ctx := context.Background()

	switch eventName {

	// ── Stack operations ──────────────────────────────────────────────────

	case "deployStack":
		name, _ := strArg(args, 0)
		composeYAML, _ := strArg(args, 1)
		composeENV, _ := strArg(args, 2)
		isAdd := boolArg(args, 3)
		st := stack.New(srv.StacksDir, name, composeYAML, composeENV, false)
		if err := st.Save(isAdd); err != nil {
			return errResp(err.Error())
		}
		if err := st.Deploy(terminal.NewSocketAdapter(socket, "")); err != nil {
			return errResp(err.Error())
		}
		go srv.BroadcastStackListAfter(500 * time.Millisecond)
		return map[string]any{"ok": true, "msg": "Deployed", "msgi18n": true}

	case "saveStack":
		name, _ := strArg(args, 0)
		composeYAML, _ := strArg(args, 1)
		composeENV, _ := strArg(args, 2)
		isAdd := boolArg(args, 3)
		st := stack.New(srv.StacksDir, name, composeYAML, composeENV, false)
		if err := st.Save(isAdd); err != nil {
			return errResp(err.Error())
		}
		return map[string]any{"ok": true, "msg": "Saved", "msgi18n": true}

	case "deleteStack":
		name, _ := strArg(args, 0)
		st, err := stack.GetStack(srv.StacksDir, name, false)
		if err != nil {
			return errResp(err.Error())
		}
		if err := st.Delete(terminal.NewSocketAdapter(socket, "")); err != nil {
			return errResp(err.Error())
		}
		go srv.BroadcastStackListAfter(500 * time.Millisecond)
		return map[string]any{"ok": true, "msg": "Deleted", "msgi18n": true}

	case "getStack":
		name, _ := strArg(args, 0)
		st, err := stack.GetStack(srv.StacksDir, name, false)
		if err != nil {
			// Stack exists in docker but has no directory in the stacks folder —
			// return a minimal object with isManagedByDockge:false so the frontend
			// renders the "not managed by Dockge" message instead of a sticky error toast.
			return map[string]any{
				"ok": true,
				"stack": map[string]any{
					"name":              name,
					"isManagedByDockge": false,
					"composeYAML":       "",
					"composeENV":        "",
					"status":            0,
					"endpoint":          "",
				},
			}
		}
		st.Load()
		obj, err := st.ToJSON(ctx, "")
		if err != nil {
			return errResp(err.Error())
		}
		st.JoinCombinedTerminal(terminal.NewSocketAdapter(socket, ""))
		return map[string]any{"ok": true, "stack": obj}

	case "requestStackList":
		srv.BroadcastStackList()
		return map[string]any{"ok": true, "msg": "scanFolder", "msgi18n": true}

	case "startStack":
		name, _ := strArg(args, 0)
		st, err := stack.GetStack(srv.StacksDir, name, false)
		if err != nil {
			return errResp(err.Error())
		}
		if err := st.Start(terminal.NewSocketAdapter(socket, "")); err != nil {
			return errResp(err.Error())
		}
		go srv.BroadcastStackListAfter(500 * time.Millisecond)
		return map[string]any{"ok": true, "msg": "Started", "msgi18n": true}

	case "stopStack":
		name, _ := strArg(args, 0)
		st, err := stack.GetStack(srv.StacksDir, name, false)
		if err != nil {
			return errResp(err.Error())
		}
		if err := st.Stop(terminal.NewSocketAdapter(socket, "")); err != nil {
			return errResp(err.Error())
		}
		go srv.BroadcastStackListAfter(500 * time.Millisecond)
		return map[string]any{"ok": true, "msg": "Stopped", "msgi18n": true}

	case "restartStack":
		name, _ := strArg(args, 0)
		st, err := stack.GetStack(srv.StacksDir, name, false)
		if err != nil {
			return errResp(err.Error())
		}
		if err := st.Restart(terminal.NewSocketAdapter(socket, "")); err != nil {
			return errResp(err.Error())
		}
		go srv.BroadcastStackListAfter(500 * time.Millisecond)
		return map[string]any{"ok": true, "msg": "Restarted", "msgi18n": true}

	case "updateStack":
		name, _ := strArg(args, 0)
		st, err := stack.GetStack(srv.StacksDir, name, false)
		if err != nil {
			return errResp(err.Error())
		}
		st.Load()

		// Self-update path: pull blocks with terminal output, then the
		// sibling container will replace this container. Ack before returning.
		if name == stack.SelfStackName() {
			if err := st.SelfUpdate(terminal.NewSocketAdapter(socket, "")); err != nil {
				return errResp(err.Error())
			}
			if err := models.ClearStackUpdateResult(context.Background(), name); err != nil {
				log.Warn().Err(err).Str("stack", name).Msg("Failed to clear update result cache")
			}
			// Tell all clients to reload on their next reconnect.
			srv.BroadcastReloadOnReconnect()
			return map[string]any{"ok": true, "msg": "Updated", "msgi18n": true}
		}

		if err := st.Update(terminal.NewSocketAdapter(socket, "")); err != nil {
			return errResp(err.Error())
		}
		// Clear the persisted update badge so the next BroadcastStackList
		// call does not re-overlay the stale "updateAvailable: true" value
		// from the DB cache.
		if err := models.ClearStackUpdateResult(context.Background(), name); err != nil {
			log.Warn().Err(err).Str("stack", name).Msg("Failed to clear update result cache")
		}
		go srv.BroadcastStackListAfter(500 * time.Millisecond)
		return map[string]any{"ok": true, "msg": "Updated", "msgi18n": true}

	case "downStack":
		name, _ := strArg(args, 0)
		st, err := stack.GetStack(srv.StacksDir, name, false)
		if err != nil {
			return errResp(err.Error())
		}
		if err := st.Down(terminal.NewSocketAdapter(socket, "")); err != nil {
			return errResp(err.Error())
		}
		go srv.BroadcastStackListAfter(500 * time.Millisecond)
		return map[string]any{"ok": true, "msg": "Stopped", "msgi18n": true}

	case "serviceStatusList":
		name, _ := strArg(args, 0)
		st, err := stack.GetStack(srv.StacksDir, name, false)
		if err != nil {
			return errResp(err.Error())
		}
		statuses, err := st.GetServiceStatusList()
		if err != nil {
			return errResp(err.Error())
		}
		out := make(map[string]any, len(statuses))
		for svcName, ss := range statuses {
			out[svcName] = map[string]any{"state": ss.State, "ports": ss.Ports}
		}
		return map[string]any{"ok": true, "serviceStatusList": out}

	case "checkAllStacksUpdates":
		allResults, err := srv.RunUpdateCheck()
		if err != nil {
			return errResp(err.Error())
		}
		// Persist so late-connecting clients and future scheduled checks
		// can use these results (same as the scheduled check does).
		if perr := models.SetLastUpdateResults(ctx, allResults); perr != nil {
			log.Warn().Err(perr).Msg("checkAllStacksUpdates: failed to persist results")
		}
		return map[string]any{"ok": true, "allResults": allResults}

	case "pruneImages":
		aggressive := boolArg(args, 0)
		pruneArgs := []string{"image", "prune", "-f"}
		if aggressive {
			pruneArgs = append(pruneArgs, "-a")
		}
		out, err := exec.Command("docker", pruneArgs...).Output()
		if err != nil {
			return errResp(fmt.Sprintf("docker image prune: %v", err))
		}
		return parsePruneOutput(string(out))

	case "getDockerNetworkList":
		out, err := exec.Command("docker", "network", "ls", "--format", "{{.Name}}").Output()
		if err != nil {
			return errResp(err.Error())
		}
		var networks []string
		for _, line := range strings.Split(string(out), "\n") {
			if line = strings.TrimSpace(line); line != "" {
				networks = append(networks, line)
			}
		}
		return map[string]any{"ok": true, "dockerNetworkList": networks}

	// ── Terminal operations ───────────────────────────────────────────────

	case "terminalInput":
		termName, _ := strArg(args, 0)
		cmd, _ := strArg(args, 1)
		t := terminal.GetTerminal(termName)
		if t == nil {
			return errResp("terminal not found")
		}
		type writer interface{ Write(string) }
		if wt, ok := any(t).(writer); ok {
			wt.Write(cmd)
		}
		return okResp()

	case "mainTerminal":
		termName, _ := strArg(args, 0)
		adapter := terminal.NewSocketAdapter(socket, "")
		t := terminal.GetTerminal(termName)
		if t == nil {
			mt, err := terminal.NewMainTerminal(termName, srv.StacksDir, srv.EnableConsole)
			if err != nil {
				return errResp(err.Error())
			}
			mt.Join(adapter)
			mt.Start()
		} else {
			t.Join(adapter)
		}
		return okResp()

	case "checkMainTerminal":
		return map[string]any{"ok": srv.EnableConsole}

	case "interactiveTerminal":
		stackName, _ := strArg(args, 0)
		serviceName, _ := strArg(args, 1)
		shellName, _ := strArg(args, 2)
		st, err := stack.GetStack(srv.StacksDir, stackName, false)
		if err != nil {
			return errResp(err.Error())
		}
		st.JoinContainerTerminal(terminal.NewSocketAdapter(socket, ""), serviceName, shellName, 0)
		return okResp()

	case "terminalJoin":
		termName, _ := strArg(args, 0)
		t := terminal.GetTerminal(termName)
		adapter := terminal.NewSocketAdapter(socket, "")
		if t == nil {
			// Terminal doesn't exist yet (e.g. before first deploy) — not an
			// error, just no buffer to replay. Return ok:true with empty buffer
			// to avoid an empty error toast in the frontend.
			return map[string]any{"ok": true, "buffer": ""}
		}
		t.Join(adapter)
		return map[string]any{"ok": true, "buffer": t.GetBuffer()}

	case "leaveCombinedTerminal":
		stackName, _ := strArg(args, 0)
		st, err := stack.GetStack(srv.StacksDir, stackName, true)
		if err != nil {
			return errResp(err.Error())
		}
		st.LeaveCombinedTerminal(terminal.NewSocketAdapter(socket, ""))
		return okResp()

	case "terminalResize":
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
		return nil

	default:
		return errResp(fmt.Sprintf("unknown agent event: %q", eventName))
	}
}

func parsePruneOutput(output string) map[string]any {
	var images []string
	spaceReclaimed := ""
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "untagged:") || strings.HasPrefix(line, "deleted:") {
			images = append(images, line)
		}
		if strings.HasPrefix(line, "Total reclaimed space:") {
			spaceReclaimed = strings.TrimSpace(strings.TrimPrefix(line, "Total reclaimed space:"))
		}
	}
	return map[string]any{
		"ok":             true,
		"count":          len(images),
		"spaceReclaimed": spaceReclaimed,
		"images":         images,
	}
}
