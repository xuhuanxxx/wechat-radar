import { db } from './db';
import { upsertLinksForMessage } from './message-links';

export interface MessageRow {
  chatroom_id: string;
  local_id: number | string;
  sender: string;
  sender_name?: string;
  content: string;
  time: string;
  timestamp: number;
  type: string;
  date: string;
  source?: string;
  raw?: string;
}

const SYSTEM_TYPES = new Set(['系统', 'system']);
const REVOKE_RE = /撤回了一条消息|recalled a message/i;

export function dateOfMessage(m: Pick<MessageRow, 'time' | 'timestamp'>): string {
  if (m.time && m.time.length >= 10) return m.time.slice(0, 10);
  if (m.timestamp) {
    const d = new Date(m.timestamp * 1000);
    const y = d.getFullYear();
    const mo = String(d.getMonth() + 1).padStart(2, '0');
    const da = String(d.getDate()).padStart(2, '0');
    return `${y}-${mo}-${da}`;
  }
  return 'unknown';
}

export function bulkInsertMessages(chatroomId: string, messages: Array<Partial<MessageRow>>): number {
  if (messages.length === 0) return 0;
  const stmt = db().prepare(`
    INSERT OR IGNORE INTO messages
      (chatroom_id, local_id, sender, sender_name, content, time, timestamp, type, date, source, raw)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
  `);
  let inserted = 0;
  const tx = db().transaction((msgs: typeof messages) => {
    for (const m of msgs) {
      if (SYSTEM_TYPES.has(m.type ?? '') && REVOKE_RE.test(m.content ?? '')) continue;
      const r = stmt.run(
        chatroomId,
        String(m.local_id ?? ''),
        m.sender ?? '',
        m.sender_name ?? null,
        m.content ?? '',
        m.time ?? '',
        m.timestamp ?? 0,
        m.type ?? '',
        dateOfMessage(m as MessageRow),
        m.source ?? 'lark',
        m.raw ?? null,
      );
      upsertLinksForMessage({
        chatroom_id: chatroomId,
        local_id: m.local_id ?? '',
        sender: m.sender ?? '',
        content: m.content ?? '',
        time: m.time ?? '',
        timestamp: m.timestamp ?? 0,
        date: dateOfMessage(m as MessageRow),
      });
      if (r.changes > 0) inserted++;
    }
  });
  tx(messages);
  return inserted;
}

export function listMessagesForDate(chatroomId: string, date: string, limit = 1000): MessageRow[] {
  return db()
    .prepare(
      `SELECT chatroom_id, local_id, sender, content, time, timestamp, type, date
       FROM messages
       WHERE chatroom_id = ? AND date = ?
       ORDER BY timestamp ASC, local_id ASC
       LIMIT ?`,
    )
    .all(chatroomId, date, limit) as MessageRow[];
}

export interface DailyStatsAggregate {
  date: string;
  total: number;
  by_hour: Array<{ hour: number; count: number }>;
  top_senders: Array<{ sender: string; count: number }>;
}

export function aggregateDailyStats(chatroomId: string, dates: string[]): DailyStatsAggregate[] {
  if (dates.length === 0) return [];
  const placeholders = dates.map(() => '?').join(',');
  const rows = db()
    .prepare(
      `SELECT date(time) AS date,
              COUNT(*) AS total,
              CAST(strftime('%H', time) AS INTEGER) AS hour,
              sender,
              COUNT(*) AS sender_count
       FROM messages
       WHERE chatroom_id = ? AND date(time) IN (${placeholders})
       GROUP BY date(time), hour, sender
       ORDER BY date(time), hour, sender_count DESC`,
    )
    .all(chatroomId, ...dates) as Array<{
    date: string;
    total: number;
    hour: number;
    sender: string;
    sender_count: number;
  }>;

  const byDate = new Map<string, DailyStatsAggregate>();
  for (const r of rows) {
    if (!byDate.has(r.date)) {
      byDate.set(r.date, { date: r.date, total: 0, by_hour: [], top_senders: [] });
    }
    const agg = byDate.get(r.date)!;
    agg.total = r.total;
    const hourEntry = agg.by_hour.find((h) => h.hour === r.hour);
    if (hourEntry) {
      hourEntry.count += r.sender_count;
    } else {
      agg.by_hour.push({ hour: r.hour, count: r.sender_count });
    }
    const senderEntry = agg.top_senders.find((s) => s.sender === r.sender);
    if (senderEntry) {
      senderEntry.count += r.sender_count;
    } else {
      agg.top_senders.push({ sender: r.sender, count: r.sender_count });
    }
  }

  // Sort by hour and limit top senders
  for (const agg of byDate.values()) {
    agg.by_hour.sort((a, b) => a.hour - b.hour);
    agg.top_senders.sort((a, b) => b.count - a.count);
    agg.top_senders = agg.top_senders.slice(0, 10);
  }

  return dates.map((d) => byDate.get(d) || { date: d, total: 0, by_hour: [], top_senders: [] });
}

export function getSyncState(chatroomId: string) {
  return db()
    .prepare(
      'SELECT chatroom_id, message_count, first_date, last_date, meta, updated_at FROM sync_state WHERE chatroom_id = ?',
    )
    .get(chatroomId) as
    | {
        chatroom_id: string;
        message_count: number;
        first_date: string | null;
        last_date: string | null;
        meta: string | null;
        updated_at: number;
      }
    | undefined;
}

export function listAllSyncedDates(chatroomId: string): string[] {
  const rows = db()
    .prepare(
      'SELECT DISTINCT date FROM messages WHERE chatroom_id = ? ORDER BY date ASC',
    )
    .all(chatroomId) as Array<{ date: string }>;
  return rows.map((r) => r.date);
}

export function upsertSyncState(
  chatroomId: string,
  messageCount: number,
  firstDate: string | null,
  lastDate: string | null,
  meta?: Record<string, unknown>,
) {
  db()
    .prepare(
      `INSERT INTO sync_state (chatroom_id, message_count, first_date, last_date, meta, updated_at)
       VALUES (?, ?, ?, ?, ?, ?)
       ON CONFLICT(chatroom_id, source) DO UPDATE SET
         message_count = excluded.message_count,
         first_date = excluded.first_date,
         last_date = excluded.last_date,
         meta = excluded.meta,
         updated_at = excluded.updated_at`,
    )
    .run(
      chatroomId,
      messageCount,
      firstDate,
      lastDate,
      meta ? JSON.stringify(meta) : null,
      Date.now(),
    );
}
