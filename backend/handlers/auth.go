package handlers

import (
	"context"
	"database/sql"
	"fmt"

	sio "github.com/zishang520/socket.io/servers/socket/v3"

	"github.com/rahbut/dockge/backend/auth"
	"github.com/rahbut/dockge/backend/models"
	"github.com/rahbut/dockge/backend/ratelimit"
)

// RegisterAuthHandlers registers auth-related socket events on each connection.
// Called once per new socket from the "connection" handler in main.go.
func RegisterAuthHandlers(socket *sio.Socket, srv *Server) {
	// needSetup — returns bare bool (not {ok,...}) matching frontend expectation.
	socket.On("needSetup", func(args ...any) {
		ctx := context.Background()
		count, err := models.CountUsers(ctx)
		ack := extractAck(args)
		if ack == nil {
			return
		}
		if err != nil {
			ack(false)
			return
		}
		ack(count == 0)
	})

	// setup — first-run user creation.
	socket.On("setup", func(args ...any) {
		ack := extractAck(args)
		username, _ := strArg(args, 0)
		password, _ := strArg(args, 1)
		ctx := context.Background()
		count, err := models.CountUsers(ctx)
		if err != nil || count > 0 {
			if ack != nil {
				ack(errResp("Setup is not available"))
			}
			return
		}
		if _, err := models.CreateUser(ctx, username, password); err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		if ack != nil {
			ack(map[string]any{"ok": true, "msg": "Saved", "msgi18n": true})
		}
	})

	// login — username + password authentication.
	socket.On("login", func(args ...any) {
		ack := extractAck(args)
		req, _ := mapArg(args, 0)
		username, _ := req["username"].(string)
		password, _ := req["password"].(string)

		ip := connIP(socket)
		if !ratelimit.LoginLimiter.Allow(ip) {
			if ack != nil {
				ack(errResp("Too many requests, please try again later"))
			}
			return
		}

		ctx := context.Background()
		user, err := models.GetUserByUsername(ctx, username)
		if err == sql.ErrNoRows || user == nil || !auth.VerifyPassword(user.Password.String, password) {
			if ack != nil {
				ack(errResp("Incorrect username or password"))
			}
			return
		}
		if err != nil {
			if ack != nil {
				ack(errResp("Internal error"))
			}
			return
		}

		secret, err := models.GetJWTSecret(ctx)
		if err != nil || secret == "" {
			if ack != nil {
				ack(errResp("Server misconfiguration: missing JWT secret"))
			}
			return
		}
		token, err := user.CreateJWT(secret)
		if err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}

		socket.SetData(username)
		if ack != nil {
			ack(map[string]any{"ok": true, "token": token, "tokenRequired": false})
		}
	})

	// loginByToken — re-authenticate with stored JWT.
	socket.On("loginByToken", func(args ...any) {
		ack := extractAck(args)
		token, _ := strArg(args, 0)
		ctx := context.Background()

		secret, err := models.GetJWTSecret(ctx)
		if err != nil || secret == "" {
			if ack != nil {
				ack(errResp("Server misconfiguration: missing JWT secret"))
			}
			return
		}
		claims, err := auth.ParseJWT(token, secret)
		if err != nil {
			if ack != nil {
				ack(errResp("Invalid or expired token"))
			}
			return
		}
		user, err := models.GetUserByUsername(ctx, claims.Username)
		if err != nil {
			if ack != nil {
				ack(errResp("User not found"))
			}
			return
		}
		if user.PasswordShake256() != claims.H {
			if ack != nil {
				ack(errResp("Token invalidated by password change"))
			}
			return
		}

		socket.SetData(claims.Username)
		if ack != nil {
			ack(okResp())
		}
	})

	// logout.
	socket.On("logout", func(args ...any) {
		socket.SetData(nil)
		ack := extractAck(args)
		if ack != nil {
			ack(okResp())
		}
	})

	// changePassword.
	socket.On("changePassword", func(args ...any) {
		ack := extractAck(args)
		if err := requireLogin(socket); err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		req, _ := mapArg(args, 0)
		current, _ := req["currentPassword"].(string)
		newPwd, _ := req["newPassword"].(string)

		ctx := context.Background()
		username := socketUsername(socket)
		user, err := models.GetUserByUsername(ctx, username)
		if err != nil {
			if ack != nil {
				ack(errResp("User not found"))
			}
			return
		}
		if !auth.VerifyPassword(user.Password.String, current) {
			if ack != nil {
				ack(errResp("Incorrect current password"))
			}
			return
		}
		if err := user.SetPassword(ctx, newPwd); err != nil {
			if ack != nil {
				ack(errResp(err.Error()))
			}
			return
		}
		// Broadcast refresh to all sockets so other sessions re-authenticate.
		srv.SIOServer.Sockets().Emit("refresh")
		if ack != nil {
			ack(map[string]any{"ok": true, "msg": "Saved", "msgi18n": true})
		}
	})
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func requireLogin(socket *sio.Socket) error {
	if socket.Data() == nil {
		return fmt.Errorf("not authenticated")
	}
	return nil
}

func errResp(msg string) map[string]any {
	return map[string]any{"ok": false, "msg": msg}
}

func okResp() map[string]any {
	return map[string]any{"ok": true}
}

// connIP extracts the best-effort remote IP from a socket.
func connIP(socket *sio.Socket) string {
	req := socket.Request()
	if req == nil {
		return ""
	}
	if ip := req.Headers().Peek("x-forwarded-for"); ip != "" {
		return ip
	}
	if ip := req.Headers().Peek("x-real-ip"); ip != "" {
		return ip
	}
	return socket.Conn().RemoteAddress()
}

// extractAck pulls the ack callback out of the variadic args list.
// In zishang520/socket.io v3 the ack is the last argument with type sio.Ack = func([]any, error).
func extractAck(args []any) func(v any) {
	if len(args) == 0 {
		return nil
	}
	if ack, ok := args[len(args)-1].(sio.Ack); ok {
		return func(v any) { ack([]any{v}, nil) }
	}
	return nil
}

// strArg extracts a string argument at position i (before the trailing ack).
func strArg(args []any, i int) (string, bool) {
	if i < len(args) {
		s, ok := args[i].(string)
		return s, ok
	}
	return "", false
}

// mapArg extracts a map argument at position i.
func mapArg(args []any, i int) (map[string]any, bool) {
	if i < len(args) {
		m, ok := args[i].(map[string]any)
		return m, ok
	}
	return nil, false
}

// socketUsername extracts the authenticated username from the socket context,
// handling both the plain string set by login/loginByToken and the *connContext
// set by getOrCreateManager (which wraps the username alongside the AgentManager).
func socketUsername(socket *sio.Socket) string {
	switch v := socket.Data().(type) {
	case string:
		return v
	case *connContext:
		return v.Username
	}
	return ""
}

// boolArg extracts a bool argument at position i.
func boolArg(args []any, i int) bool {
	if i < len(args) {
		b, _ := args[i].(bool)
		return b
	}
	return false
}

// asyncHandler wraps a socket event handler func so it runs in a goroutine,
// preventing the socket's sequential task queue from blocking on long operations
// (e.g. docker compose up). The ack, if present, is called from the goroutine.
func asyncHandler(fn func(args ...any)) func(args ...any) {
	return func(args ...any) {
		go fn(args...)
	}
}
