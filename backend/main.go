package main

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	sio "github.com/zishang520/socket.io/servers/socket/v3"

	"github.com/rahbut/dockge/backend/db"
	"github.com/rahbut/dockge/backend/handlers"
	"github.com/rahbut/dockge/backend/models"
	"github.com/rahbut/dockge/backend/router"
)

var Version = "dev"

func main() {
	cfg := LoadConfig()

	rootCmd := &cobra.Command{
		Use:   "dockge",
		Short: "Dockge — a fancy Docker Compose stack manager",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cfg)
		},
	}

	flags := rootCmd.Flags()
	flags.IntVar(&cfg.Port, "port", cfg.Port, "HTTP listen port")
	flags.StringVar(&cfg.Hostname, "hostname", cfg.Hostname, "HTTP listen hostname")
	flags.StringVar(&cfg.DataDir, "dataDir", cfg.DataDir, "Data directory")
	flags.StringVar(&cfg.StacksDir, "stacksDir", cfg.StacksDir, "Stacks directory")
	flags.StringVar(&cfg.SSLKey, "sslKey", cfg.SSLKey, "Path to SSL private key")
	flags.StringVar(&cfg.SSLCert, "sslCert", cfg.SSLCert, "Path to SSL certificate")
	flags.StringVar(&cfg.SSLKeyPassphrase, "sslKeyPassphrase", cfg.SSLKeyPassphrase, "SSL key passphrase")
	flags.BoolVar(&cfg.EnableConsole, "enableConsole", cfg.EnableConsole, "Enable bash console")

	rootCmd.AddCommand(&cobra.Command{
		Use:   "healthcheck",
		Short: "Check the health of a running Dockge instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHealthCheck(cfg)
		},
	})

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cfg *Config) error {
	initLogger(cfg)
	log.Info().Str("version", Version).Msg("Starting Dockge")

	for _, dir := range []string{cfg.DataDir, cfg.StacksDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create directory %q: %w", dir, err)
		}
	}

	dbPath := filepath.Join(cfg.DataDir, "dockge.db")
	if err := db.Init(dbPath); err != nil {
		return fmt.Errorf("database init: %w", err)
	}
	defer db.Close()

	ctx := context.Background()
	secret, err := models.GetJWTSecret(ctx)
	if err != nil || secret == "" {
		b := make([]byte, 32)
		rand.Read(b)
		secret = hex.EncodeToString(b)
		if err := models.SetSetting(ctx, "jwtSecret", secret, "security"); err != nil {
			return fmt.Errorf("persist JWT secret: %w", err)
		}
	}

	// ── Socket.io server (zishang520/socket.io v3 — EIO v4 compatible) ───
	io := sio.NewServer(nil, nil)

	srv := &handlers.Server{
		SIOServer:     io,
		StacksDir:     cfg.StacksDir,
		DataDir:       cfg.DataDir,
		Version:       Version,
		IsDev:         cfg.IsDev,
		IsContainer:   cfg.IsContainer,
		EnableConsole: cfg.EnableConsole,
	}

	// Register per-connection handlers.
	io.On("connection", func(args ...any) {
		socket, ok := args[0].(*sio.Socket)
		if !ok {
			return
		}

		// Origin check.
		if !cfg.IsDev && cfg.WSOriginCheck != "bypass" {
			req := socket.Request()
			if req != nil {
				origin := req.Headers().Peek("origin")
				host := req.Headers().Peek("host")
				if !originAllowed(origin, host) {
					log.Warn().Str("origin", origin).Msg("WebSocket origin rejected")
					socket.Disconnect(true)
					return
				}
			}
		}

		// Send server info immediately on connect.
		srv.SendInfo(socket)

		// Check if setup is needed or auth is disabled.
		connCtx := context.Background()
		count, _ := models.CountUsers(connCtx)
		if count == 0 {
			socket.Emit("setup")
		} else {
			disableAuth, _ := models.GetDisableAuth(connCtx)
			if disableAuth {
				socket.Emit("autoLogin")
			}
		}

		// Register all event handlers on this socket.
		handlers.RegisterAuthHandlers(socket, srv)
		handlers.RegisterSettingsHandlers(socket, srv)
		handlers.RegisterDockerHandlers(socket, srv)
		handlers.RegisterTerminalHandlers(socket, srv)
		handlers.RegisterAgentHandlers(socket, srv)
		handlers.RegisterAgentProxyHandler(socket, srv)
	})

	// ── HTTP mux ──────────────────────────────────────────────────────────
	frontendDist := "frontend-dist"
	if _, err := os.Stat(frontendDist); os.IsNotExist(err) {
		log.Warn().Msg("frontend-dist/ not found — UI will not be served")
	}

	mux := http.NewServeMux()
	router.Register(mux, frontendDist, io.ServeHandler(nil))

	// ── Cron: broadcast stack list every 10 s ─────────────────────────────
	c := cron.New()
	c.AddFunc("@every 10s", srv.BroadcastStackList)
	c.Start()
	defer c.Stop()

	// ── HTTP / HTTPS server ───────────────────────────────────────────────
	addr := fmt.Sprintf("%s:%d", cfg.Hostname, cfg.Port)
	httpServer := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	if cfg.SSLKey != "" && cfg.SSLCert != "" {
		tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
		cert, err := tls.LoadX509KeyPair(cfg.SSLCert, cfg.SSLKey)
		if err != nil {
			return fmt.Errorf("load TLS cert: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
		httpServer.TLSConfig = tlsCfg
	}

	proto := "http"
	if httpServer.TLSConfig != nil {
		proto = "https"
	}
	log.Info().Str("addr", fmt.Sprintf("%s://%s", proto, addr)).Msg("Dockge listening")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		if httpServer.TLSConfig != nil {
			errCh <- httpServer.ListenAndServeTLS(cfg.SSLCert, cfg.SSLKey)
		} else {
			errCh <- httpServer.ListenAndServe()
		}
	}()

	select {
	case err := <-errCh:
		if err != http.ErrServerClosed {
			return err
		}
	case <-quit:
		log.Info().Msg("Shutting down...")
		shutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		httpServer.Shutdown(shutCtx)
		io.Close(nil)
	}

	return nil
}

func runHealthCheck(cfg *Config) error {
	isK8s := strings.HasPrefix(os.Getenv("DOCKGE_PORT"), "tcp://")

	port := ""
	if !isK8s {
		port = os.Getenv("DOCKGE_PORT")
	}
	if port == "" {
		port = fmt.Sprintf("%d", cfg.Port)
	}

	hostname := os.Getenv("DOCKGE_HOST")
	if hostname == "" {
		hostname = "127.0.0.1"
	}

	scheme := "http"
	if cfg.SSLKey != "" && cfg.SSLCert != "" {
		scheme = "https"
	}

	url := fmt.Sprintf("%s://%s:%s/health", scheme, hostname, port)
	client := &http.Client{
		Timeout: 28 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("healthcheck failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("healthcheck returned status %d", resp.StatusCode)
	}
	return nil
}

func originAllowed(origin, host string) bool {
	if origin == "" {
		return true
	}
	origin = strings.TrimPrefix(strings.TrimPrefix(origin, "https://"), "http://")
	return origin == host || strings.HasPrefix(origin, host)
}
