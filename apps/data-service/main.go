package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/xuhuanxxx/wechat-radar/apps/data-service/db"
	"github.com/xuhuanxxx/wechat-radar/apps/data-service/handlers"
	"github.com/xuhuanxxx/wechat-radar/apps/data-service/sync"
)

const version = "0.1.0"

func main() {
	// Parse command line flags
	var (
		dataDirFlag = flag.String("data-dir", "", "Data directory (default: ~/.lark-radar)")
		portFlag    = flag.Int("port", 0, "HTTP port (overrides config)")
	)
	flag.Parse()

	// Determine data directory
	dataDir := *dataDirFlag
	if dataDir == "" {
		dataDir = os.Getenv("LARK_RADAR_DATA_DIR")
	}
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("Failed to get home directory: %v", err)
		}
		dataDir = home + "/.lark-radar"
	}

	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	dbPath := dataDir + "/radar.db"

	// Initialize database
	database, err := db.Init(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Initialize config manager
	configPath := dataDir + "/config.json"
	configMgr := db.NewConfigManager(configPath)

	// Initialize sync engine
	syncEngine := sync.NewEngine(database, configMgr)

	// Initialize handlers
	h := handlers.New(database, configMgr, syncEngine)

	// Setup routes
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("/api/health", h.Health)
	mux.HandleFunc("/api/doctor", h.Doctor)

	// Setup
	mux.HandleFunc("/api/setup", h.Setup)
	mux.HandleFunc("/api/config", h.GetConfig)

	// Stats & Dashboard
	mux.HandleFunc("/api/stats", h.Stats)
	mux.HandleFunc("/api/intelligence", h.Intelligence)

	// Sessions / Chatrooms
	mux.HandleFunc("/api/sessions", h.Sessions)
	mux.HandleFunc("/api/sessions/", h.SessionDetail)

	// Groups (more specific first)
	mux.HandleFunc("/api/groups/tags", h.GroupTags)
	mux.HandleFunc("/api/groups/", h.GroupDetail)
	mux.HandleFunc("/api/groups", h.Groups)

	// Sync
	mux.HandleFunc("/api/sync", h.Sync)
	mux.HandleFunc("/api/sync/progress", h.SyncProgress)

	// Topics (more specific first)
	mux.HandleFunc("/api/topics/analyze", h.AnalyzeTopics)
	mux.HandleFunc("/api/topics/links", h.TopicLinks)
	mux.HandleFunc("/api/topics/", h.TopicDetail)
	mux.HandleFunc("/api/topics", h.Topics)

	// Mentions
	mux.HandleFunc("/api/mentions/stats", h.MentionStats)
	mux.HandleFunc("/api/mentions", h.Mentions)

	// Links
	mux.HandleFunc("/api/links/analyze", h.AnalyzeLinks)
	mux.HandleFunc("/api/links", h.Links)

	// Favorites
	mux.HandleFunc("/api/favorites/toggle", h.ToggleFavorite)
	mux.HandleFunc("/api/favorites", h.Favorites)

	// Reports
	mux.HandleFunc("/api/reports/generate", h.GenerateReport)

	// Search
	mux.HandleFunc("/api/search", h.Search)

	// Lark specific
	mux.HandleFunc("/api/lark/sync", h.LarkSync)
	mux.HandleFunc("/api/lark/messages", h.LarkMessages)
	mux.HandleFunc("/api/lark/chats", h.LarkChats)

	// AI Classify
	mux.HandleFunc("/api/ai-classify", h.AIClassify)

	// Message Links
	mux.HandleFunc("/api/message-links/raw", h.MessageLinksRaw)
	mux.HandleFunc("/api/message-links/backfill", h.MessageLinksBackfill)
	mux.HandleFunc("/api/message-links/resolve", h.MessageLinksResolve)

	// Extra handlers
	mux.HandleFunc("/api/new-messages", h.NewMessages)
	mux.HandleFunc("/api/rescan", h.Rescan)
	mux.HandleFunc("/api/wx-image", h.WXImage)

	// Wrap with CORS and logging
	handler := withCORS(withLogging(mux))

	// Determine port: flag > config > default
	port := 8787
	if cfg, err := configMgr.Load(); err == nil && cfg.Port > 0 {
		port = cfg.Port
	}
	if *portFlag > 0 {
		port = *portFlag
	}

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	// Ignore SIGHUP so the server keeps running when the parent shell exits
	signal.Ignore(syscall.SIGHUP)

	log.Printf("Lark Radar Data Service v%s starting on port %d", version, port)
	log.Printf("Database: %s", dbPath)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)
		log.Printf("%s %s %s", r.Method, r.URL.Path, duration)
	})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]interface{}{
		"ok":    false,
		"error": message,
	})
}
