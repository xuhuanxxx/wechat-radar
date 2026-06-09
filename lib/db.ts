import Database from 'better-sqlite3';
import { existsSync, mkdirSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { DATA_DIR } from './config';

const DB_PATH = join(DATA_DIR, 'radar.db');

let _db: Database.Database | null = null;

export function db(): Database.Database {
  if (_db) return _db;
  const dir = dirname(DB_PATH);
  if (!existsSync(dir)) mkdirSync(dir, { recursive: true });
  _db = new Database(DB_PATH);
  _db.pragma('journal_mode = WAL');
  _db.pragma('foreign_keys = ON');
  migrate(_db);
  seed(_db);
  return _db;
}

function migrate(d: Database.Database) {
  d.exec(`
    CREATE TABLE IF NOT EXISTS groups (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      name TEXT NOT NULL UNIQUE,
      color TEXT NOT NULL,
      emoji TEXT,
      sort_order INTEGER NOT NULL DEFAULT 0,
      created_at INTEGER NOT NULL
    );

    CREATE TABLE IF NOT EXISTS group_tags (
      chatroom_id TEXT NOT NULL,
      group_id INTEGER NOT NULL,
      PRIMARY KEY (chatroom_id, group_id),
      FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE
    );

    CREATE TABLE IF NOT EXISTS favorites (
      chatroom_id TEXT PRIMARY KEY,
      starred_at INTEGER NOT NULL
    );

    CREATE TABLE IF NOT EXISTS daily_stats (
      chatroom_id TEXT NOT NULL,
      date TEXT NOT NULL,
      total INTEGER NOT NULL,
      top_senders TEXT NOT NULL,
      by_hour TEXT NOT NULL,
      refreshed_at INTEGER NOT NULL,
      PRIMARY KEY (chatroom_id, date)
    );

    CREATE INDEX IF NOT EXISTS idx_daily_stats_date ON daily_stats(date);

    CREATE TABLE IF NOT EXISTS mentions (
      chatroom_id TEXT NOT NULL,
      local_id TEXT NOT NULL,
      sender TEXT NOT NULL,
      content TEXT NOT NULL,
      time TEXT NOT NULL,
      timestamp INTEGER NOT NULL,
      seen INTEGER NOT NULL DEFAULT 0,
      PRIMARY KEY (chatroom_id, local_id)
    );

    CREATE INDEX IF NOT EXISTS idx_mentions_time ON mentions(timestamp DESC);

    CREATE TABLE IF NOT EXISTS messages (
      chatroom_id TEXT NOT NULL,
      local_id TEXT NOT NULL,
      sender TEXT NOT NULL,
      sender_name TEXT,
      content TEXT NOT NULL,
      time TEXT NOT NULL,
      timestamp INTEGER NOT NULL,
      type TEXT NOT NULL,
      date TEXT NOT NULL,
      source TEXT NOT NULL DEFAULT 'lark',
      raw TEXT,
      PRIMARY KEY (chatroom_id, local_id)
    );

    CREATE INDEX IF NOT EXISTS idx_messages_chatroom_date ON messages(chatroom_id, date);
    CREATE INDEX IF NOT EXISTS idx_messages_date ON messages(date);
    CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp DESC);
    CREATE INDEX IF NOT EXISTS idx_messages_sender ON messages(sender);

    CREATE TABLE IF NOT EXISTS message_links (
      chatroom_id TEXT NOT NULL,
      local_id TEXT NOT NULL,
      date TEXT NOT NULL,
      sender TEXT NOT NULL,
      time TEXT NOT NULL,
      timestamp INTEGER NOT NULL,
      url TEXT NOT NULL,
      canonical_url TEXT NOT NULL,
      title TEXT,
      description TEXT,
      domain TEXT NOT NULL,
      source TEXT NOT NULL,
      raw_kind TEXT NOT NULL,
      confidence REAL NOT NULL DEFAULT 1,
      created_at INTEGER NOT NULL,
      PRIMARY KEY (chatroom_id, local_id, canonical_url)
    );

    CREATE INDEX IF NOT EXISTS idx_message_links_date
      ON message_links(date, timestamp DESC);
    CREATE INDEX IF NOT EXISTS idx_message_links_canonical
      ON message_links(canonical_url);
    CREATE INDEX IF NOT EXISTS idx_message_links_domain
      ON message_links(domain);
    CREATE INDEX IF NOT EXISTS idx_message_links_source
      ON message_links(source);

    CREATE TABLE IF NOT EXISTS topics (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      date TEXT NOT NULL,
      title TEXT NOT NULL,
      summary TEXT,
      message_count INTEGER NOT NULL,
      group_count INTEGER NOT NULL,
      created_at INTEGER NOT NULL
    );

    CREATE INDEX IF NOT EXISTS idx_topics_date ON topics(date DESC, message_count DESC);

    CREATE TABLE IF NOT EXISTS topic_messages (
      topic_id INTEGER NOT NULL,
      chatroom_id TEXT NOT NULL,
      local_id TEXT NOT NULL,
      score REAL NOT NULL DEFAULT 0,
      PRIMARY KEY (topic_id, chatroom_id, local_id),
      FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE
    );

    CREATE INDEX IF NOT EXISTS idx_topic_messages_topic ON topic_messages(topic_id);

    CREATE TABLE IF NOT EXISTS link_intelligence_cache (
      date TEXT NOT NULL,
      version TEXT NOT NULL,
      payload TEXT NOT NULL,
      generated_at INTEGER NOT NULL,
      PRIMARY KEY (date, version)
    );

    CREATE INDEX IF NOT EXISTS idx_link_intelligence_cache_generated
      ON link_intelligence_cache(generated_at DESC);

    CREATE TABLE IF NOT EXISTS sync_state (
      chatroom_id TEXT NOT NULL,
      source TEXT NOT NULL DEFAULT 'lark',
      last_synced_at INTEGER NOT NULL,
      first_message_date TEXT,
      last_message_date TEXT,
      total_messages INTEGER NOT NULL DEFAULT 0,
      last_sync_time TEXT,
      PRIMARY KEY (chatroom_id, source)
    );

    CREATE TABLE IF NOT EXISTS meta (
      key TEXT PRIMARY KEY,
      value TEXT NOT NULL
    );
  `);

  ensureColumn(d, 'sync_state', 'status', "TEXT NOT NULL DEFAULT 'unknown'");
  ensureColumn(d, 'sync_state', 'last_error', 'TEXT');
  ensureColumn(d, 'sync_state', 'failed_chunks', 'INTEGER NOT NULL DEFAULT 0');
  ensureColumn(d, 'sync_state', 'empty_chunks', 'INTEGER NOT NULL DEFAULT 0');
  ensureColumn(d, 'sync_state', 'total_chunks', 'INTEGER NOT NULL DEFAULT 0');
  ensureColumn(d, 'sync_state', 'last_sync_time', 'TEXT');
  ensureColumn(d, 'messages', 'sender_name', 'TEXT');
  ensureColumn(d, 'messages', 'raw', 'TEXT');
  ensureColumn(d, 'messages', 'source', "TEXT NOT NULL DEFAULT 'lark'");

  // Migrate message_links.local_id from INTEGER to TEXT if it was created as INTEGER.
  const linksInfo = d.prepare("PRAGMA table_info(message_links)").all() as Array<{ name: string; type: string; pk: number }>;
  const linksLocalId = linksInfo.find((c) => c.name === 'local_id');
  if (linksLocalId && linksLocalId.type.toUpperCase() === 'INTEGER') {
    d.exec(`
      CREATE TABLE message_links_new (
        chatroom_id TEXT NOT NULL,
        local_id TEXT NOT NULL,
        date TEXT NOT NULL,
        sender TEXT NOT NULL,
        time TEXT NOT NULL,
        timestamp INTEGER NOT NULL,
        url TEXT NOT NULL,
        canonical_url TEXT NOT NULL,
        title TEXT,
        description TEXT,
        domain TEXT NOT NULL,
        source TEXT NOT NULL,
        raw_kind TEXT NOT NULL,
        confidence REAL NOT NULL DEFAULT 1,
        created_at INTEGER NOT NULL,
        PRIMARY KEY (chatroom_id, local_id, canonical_url)
      );
      INSERT INTO message_links_new
        (chatroom_id, local_id, date, sender, time, timestamp, url, canonical_url, title, description, domain, source, raw_kind, confidence, created_at)
      SELECT
        chatroom_id, CAST(local_id AS TEXT), date, sender, time, timestamp, url, canonical_url, title, description, domain, source, raw_kind, confidence, created_at
      FROM message_links;
      DROP TABLE message_links;
      ALTER TABLE message_links_new RENAME TO message_links;
      CREATE INDEX IF NOT EXISTS idx_message_links_date ON message_links(date, timestamp DESC);
      CREATE INDEX IF NOT EXISTS idx_message_links_canonical ON message_links(canonical_url);
      CREATE INDEX IF NOT EXISTS idx_message_links_domain ON message_links(domain);
      CREATE INDEX IF NOT EXISTS idx_message_links_source ON message_links(source);
    `);
  }

  // Migrate mentions.local_id from INTEGER to TEXT if needed.
  const mentionsInfo = d.prepare("PRAGMA table_info(mentions)").all() as Array<{ name: string; type: string }>;
  const mentionsLocalId = mentionsInfo.find((c) => c.name === 'local_id');
  if (mentionsLocalId && mentionsLocalId.type.toUpperCase() === 'INTEGER') {
    d.exec(`
      CREATE TABLE mentions_new (
        chatroom_id TEXT NOT NULL,
        local_id TEXT NOT NULL,
        sender TEXT NOT NULL,
        content TEXT NOT NULL,
        time TEXT NOT NULL,
        timestamp INTEGER NOT NULL,
        seen INTEGER NOT NULL DEFAULT 0,
        PRIMARY KEY (chatroom_id, local_id)
      );
      INSERT INTO mentions_new (chatroom_id, local_id, sender, content, time, timestamp, seen)
      SELECT chatroom_id, CAST(local_id AS TEXT), sender, content, time, timestamp, seen FROM mentions;
      DROP TABLE mentions;
      ALTER TABLE mentions_new RENAME TO mentions;
      CREATE INDEX IF NOT EXISTS idx_mentions_time ON mentions(timestamp DESC);
    `);
  }

  // Migrate topic_messages.local_id from INTEGER to TEXT if needed.
  const tmInfo = d.prepare("PRAGMA table_info(topic_messages)").all() as Array<{ name: string; type: string }>;
  const tmLocalId = tmInfo.find((c) => c.name === 'local_id');
  if (tmLocalId && tmLocalId.type.toUpperCase() === 'INTEGER') {
    d.exec(`
      CREATE TABLE topic_messages_new (
        topic_id INTEGER NOT NULL,
        chatroom_id TEXT NOT NULL,
        local_id TEXT NOT NULL,
        score REAL NOT NULL DEFAULT 0,
        PRIMARY KEY (topic_id, chatroom_id, local_id),
        FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE
      );
      INSERT INTO topic_messages_new (topic_id, chatroom_id, local_id, score)
      SELECT topic_id, chatroom_id, CAST(local_id AS TEXT), score FROM topic_messages;
      DROP TABLE topic_messages;
      ALTER TABLE topic_messages_new RENAME TO topic_messages;
      CREATE INDEX IF NOT EXISTS idx_topic_messages_topic ON topic_messages(topic_id);
    `);
  }

  // SQLite < 3.35.0 does not support DROP COLUMN; we recreate sync_state if needed
  // to switch from single-column PK to composite PK. This is safe because sync_state
  // is derived data that can be rebuilt from messages.
  const syncInfo = d.prepare("PRAGMA table_info(sync_state)").all() as Array<{ name: string; pk: number }>;
  const hasSource = syncInfo.some((c) => c.name === 'source');
  const pkCols = syncInfo.filter((c) => c.pk > 0).map((c) => c.name);
  if (hasSource && pkCols.length === 1 && pkCols[0] === 'chatroom_id') {
    d.exec(`
      CREATE TABLE sync_state_new (
        chatroom_id TEXT NOT NULL,
        source TEXT NOT NULL DEFAULT 'lark',
        last_synced_at INTEGER NOT NULL,
        first_message_date TEXT,
        last_message_date TEXT,
        total_messages INTEGER NOT NULL DEFAULT 0,
        last_sync_time TEXT,
        status TEXT NOT NULL DEFAULT 'unknown',
        last_error TEXT,
        failed_chunks INTEGER NOT NULL DEFAULT 0,
        empty_chunks INTEGER NOT NULL DEFAULT 0,
        total_chunks INTEGER NOT NULL DEFAULT 0,
        PRIMARY KEY (chatroom_id, source)
      );
      INSERT INTO sync_state_new (
        chatroom_id, source, last_synced_at, first_message_date, last_message_date,
        total_messages, last_sync_time, status, last_error, failed_chunks, empty_chunks, total_chunks
      )
      SELECT
        chatroom_id, 'lark', last_synced_at, first_message_date, last_message_date,
        total_messages, NULL, status, last_error, failed_chunks, empty_chunks, total_chunks
      FROM sync_state;
      DROP TABLE sync_state;
      ALTER TABLE sync_state_new RENAME TO sync_state;
    `);
  }
}

function ensureColumn(
  d: Database.Database,
  table: string,
  name: string,
  definition: string,
) {
  const rows = d.prepare(`PRAGMA table_info(${table})`).all() as Array<{ name: string }>;
  if (rows.some((r) => r.name === name)) return;
  d.prepare(`ALTER TABLE ${table} ADD COLUMN ${name} ${definition}`).run();
}

const SEED_VERSION = 'qiaomu_v2_2026_05_23';

const DEFAULT_GROUPS: Array<{ name: string; color: string; emoji: string }> = [
  { name: 'AI产品蝗虫团', color: '#ef4444', emoji: '🐝' },
  { name: '自营/读者群', color: '#22c55e', emoji: '🌟' },
  { name: 'WaytoAGI', color: '#06b6d4', emoji: '🛸' },
  { name: 'HowOneAI', color: '#0ea5e9', emoji: '🚀' },
  { name: 'Vibe Coding · 编程', color: '#6366f1', emoji: '💻' },
  { name: 'AIGC · 内容创作', color: '#ec4899', emoji: '🎨' },
  { name: 'AI 学术', color: '#a855f7', emoji: '🎓' },
  { name: 'AI 商业 · 营销', color: '#10b981', emoji: '💰' },
  { name: 'AI 工具用户群', color: '#f59e0b', emoji: '🛠️' },
  { name: '付费社区', color: '#eab308', emoji: '💎' },
  { name: 'AI 圈社交', color: '#8b5cf6', emoji: '🤖' },
  { name: '大佬 · 媒体圈', color: '#f97316', emoji: '📰' },
  { name: '行业活动', color: '#22d3ee', emoji: '🎯' },
  { name: '生活 · 兴趣', color: '#fb7185', emoji: '🏘️' },
];

function seed(d: Database.Database) {
  const meta = d
    .prepare("SELECT value FROM meta WHERE key = 'seed_version'")
    .get() as { value: string } | undefined;

  if (meta?.value === SEED_VERSION) return;

  // Check if any groups have user tags — if so, leave them alone (additive seed).
  const tagged = d.prepare('SELECT COUNT(*) AS n FROM group_tags').get() as { n: number };

  if (tagged.n === 0) {
    // Safe to wipe and re-seed.
    d.prepare('DELETE FROM groups').run();
  }

  const now = Date.now();
  const insertOrIgnore = d.prepare(
    'INSERT OR IGNORE INTO groups (name, color, emoji, sort_order, created_at) VALUES (?, ?, ?, ?, ?)',
  );
  d.transaction(() => {
    DEFAULT_GROUPS.forEach((g, i) => insertOrIgnore.run(g.name, g.color, g.emoji, i, now));
  })();

  d.prepare(
    "INSERT OR REPLACE INTO meta (key, value) VALUES ('seed_version', ?)",
  ).run(SEED_VERSION);
}
