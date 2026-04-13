package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
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

	// Cron scheduler — set by main after startup.
	Cron         *cron.Cron
	updateCronID cron.EntryID
}

// RunUpdateCheck checks every managed stack for image updates and returns a
// results map keyed by stack name. It does NOT broadcast — callers decide
// whether to ack a socket response or broadcast to all connected clients.
func (srv *Server) RunUpdateCheck() (map[string]any, error) {
	stacks, err := stack.GetStackList(srv.StacksDir, false)
	if err != nil {
		return nil, err
	}
	allResults := make(map[string]any, len(stacks))
	for name, st := range stacks {
		if !st.IsManagedByDockge() {
			continue
		}
		st.Load()
		results, _ := st.CheckUpdates()
		hasUpdate := false
		for _, r := range results {
			if r.UpdateAvailable {
				hasUpdate = true
				break
			}
		}
		allResults[name] = map[string]any{
			"updateAvailable": hasUpdate,
			"services":        results,
		}
	}
	return allResults, nil
}

// ScheduledUpdateCheck runs RunUpdateCheck and broadcasts the results to all
// connected clients so their nav badges update without a manual check.
func (srv *Server) ScheduledUpdateCheck() {
	log.Info().Msg("Scheduled update check: starting")
	allResults, err := srv.RunUpdateCheck()
	if err != nil {
		log.Error().Err(err).Msg("Scheduled update check failed")
		return
	}
	updates := 0
	for _, v := range allResults {
		if m, ok := v.(map[string]any); ok {
			if avail, ok := m["updateAvailable"].(bool); ok && avail {
				updates++
			}
		}
	}
	log.Info().Int("stacks_with_updates", updates).Msg("Scheduled update check: complete — broadcasting")
	// Broadcast the full allResults so every connected client can update badges.
	srv.SIOServer.Sockets().Emit("agent", "updateCheckResults", map[string]any{
		"ok":         true,
		"allResults": allResults,
	})
}

// SetUpdateCheckSchedule configures (or clears) the daily update-check cron
// job. timeStr should be "HH:MM" (24-hour); empty string disables the job.
func (srv *Server) SetUpdateCheckSchedule(timeStr string) {
	// Remove any existing job first.
	if srv.updateCronID != 0 {
		srv.Cron.Remove(srv.updateCronID)
		srv.updateCronID = 0
	}
	if timeStr == "" {
		log.Info().Msg("Scheduled update check: disabled")
		return
	}
	// Parse HH:MM.
	var hh, mm int
	if _, err := fmt.Sscanf(timeStr, "%d:%d", &hh, &mm); err != nil || hh < 0 || hh > 23 || mm < 0 || mm > 59 {
		log.Warn().Str("time", timeStr).Msg("Invalid updateCheckTime — scheduled check not registered")
		return
	}
	spec := fmt.Sprintf("0 %d %d * * *", mm, hh)
	id, err := srv.Cron.AddFunc(spec, srv.ScheduledUpdateCheck)
	if err != nil {
		log.Error().Err(err).Str("spec", spec).Msg("Failed to register update check cron job")
		return
	}
	srv.updateCronID = id
	log.Info().Str("time", timeStr).Str("spec", spec).Msg("Scheduled update check: registered")
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
