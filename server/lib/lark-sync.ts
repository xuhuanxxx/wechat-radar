import { db } from './db';
import { readConfig, type Config } from './config';
import {
  larkAllChats,
  larkAllMessages,
  larkDoctor,
  type LarkChat,
  type LarkMessage,
} from './lark';
import { upsertLinksForMessage } from './message-links';
import { aggregateDailyStats } from './messages-store';
import { saveStats } from './stats-aggregator';
import { rebuildMentionIndexFromMessages } from './mentions';
import { cache, CK } from './cache';

export interface LarkSyncResult {
  ok: boolean;
  error?: string;
  synced: Record<
    string,
    {
      inserted: number;
      skipped: number;
      error: string | null;
    }
  >;
}

export interface LarkMessageRow {
  chatroom_id: string;
  local_id: string;
  sender: string;
  sender_name: string;
  content: string;
  time: string;
  timestamp: number;
  type: string;
  date: string;
  source: 'lark';
  raw: string;
}

// In-memory sync lock to prevent concurrent sync runs
let _syncRunning = false;

// Per-chat cooldown: minimum seconds between syncs for the same chat
const CHAT_COOLDOWN_MS = 30_000;
const _lastSyncByChat = new Map<string, number>();

// In-memory dedup cache for message IDs (LRU-style, max 50k entries)
const _messageIdCache = new Set<string>();
const MAX_CACHE_SIZE = 50_000;

export function larkMessageToRow(chatId: string, m: LarkMessage): LarkMessageRow {
  const timestamp = parseCreateTime(m.create_time);
  const date = formatDate(timestamp);
  const time = formatTime(timestamp);
  const { text, msgType } = extractText(m);
  return {
    chatroom_id: chatId,
    local_id: m.message_id || '',
    sender: m.sender?.id || '',
    sender_name: m.sender?.name || '',
    content: text,
    time,
    timestamp,
    type: msgType,
    date,
    source: 'lark',
    raw: JSON.stringify(m),
  };
}

function parseCreateTime(raw: string | undefined): number {
  if (!raw) return Math.floor(Date.now() / 1000);
  const iso = Date.parse(raw);
  if (Number.isFinite(iso)) return Math.floor(iso / 1000);
  const ms = Number(raw);
  if (Number.isFinite(ms)) {
    return ms > 1e12 ? Math.floor(ms / 1000) : Math.floor(ms);
  }
  return Math.floor(Date.now() / 1000);
}

function formatDate(ts: number): string {
  const d = new Date(ts * 1000);
  const y = d.getFullYear();
  const mo = String(d.getMonth() + 1).padStart(2, '0');
  const da = String(d.getDate()).padStart(2, '0');
  return `${y}-${mo}-${da}`;
}

function formatTime(ts: number): string {
  const d = new Date(ts * 1000);
  const y = d.getFullYear();
  const mo = String(d.getMonth() + 1).padStart(2, '0');
  const da = String(d.getDate()).padStart(2, '0');
  const h = String(d.getHours()).padStart(2, '0');
  const mi = String(d.getMinutes()).padStart(2, '0');
  const s = String(d.getSeconds()).padStart(2, '0');
  return `${y}-${mo}-${da} ${h}:${mi}:${s}`;
}

function extractText(m: LarkMessage): { text: string; msgType: string } {
  const rawType = (m.msg_type || 'unknown').toLowerCase();
  const contentRaw = m.content || m.body?.content || '{}';

  switch (rawType) {
    case 'text': {
      if (contentRaw && !contentRaw.startsWith('{')) {
        return { text: contentRaw, msgType: 'text' };
      }
      try {
        const parsed = JSON.parse(contentRaw);
        return { text: parsed.text || '', msgType: 'text' };
      } catch {
        return { text: contentRaw, msgType: 'text' };
      }
    }
    case 'post': {
      try {
        const parsed = JSON.parse(contentRaw);
        return { text: extractPostText(parsed), msgType: 'post' };
      } catch {
        return { text: '[富文本]', msgType: 'post' };
      }
    }
    case 'card': {
      try {
        const parsed = JSON.parse(contentRaw);
        return { text: extractCardText(parsed), msgType: 'card' };
      } catch {
        return { text: '[卡片消息]', msgType: 'card' };
      }
    }
    case 'image':
      return { text: '[图片]', msgType: 'image' };
    case 'file':
      return { text: '[文件]', msgType: 'file' };
    case 'media':
    case 'video':
      return { text: '[视频]', msgType: 'video' };
    case 'audio':
    case 'voice':
      return { text: '[语音]', msgType: 'audio' };
    case 'sticker':
      return { text: '[表情]', msgType: 'sticker' };
    case 'interactive':
      return { text: '[交互卡片]', msgType: 'interactive' };
    case 'system':
      return { text: contentRaw, msgType: 'system' };
    default:
      return { text: `[未知消息: ${rawType}]`, msgType: rawType };
  }
}

function extractPostText(node: unknown): string {
  if (!node || typeof node !== 'object') return '';
  const n = node as Record<string, unknown>;
  if (n.content && Array.isArray(n.content)) {
    return n.content.map(extractPostText).join('');
  }
  if (Array.isArray(n)) {
    return n.map(extractPostText).join('');
  }
  if (n.tag === 'text' && typeof n.text === 'string') {
    return n.text;
  }
  if (n.tag === 'a' && typeof n.text === 'string') {
    const href = typeof n.href === 'string' ? ` (${n.href})` : '';
    return n.text + href;
  }
  if (n.tag === 'at' && typeof n.user_name === 'string') {
    return `@${n.user_name} `;
  }
  if (n.children && Array.isArray(n.children)) {
    return n.children.map(extractPostText).join('');
  }
  if (n.elements && Array.isArray(n.elements)) {
    return n.elements.map(extractPostText).join('');
  }
  return '';
}

function extractCardText(card: unknown): string {
  if (!card || typeof card !== 'object') return '[卡片消息]';
  const c = card as Record<string, unknown>;
  const parts: string[] = [];
  if (typeof c.header === 'object' && c.header) {
    const h = c.header as Record<string, unknown>;
    if (typeof h.title === 'object' && h.title) {
      const t = h.title as Record<string, unknown>;
      if (typeof t.content === 'string') parts.push(t.content);
      if (typeof t.tag === 'string' && t.tag === 'plain_text' && typeof t.content === 'string') {
        parts.push(t.content);
      }
    }
    if (typeof h.subtitle === 'object' && h.subtitle) {
      const s = h.subtitle as Record<string, unknown>;
      if (typeof s.content === 'string') parts.push(s.content);
    }
  }
  if (Array.isArray(c.elements)) {
    for (const el of c.elements) {
      parts.push(extractCardElementText(el));
    }
  }
  return parts.filter(Boolean).join('\n') || '[卡片消息]';
}

function extractCardElementText(el: unknown): string {
  if (!el || typeof el !== 'object') return '';
  const e = el as Record<string, unknown>;
  if (e.tag === 'div' && typeof e.text === 'object' && e.text) {
    const t = e.text as Record<string, unknown>;
    if (typeof t.content === 'string') return t.content;
  }
  if (e.tag === 'markdown' && typeof e.content === 'string') return e.content;
  if (e.tag === 'plain_text' && typeof e.content === 'string') return e.content;
  if (Array.isArray(e.actions)) {
    return e.actions.map(extractCardElementText).join(' ');
  }
  if (typeof e.text === 'string') return e.text;
  return '';
}

export function filterChats(chats: LarkChat[], filter: Config['larkChatFilter']): LarkChat[] {
  if (!filter || filter.mode === 'all') return chats;
  const allow = new Set(filter.allowlist || []);
  const block = new Set(filter.blocklist || []);
  if (filter.mode === 'allowlist') {
    return chats.filter((c) => allow.has(c.chat_id));
  }
  if (filter.mode === 'blocklist') {
    return chats.filter((c) => !block.has(c.chat_id));
  }
  return chats;
}

export async function syncLarkMessages(opts: {
  chatId?: string;
  daysBack?: number;
  onProgress?: (chatId: string, info: { phase: string; count: number }) => void;
} = {}): Promise<LarkSyncResult> {
  // 1. Global sync lock
  if (_syncRunning) {
    return {
      ok: false,
      error: '同步正在进行中，请稍后再试',
      synced: {},
    };
  }
  _syncRunning = true;

  const cfg = readConfig();
  const result: LarkSyncResult = { ok: true, synced: {} };

  try {
    const doctor = await larkDoctor();
    if (!doctor.ok || !doctor.authenticated) {
      return {
        ok: false,
        error: doctor.error || 'lark-cli 未配置或未登录，请先运行 lark-cli auth login --as user',
        synced: {},
      };
    }

    let chats: LarkChat[];
    if (opts.chatId) {
      chats = [{ chat_id: opts.chatId, name: opts.chatId }];
    } else {
      try {
        chats = await larkAllChats();
      } catch (e) {
        return {
          ok: false,
          error: e instanceof Error ? e.message : '获取飞书群列表失败',
          synced: {},
        };
      }
      chats = filterChats(chats, cfg.larkChatFilter);
    }

    const since = opts.daysBack
      ? new Date(Date.now() - opts.daysBack * 24 * 60 * 60 * 1000).toISOString()
      : undefined;

    // Parallel sync with concurrency limit
    const CONCURRENCY = 3;
    const queue = [...chats];
    const running = new Set<Promise<void>>();

    async function pumpQueue(): Promise<void> {
      while (queue.length > 0 && running.size < CONCURRENCY) {
        const chat = queue.shift()!;
        const p = syncSingleChat(chat, since, opts.onProgress, result, cfg).finally(() => {
          running.delete(p);
        });
        running.add(p);
      }
    }

    while (queue.length > 0 || running.size > 0) {
      await pumpQueue();
      if (running.size > 0) {
        await Promise.race(running);
      }
    }

    result.ok = Object.values(result.synced).every((s) => s.error === null);

    // Rebuild mentions index once at the end
    const anyInserted = Object.values(result.synced).some((s) => s.inserted > 0);
    if (anyInserted) {
      rebuildMentionIndexFromMessages();
      // Invalidate relevant caches
      cache.del(CK.mentions());
      cache.del(CK.sessions());
    }
  } finally {
    _syncRunning = false;
  }

  return result;
}

async function syncSingleChat(
  chat: LarkChat,
  since: string | undefined,
  onProgress: ((chatId: string, info: { phase: string; count: number }) => void) | undefined,
  result: LarkSyncResult,
  cfg: Config,
): Promise<void> {
  const cid = chat.chat_id;
  result.synced[cid] = { inserted: 0, skipped: 0, error: null };

  // Per-chat cooldown check
  const lastSync = _lastSyncByChat.get(cid) ?? 0;
  const now = Date.now();
  if (now - lastSync < CHAT_COOLDOWN_MS) {
    result.synced[cid].error = `冷却中，上次同步 ${Math.round((now - lastSync) / 1000)} 秒前`;
    return;
  }
  _lastSyncByChat.set(cid, now);

  // Load last sync timestamp for incremental sync
  const stateRow = db()
    .prepare('SELECT last_sync_time FROM sync_state WHERE chatroom_id = ? AND source = ?')
    .get(cid, 'lark') as { last_sync_time: string | null } | undefined;
  const startTime = stateRow?.last_sync_time || since;

  try {
    onProgress?.(cid, { phase: 'fetch', count: 0 });
    const messages = await larkAllMessages(cid, {
      start: startTime,
      maxPages: 4,
    });

    onProgress?.(cid, { phase: 'persist', count: messages.length });
    const { inserted, skipped, changedDates } = persistMessages(cid, messages);
    result.synced[cid].inserted = inserted;
    result.synced[cid].skipped = skipped;

    // Update sync_state
    const maxTime = messages.reduce((max, m) => {
      const t = m.create_time ? Number(m.create_time) : 0;
      return t > max ? t : max;
    }, 0);
    const lastSyncTime = maxTime
      ? new Date(maxTime).toISOString()
      : startTime || new Date().toISOString();

    const firstDate = messages.length
      ? formatDate(Math.min(...messages.map((m) => Number(m.create_time || Date.now()) / 1000)))
      : null;
    const lastDate = messages.length
      ? formatDate(Math.max(...messages.map((m) => Number(m.create_time || Date.now()) / 1000)))
      : null;

    const total = db()
      .prepare('SELECT COUNT(*) AS n FROM messages WHERE chatroom_id = ? AND source = ?')
      .get(cid, 'lark') as { n: number };

    db()
      .prepare(
        `INSERT INTO sync_state (
           chatroom_id, source, last_synced_at, first_message_date, last_message_date,
           total_messages, status, last_error, failed_chunks, empty_chunks, total_chunks, last_sync_time
         )
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
         ON CONFLICT(chatroom_id, source) DO UPDATE SET
           last_synced_at = excluded.last_synced_at,
           first_message_date = COALESCE(excluded.first_message_date, sync_state.first_message_date),
           last_message_date = COALESCE(excluded.last_message_date, sync_state.last_message_date),
           total_messages = excluded.total_messages,
           status = excluded.status,
           last_error = excluded.last_error,
           last_sync_time = excluded.last_sync_time`,
      )
      .run(
        cid,
        'lark',
        Date.now(),
        firstDate,
        lastDate,
        total.n,
        'ok',
        null,
        0,
        messages.length === 0 ? 1 : 0,
        1,
        lastSyncTime,
      );

    // Only aggregate daily_stats for changed dates
    if (changedDates.size > 0) {
      const dateList = Array.from(changedDates).sort();
      const buckets = aggregateDailyStats(cid, dateList);
      for (const b of buckets) {
        saveStats({
          chatroom_id: cid,
          date: b.date,
          total: b.total,
          top_senders: b.top_senders,
          by_hour: b.by_hour,
        });
      }
      // Invalidate stats cache for changed dates
      for (const d of changedDates) {
        cache.del(CK.stats('day', d));
        cache.del(CK.stats('week', d));
        cache.del(CK.stats('month', d));
      }
    }
  } catch (e) {
    const err = e instanceof Error ? e.message : 'unknown';
    result.synced[cid].error = err;
    db()
      .prepare(
        `INSERT INTO sync_state (chatroom_id, source, last_synced_at, status, last_error, total_messages)
         VALUES (?, ?, ?, ?, ?, ?)
         ON CONFLICT(chatroom_id, source) DO UPDATE SET
           last_synced_at = excluded.last_synced_at,
           status = 'failed',
           last_error = excluded.last_error`,
      )
      .run(cid, 'lark', Date.now(), 'failed', err, 0);
  }
}

function persistMessages(
  chatId: string,
  messages: LarkMessage[],
): { inserted: number; skipped: number; changedDates: Set<string> } {
  if (messages.length === 0) return { inserted: 0, skipped: 0, changedDates: new Set() };

  const insertStmt = db().prepare(
    `INSERT OR IGNORE INTO messages
       (chatroom_id, local_id, sender, sender_name, content, time, timestamp, type, date, source, raw)
     VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
  );

  let inserted = 0;
  let skipped = 0;
  const changedDates = new Set<string>();

  // Batch process in chunks to avoid long transactions
  const BATCH_SIZE = 500;
  for (let i = 0; i < messages.length; i += BATCH_SIZE) {
    const batch = messages.slice(i, i + BATCH_SIZE);

    const tx = db().transaction((msgs: LarkMessage[]) => {
      for (const m of msgs) {
        if (!m.message_id) continue;

        const cacheKey = `${chatId}#${m.message_id}`;
        if (_messageIdCache.has(cacheKey)) {
          skipped++;
          continue;
        }

        const row = larkMessageToRow(chatId, m);
        const r = insertStmt.run(
          row.chatroom_id,
          row.local_id,
          row.sender,
          row.sender_name,
          row.content,
          row.time,
          row.timestamp,
          row.type,
          row.date,
          row.source,
          row.raw,
        );
        if (r.changes > 0) {
          inserted++;
          changedDates.add(row.date);
          upsertLinksForMessage({
            chatroom_id: row.chatroom_id,
            local_id: row.local_id,
            sender: row.sender_name || row.sender,
            content: row.content,
            time: row.time,
            timestamp: row.timestamp,
            date: row.date,
          });

          _messageIdCache.add(cacheKey);
          if (_messageIdCache.size > MAX_CACHE_SIZE) {
            const toDelete = Math.floor(MAX_CACHE_SIZE / 2);
            const iter = _messageIdCache.values();
            for (let i = 0; i < toDelete; i++) {
              const val = iter.next().value;
              if (val) _messageIdCache.delete(val);
            }
          }
        } else {
          skipped++;
          _messageIdCache.add(cacheKey);
        }
      }
    });

    tx(batch);
  }

  return { inserted, skipped, changedDates };
}
