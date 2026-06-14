package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

// CloudBridge Web Gateway
// Serves the Flutter web app on port 10990 and proxies /api, /health, /signal
// to the signal server, so the browser only needs one host:port.
func main() {
	var (
		addr       = envOr("ADDR", ":10990")
		signalAddr = envOr("SIGNAL_ADDR", "http://localhost:10980")
		webDir     = envOr("WEB_DIR", "./app/build/web")
	)

	signalURL, err := url.Parse(signalAddr)
	if err != nil {
		log.Fatalf("invalid SIGNAL_ADDR %q: %v", signalAddr, err)
	}

	// Reverse proxy for /api, /health, /signal (REST + WebSocket)
	proxy := httputil.NewSingleHostReverseProxy(signalURL)

	// WebSocket needs the director to handle Upgrade requests
	director := proxy.Director
	proxy.Director = func(req *http.Request) {
		director(req)
		req.Host = signalURL.Host
	}

	// CORS middleware
	corsHandler := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	// Static file server for Flutter web app
	fileServer := http.FileServer(http.Dir(webDir))

	mux := http.NewServeMux()

	// API & health endpoints → proxy to signal server
	mux.Handle("/api/", corsHandler(proxy))
	mux.Handle("/health", corsHandler(proxy))

	// WebSocket endpoint → proxy to signal server
	mux.Handle("/signal", corsHandler(proxy))

	// Static files (Flutter web app)
	mux.Handle("/", fileServer)

	log.Printf("CloudBridge Web Gateway starting on %s", addr)
	log.Printf("  Signal server: %s", signalAddr)
	log.Printf("  Web files: %s", webDir)
	log.Printf("  Open http://localhost%s in your browser", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// suppress unused warning
var _ = fmt.Sprintf
var _ = strings.Split