package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
)

var supportedLangs = map[string]bool{
	"zh": true, "ja": true, "ko": true, "es": true,
}

//go:embed static/*
var staticFiles embed.FS

func main() {
	loadConfig()

	var rdb *redis.Client
	if AppConfig.RedisAddr == "" {
		log.Printf("Redis disabled, running without cache")
	} else if conn, err := net.DialTimeout("tcp", AppConfig.RedisAddr, 2*time.Second); err != nil {
		log.Printf("Redis not available at %s, running without cache", AppConfig.RedisAddr)
	} else {
		conn.Close()
		rdb = redis.NewClient(&redis.Options{Addr: AppConfig.RedisAddr})
		log.Printf("Redis connected at %s", AppConfig.RedisAddr)
	}

	limiter := NewRateLimiter(AppConfig.RateLimit, AppConfig.RateBurst)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/search", handleSearch(rdb))
	mux.HandleFunc("/api/whois/", handleWhois(rdb))

	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatal(err)
	}
	buildLangPages(staticFS)
	fileServer := http.FileServer(http.FS(staticFS))
	mux.Handle("/", langMiddleware(fileServer))

	handler := limiter.Middleware(corsMiddleware(mux))

	srv := &http.Server{
		Addr:         ":" + AppConfig.Port,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("dmcheck service listening on :%s", AppConfig.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-sigCtx.Done()
	log.Println("shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
	log.Println("server stopped")
}

func langMiddleware(fileServer http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")

		if path == "" || path == "index.html" {
			serveLangPage(w, "en")
			return
		}

		parts := strings.SplitN(path, "/", 2)
		if len(parts) >= 1 && supportedLangs[parts[0]] {
			rest := ""
			if len(parts) > 1 {
				rest = parts[1]
			}
			if rest == "" || rest == "index.html" {
				serveLangPage(w, parts[0])
				return
			}
			r.URL.Path = "/" + rest
		}
		fileServer.ServeHTTP(w, r)
	})
}

func serveLangPage(w http.ResponseWriter, lang string) {
	page, ok := langPages[lang]
	if !ok {
		page = langPages["en"]
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(page)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
