import pLimit from 'p-limit';
import { db } from './db';
import {
  aggregateDailyStats,
  bulkInsertMessages,
  upsertSyncState,
} from './messages-store';
import { rebuildMentionIndexFromMessages } from './mentions';

export type StatsRow = {
  chatroom_id: string;
  date: string;
  total: number;
  top_senders: Array<{ sender: string; count: number }>;
  by_hour: Array<{ hour: number; count: number }>;
};

export function getCachedStats(chatroomId: string, date: string): StatsRow | null {
  const row = db()
    .prepare(
      'SELECT chatroom_id, date, total, top_senders, by_hour FROM daily_stats WHERE chatroom_id = ? AND date = ?',
    )
    .get(chatroomId, date) as
    | {
        chatroom_id: string;
        date: string;
        total: number;
        top_senders: string;
        by_hour: string;
      }
    | undefined;
  if (!row) return null;
  return {
    chatroom_id: row.chatroom_id,
    date: row.date,
    total: row.total,
    top_senders: JSON.parse(row.top_senders),
    by_hour: JSON.parse(row.by_hour),
  };
}

export function listCachedStatsForDate(date: string): StatsRow[] {
  const rows = db()
    .prepare(
      'SELECT chatroom_id, date, total, top_senders, by_hour FROM daily_stats WHERE date = ? ORDER BY total DESC',
    )
    .all(date) as Array<{
    chatroom_id: string;
    date: string;
    total: number;
    top_senders: string;
    by_hour: string;
  }>;
  return rows.map((r) => ({
    chatroom_id: r.chatroom_id,
    date: r.date,
    total: r.total,
    top_senders: JSON.parse(r.top_senders),
    by_hour: JSON.parse(r.by_hour),
  }));
}

export function listCachedStatsRange(since: string, until: string): StatsRow[] {
  const rows = db()
    .prepare(
      'SELECT chatroom_id, date, total, top_senders, by_hour FROM daily_stats WHERE date >= ? AND date <= ?',
    )
    .all(since, until) as Array<{
    chatroom_id: string;
    date: string;
    total: number;
    top_senders: string;
    by_hour: string;
  }>;
  return rows.map((r) => ({
    chatroom_id: r.chatroom_id,
    date: r.date,
    total: r.total,
    top_senders: JSON.parse(r.top_senders),
    by_hour: JSON.parse(r.by_hour),
  }));
}

const upsert = () =>
  db().prepare(`
    INSERT INTO daily_stats (chatroom_id, date, total, top_senders, by_hour, refreshed_at)
    VALUES (?, ?, ?, ?, ?, ?)
    ON CONFLICT(chatroom_id, date) DO UPDATE SET
      total = excluded.total,
      top_senders = excluded.top_senders,
      by_hour = excluded.by_hour,
      refreshed_at = excluded.refreshed_at
  `);

export function saveStats(row: StatsRow & { refreshed_at?: number }) {
  upsert().run(
    row.chatroom_id,
    row.date,
    row.total,
    JSON.stringify(row.top_senders),
    JSON.stringify(row.by_hour),
    row.refreshed_at ?? Date.now(),
  );
}

export interface RescanProgress {
  type: 'progress' | 'done' | 'error' | 'start';
  done: number;
  total: number;
  current?: string;
  error?: string;
  inserted_messages?: number;
}

export interface RescanTarget {
  chatroomId: string;
  display: string;
}

export interface SyncOptions {
  targets: RescanTarget[];
  since: string;
  until: string;
  concurrency?: number;
  onProgress?: (p: RescanProgress) => void;
}

// Helper: split a date range into month chunks ([{since, until}, ...])
function monthChunks(since: string, until: string): Array<{ since: string; until: string }> {
  const chunks: Array<{ since: string; until: string }> = [];
  const start = new Date(since);
  const end = new Date(until);
  let cur = new Date(start.getFullYear(), start.getMonth(), 1);
  while (cur <= end) {
    const chunkStart = cur < start ? start : cur;
    const nextMonth = new Date(cur.getFullYear(), cur.getMonth() + 1, 0); // last day of cur month
    const chunkEnd = nextMonth > end ? end : nextMonth;
    chunks.push({
      since: ymd(chunkStart),
      until: ymd(chunkEnd),
    });
    cur = new Date(cur.getFullYear(), cur.getMonth() + 1, 1);
  }
  return chunks;
}

function ymd(d: Date): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${y}-${m}-${day}`;
}

function dateList(since: string, until: string): string[] {
  const out: string[] = [];
  const start = new Date(since);
  const end = new Date(until);
  for (let d = new Date(start); d <= end; d.setDate(d.getDate() + 1)) {
    out.push(ymd(d));
  }
  return out;
}

/**
 * 全量同步：每群按月分批拉取历史消息 → 本地存 messages → 本地聚合 daily_stats。
 * 注：仅保留本地 DB 聚合逻辑供 lark 同步后使用。
 */
export async function syncFullHistory({
  targets,
  since,
  until,
  concurrency = 6,
  onProgress,
}: SyncOptions): Promise<{ ok: number; failed: number; messages: number }> {
  const limit = pLimit(concurrency);
  const chunks = monthChunks(since, until);
  const total = targets.length * chunks.length;
  let done = 0;
  let ok = 0;
  let failed = 0;
  let totalMessages = 0;
  const byTarget = new Map<
    string,
    {
      fetched: number;
      inserted: number;
      failedChunks: number;
      emptyChunks: number;
      errors: string[];
    }
  >();
  for (const t of targets) {
    byTarget.set(t.chatroomId, {
      fetched: 0,
      inserted: 0,
      failedChunks: 0,
      emptyChunks: 0,
      errors: [],
    });
  }

  const tasks: Promise<void>[] = [];
  for (const t of targets) {
    for (const c of chunks) {
      tasks.push(
        limit(async () => {
          const state = byTarget.get(t.chatroomId)!;
          try {
            // 本地 DB 聚合模式：从已有 messages 中聚合
            const messages: Array<{
              local_id: string | number;
              sender: string;
              content: string;
              time: string;
              timestamp: number;
              type: string;
            }> = db()
              .prepare(
                `SELECT local_id, sender, content, time, timestamp, type
                 FROM messages
                 WHERE chatroom_id = ? AND date(time) >= ? AND date(time) <= ?`,
              )
              .all(t.chatroomId, c.since, c.until) as any[];

            const inserted = bulkInsertMessages(
              t.chatroomId,
              messages.map((m) => ({
                ...m,
                username: t.chatroomId,
                chat: t.display,
              })),
            );
            state.fetched += messages.length;
            state.inserted += inserted;
            if (messages.length === 0) state.emptyChunks++;
            totalMessages += inserted;
            ok++;
          } catch (e) {
            const message = e instanceof Error ? e.message : String(e);
            state.failedChunks++;
            state.errors.push(`${c.since}~${c.until}: ${message}`);
            failed++;
            onProgress?.({
              type: 'error',
              done,
              total,
              current: `${t.display} ${c.since.slice(0, 7)}`,
              error: message,
              inserted_messages: totalMessages,
            });
          } finally {
            done++;
            onProgress?.({
              type: 'progress',
              done,
              total,
              current: `${t.display} ${c.since.slice(0, 7)}`,
              inserted_messages: totalMessages,
            });
          }
        }),
      );
    }
  }

  await Promise.all(tasks);

  // Now aggregate daily_stats from the messages for each target
  const aggLimit = pLimit(8);
  const dates = dateList(since, until);
  await Promise.all(
    targets.map((t) =>
      aggLimit(async () => {
        const buckets = aggregateDailyStats(t.chatroomId, dates);
        for (const b of buckets) {
          if (b.total === 0) {
            const existing = getCachedStats(t.chatroomId, b.date);
            if (existing && existing.total > 0) continue;
          }
          saveStats({
            chatroom_id: t.chatroomId,
            date: b.date,
            total: b.total,
            top_senders: b.top_senders,
            by_hour: b.by_hour,
          });
        }

        // Update sync_state
        const firstRow = db()
          .prepare(
            'SELECT MIN(date) AS d, MAX(date) AS dx, COUNT(*) AS n FROM messages WHERE chatroom_id = ?',
          )
          .get(t.chatroomId) as { d: string | null; dx: string | null; n: number };
        const state = byTarget.get(t.chatroomId)!;
        const status =
          state.failedChunks === chunks.length
            ? 'failed'
            : state.failedChunks > 0
              ? 'partial'
              : firstRow.n === 0 && state.fetched === 0
                ? 'empty'
                : 'ok';
        upsertSyncState(t.chatroomId, firstRow.n, firstRow.d, firstRow.dx, {
          status,
          lastError: state.errors.slice(-3).join('\n') || null,
          failedChunks: state.failedChunks,
          emptyChunks: state.emptyChunks,
          totalChunks: chunks.length,
        });
      }),
    ),
  );

  rebuildMentionIndexFromMessages();

  onProgress?.({
    type: 'done',
    done: total,
    total,
    inserted_messages: totalMessages,
  });

  return { ok, failed, messages: totalMessages };
}

/**
 * 兼容旧调用：单天 stats 模式（本地 DB 聚合）
 */
export interface RescanOptions {
  targets: RescanTarget[];
  dates: string[];
  concurrency?: number;
  onProgress?: (p: RescanProgress) => void;
}

export async function rescan({
  targets,
  dates,
  concurrency = 5,
  onProgress,
}: RescanOptions): Promise<{ ok: number; failed: number }> {
  const limit = pLimit(concurrency);
  const total = targets.length * dates.length;
  let done = 0;
  let ok = 0;
  let failed = 0;

  const tasks: Promise<void>[] = [];
  for (const t of targets) {
    for (const d of dates) {
      tasks.push(
        limit(async () => {
          try {
            const messages = db()
              .prepare(
                `SELECT sender, COUNT(*) as count FROM messages
                 WHERE chatroom_id = ? AND date(time) = ?
                 GROUP BY sender ORDER BY count DESC`,
              )
              .all(t.chatroomId, d) as Array<{ sender: string; count: number }>;

            const byHour = db()
              .prepare(
                `SELECT CAST(strftime('%H', time) AS INTEGER) as hour, COUNT(*) as count
                 FROM messages WHERE chatroom_id = ? AND date(time) = ?
                 GROUP BY hour ORDER BY hour`,
              )
              .all(t.chatroomId, d) as Array<{ hour: number; count: number }>;

            saveStats({
              chatroom_id: t.chatroomId,
              date: d,
              total: messages.reduce((sum, m) => sum + m.count, 0),
              top_senders: messages.slice(0, 10).map((m) => ({ sender: m.sender, count: m.count })),
              by_hour: byHour.map((h) => ({ hour: h.hour, count: h.count })),
            });
            ok++;
          } catch {
            failed++;
            saveStats({
              chatroom_id: t.chatroomId,
              date: d,
              total: 0,
              top_senders: [],
              by_hour: [],
            });
          } finally {
            done++;
            onProgress?.({ type: 'progress', done, total, current: t.display });
          }
        }),
      );
    }
  }

  await Promise.all(tasks);
  onProgress?.({ type: 'done', done, total });
  return { ok, failed };
}
