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
//
// Prior results are loaded from the DB so that a transient registry error for
// a service does not silently flip its status to "no update available". When
// GetRegistryDigest fails (Error == "registryError"), the previous known
// updateAvailable value is preserved instead of defaulting to false.
func (srv *Server) RunUpdateCheck() (map[string]any, error) {
	ctx := context.Background()

	// Load last-known results to use as fallback for registry errors.
	prior, _ := models.GetLastUpdateResults(ctx)

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
		for svcName, r := range results {
			if r.Error == "registryError" {
				// Registry was unreachable — fall back to the prior known state
				// for this service so a transient failure doesn't hide real updates.
				if p, ok := prior[name]; ok {
					if snap, ok := p.Services[svcName]; ok && snap.UpdateAvailable {
						hasUpdate = true
					}
				}
			} else if r.UpdateAvailable {
				hasUpdate = true
			}
		}

		// Build the services sub-map as map[string]any for JSON compatibility.
		services := make(map[string]any, len(results))
		for svcName, r := range results {
			services[svcName] = map[string]any{
				"updateAvailable": r.UpdateAvailable,
				"image":           r.Image,
				"error":           r.Error,
			}
		}

		allResults[name] = map[string]any{
			"updateAvailable": hasUpdate,
			"services":        services,
		}
	}
	return allResults, nil
}

// ScheduledUpdateCheck runs RunUpdateCheck, broadcasts the results to all
// connected clients so their nav badges update without a manual check, and
// persists the results to the DB so late-connecting clients also see them.
func (srv *Server) ScheduledUpdateCheck() {
	ctx := context.Background()
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
	// Persist results so clients that connect after the broadcast still see
	// correct update badges via BroadcastStackList (which merges cached values).
	if err := models.SetLastUpdateResults(ctx, allResults); err != nil {
		log.Warn().Err(err).Msg("Scheduled update check: failed to persist results")
	}
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
	log.Info().Str("time", timeStr).Str("spec", spec).Str("tz", srv.Cron.Location().String()).Msg("Scheduled update check: registered")
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
//
// Cached update-check results are merged in so that clients connecting after a
// scheduled check (e.g. opening the browser the morning after a 2am run) still
// see correct updateAvailable badges without needing to trigger a manual check.
func (srv *Server) BroadcastStackList() {
	ctx := context.Background()
	stacks, err := stack.GetStackList(srv.StacksDir, false)
	if err != nil {
		return
	}

	// Load the last-known update results from the DB.
	cached, _ := models.GetLastUpdateResults(ctx)

	list := make(map[string]any, len(stacks))
	for name, st := range stacks {
		obj := st.ToSimpleJSON()
		// Overlay cached updateAvailable when the stack struct has no live result
		// (UpdateAvailable == nil means CheckUpdates has not run for this instance).
		if obj["updateAvailable"] == nil {
			if entry, ok := cached[name]; ok {
				obj["updateAvailable"] = entry.UpdateAvailable
			}
		}
		list[name] = obj
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
		"selfStackName":   stack.SelfStackName(),
	})
}
