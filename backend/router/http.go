// Package router provides the HTTP routes for the Dockge web server.
package router

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Register sets up HTTP routes on mux:
//   - GET /robots.txt         → disallow all crawlers
//   - GET /* (static assets)  → serve frontend-dist/ with brotli/gzip pre-compressed support
//   - GET / (SPA shell)       → serve frontend-dist/index.html
//
// socketHandler is the Socket.io http.Handler, attached at /.
func Register(mux *http.ServeMux, frontendDistDir string, socketHandler http.Handler) {
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "User-agent: *\nDisallow: /\n")
	})

	// Health endpoint used by the Go healthcheck binary.
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Static asset handler with pre-compressed file support.
	mux.Handle("/", serveStaticWithFallback(frontendDistDir, socketHandler))
}

// serveStaticWithFallback returns a handler that:
//  1. Delegates Socket.io upgrade requests to the socket handler.
//  2. Serves pre-compressed (.br / .gz) variants when the client accepts them.
//  3. Falls back to frontend-dist/index.html for all other GET requests
//     (SPA client-side routing).
func serveStaticWithFallback(distDir string, socketHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Socket.io requests (upgrade or polling).
		if strings.HasPrefix(r.URL.Path, "/socket.io/") {
			socketHandler.ServeHTTP(w, r)
			return
		}

		// Map URL path to a filesystem path.
		reqPath := r.URL.Path
		if reqPath == "/" {
			reqPath = "/index.html"
		}
		filePath := filepath.Join(distDir, filepath.Clean(reqPath))

		// Try brotli first, then gzip, then the raw file.
		acceptEnc := r.Header.Get("Accept-Encoding")
		if strings.Contains(acceptEnc, "br") {
			if tryServe(w, r, filePath+".br", "br") {
				return
			}
		}
		if strings.Contains(acceptEnc, "gzip") {
			if tryServe(w, r, filePath+".gz", "gzip") {
				return
			}
		}
		if tryServeFile(w, r, filePath) {
			return
		}

		// SPA fallback: serve index.html for any unknown path.
		indexPath := filepath.Join(distDir, "index.html")
		if tryServeFile(w, r, indexPath) {
			return
		}

		http.NotFound(w, r)
	})
}

func tryServe(w http.ResponseWriter, r *http.Request, path, encoding string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		return false
	}
	w.Header().Set("Content-Encoding", encoding)
	setContentType(w, path)
	http.ServeContent(w, r, path, info.ModTime(), f)
	return true
}

func tryServeFile(w http.ResponseWriter, r *http.Request, path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		return false
	}
	http.ServeContent(w, r, info.Name(), info.ModTime(), f)
	return true
}

// setContentType sets the correct Content-Type for compressed files by stripping
// the compression extension (.br or .gz) and using the original extension.
func setContentType(w http.ResponseWriter, path string) {
	stripped := strings.TrimSuffix(strings.TrimSuffix(path, ".br"), ".gz")
	switch {
	case strings.HasSuffix(stripped, ".js"):
		w.Header().Set("Content-Type", "application/javascript")
	case strings.HasSuffix(stripped, ".css"):
		w.Header().Set("Content-Type", "text/css")
	case strings.HasSuffix(stripped, ".html"):
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case strings.HasSuffix(stripped, ".json"):
		w.Header().Set("Content-Type", "application/json")
	case strings.HasSuffix(stripped, ".svg"):
		w.Header().Set("Content-Type", "image/svg+xml")
	case strings.HasSuffix(stripped, ".woff2"):
		w.Header().Set("Content-Type", "font/woff2")
	}
}
