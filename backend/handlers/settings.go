package handlers

import (
	"context"
	"os"
	"path/filepath"

	sio "github.com/zishang520/socket.io/servers/socket/v3"

	"github.com/rahbut/dockge/backend/auth"
	"github.com/rahbut/dockge/backend/models"
	"github.com/rahbut/dockge/backend/stack"
)

func RegisterSettingsHandlers(socket *sio.Socket, srv *Server) {
	socket.On("getSettings", func(args ...any) {
		ack := extractAck(args)
		if err := requireLogin(socket); err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		ctx := context.Background()
		rawSettings, err := models.GetSettingsByType(ctx, "general")
		if err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		settings := make(map[string]any, len(rawSettings)+1)
		for k, v := range rawSettings {
			switch k {
			case "disableAuth", "trustProxy":
				settings[k] = v == "1" || v == "true"
			default:
				settings[k] = v
			}
		}
		globalEnv := ""
		if b, err := os.ReadFile(filepath.Join(srv.StacksDir, "global.env")); err == nil {
			globalEnv = string(b)
		}
		settings["globalENV"] = globalEnv
		if ack != nil {
			ack(map[string]any{"ok": true, "data": settings})
		}
	})

	socket.On("setSettings", func(args ...any) {
		ack := extractAck(args)
		if err := requireLogin(socket); err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		data, _ := mapArg(args, 0)
		currentPassword, _ := strArg(args, 1)

		ctx := context.Background()

		// Only validate the current password when it is explicitly provided —
		// this only happens when toggling disableAuth. Normal settings saves
		// send currentPassword as undefined/empty and must not require it.
		if currentPassword != "" {
			username := socketUsername(socket)
			user, err := models.GetUserByUsername(ctx, username)
			if err != nil {
				if ack != nil {
					ack(errResp("User not found"))
				}
				return
			}
			if !auth.VerifyPassword(user.Password.String, currentPassword) {
				if ack != nil {
					ack(errResp("Incorrect password"))
				}
				return
			}
		}
		for key, val := range data {
			if key == "globalENV" || key == "globalEnv" {
				continue
			}
			var strVal string
			switch v := val.(type) {
			case bool:
				if v {
					strVal = "1"
				} else {
					strVal = "0"
				}
			case string:
				strVal = v
			default:
				continue
			}
			_ = models.SetSetting(ctx, key, strVal, "general")
		}
		globalEnvPath := filepath.Join(srv.StacksDir, "global.env")
		globalEnvVal, ok := data["globalENV"].(string)
		if !ok {
			globalEnvVal, ok = data["globalEnv"].(string)
		}
		if ok {
			if globalEnvVal == "" {
				os.Remove(globalEnvPath)
			} else {
				os.WriteFile(globalEnvPath, []byte(globalEnvVal), 0o644)
			}
		}
		if ack != nil {
			ack(map[string]any{"ok": true, "msg": "Saved", "msgi18n": true})
		}
	})

	socket.On("composerize", func(args ...any) {
		ack := extractAck(args)
		if err := requireLogin(socket); err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		cmd, _ := strArg(args, 0)
		yaml, err := stack.Composerize(cmd)
		if err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		if ack != nil {
			ack(map[string]any{"ok": true, "composeTemplate": yaml})
		}
	})
}
