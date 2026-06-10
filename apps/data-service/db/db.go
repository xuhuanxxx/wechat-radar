package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps sql.DB with additional methods
type DB struct {
	*sql.DB
}

// Init initializes the SQLite database with migrations
func Init(dbPath string) (*DB, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_synchronous=NORMAL&_mmap_size=268435456&_cache_size=-65536&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	// Run migrations
	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate database: %w", err)
	}

	log.Printf("Database initialized: %s", dbPath)
	return &DB{db}, nil
}

func migrate(db *sql.DB) error {
	migrations := []string{
		// messages table
		`CREATE TABLE IF NOT EXISTS messages (
			chatroom_id TEXT NOT NULL,
			local_id TEXT NOT NULL,
			sender TEXT NOT NULL,
			content TEXT NOT NULL,
			timestamp INTEGER NOT NULL,
			type TEXT DEFAULT 'text',
			date TEXT NOT NULL,
			PRIMARY KEY (chatroom_id, local_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_chatroom_date ON messages(chatroom_id, date)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_sender ON messages(sender)`,

		// daily_stats table
		`CREATE TABLE IF NOT EXISTS daily_stats (
			chatroom_id TEXT NOT NULL,
			date TEXT NOT NULL,
			message_count INTEGER DEFAULT 0,
			unique_senders INTEGER DEFAULT 0,
			peak_hour INTEGER DEFAULT 0,
			PRIMARY KEY (chatroom_id, date)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_daily_stats_chatroom ON daily_stats(chatroom_id)`,
		`CREATE INDEX IF NOT EXISTS idx_daily_stats_date ON daily_stats(date)`,

		// mentions table
		`CREATE TABLE IF NOT EXISTS mentions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			chatroom_id TEXT NOT NULL,
			message_id TEXT NOT NULL,
			mentioned TEXT NOT NULL,
			mentioner TEXT NOT NULL,
			date TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_mentions_chatroom ON mentions(chatroom_id)`,
		`CREATE INDEX IF NOT EXISTS idx_mentions_mentioned ON mentions(mentioned)`,

		// topics table
		`CREATE TABLE IF NOT EXISTS topics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			chatroom_id TEXT NOT NULL,
			date TEXT NOT NULL,
			topic TEXT NOT NULL,
			category TEXT NOT NULL,
			confidence REAL DEFAULT 0,
			message_ids TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_topics_chatroom_date ON topics(chatroom_id, date)`,
		`CREATE INDEX IF NOT EXISTS idx_topics_category ON topics(category)`,

		// message_links table
		`CREATE TABLE IF NOT EXISTS message_links (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			chatroom_id TEXT NOT NULL,
			message_id TEXT NOT NULL,
			url TEXT NOT NULL,
			title TEXT,
			date TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_message_links_chatroom ON message_links(chatroom_id)`,
		`CREATE INDEX IF NOT EXISTS idx_message_links_url ON message_links(url)`,

		// sync_state table
		`CREATE TABLE IF NOT EXISTS sync_state (
			chatroom_id TEXT PRIMARY KEY,
			last_sync_at DATETIME,
			last_message_id TEXT,
			message_count INTEGER DEFAULT 0
		)`,

		// groups table
		`CREATE TABLE IF NOT EXISTS groups (
			chatroom_id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			member_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// group_tags table
		`CREATE TABLE IF NOT EXISTS group_tags (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			chatroom_id TEXT NOT NULL,
			tag TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(chatroom_id, tag)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_group_tags_chatroom ON group_tags(chatroom_id)`,

		// favorites table
		`CREATE TABLE IF NOT EXISTS favorites (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			chatroom_id TEXT NOT NULL,
			message_id TEXT NOT NULL,
			sender TEXT NOT NULL,
			content TEXT NOT NULL,
			date TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(chatroom_id, message_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_favorites_chatroom ON favorites(chatroom_id)`,

		// meta table
		`CREATE TABLE IF NOT EXISTS meta (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
	}

	for i, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration %d failed: %w", i, err)
		}
	}

	return nil
}
