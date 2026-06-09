import { db } from './db';
import { readConfig } from './config';
import { larkAllChats } from './lark';
import { cache, CK } from './cache';

export interface Session {
  chat: string;
  chat_type: 'private' | 'group';
  is_group: boolean;
  last_msg_type: string;
  last_sender: string;
  summary: string;
  time: string;
  timestamp: number;
  unread: number;
  username: string;
}

export async function listLocalSessionsFallback(limit = 500): Promise<Session[]> {
  const rows = db()
    .prepare(
      `
      SELECT m.chatroom_id, m.sender, m.sender_name, m.content, m.time, m.timestamp, m.type
      FROM messages m
      JOIN (
        SELECT chatroom_id, MAX(timestamp) AS timestamp
        FROM messages
        GROUP BY chatroom_id
      ) latest
        ON latest.chatroom_id = m.chatroom_id
       AND latest.timestamp = m.timestamp
      GROUP BY m.chatroom_id
      ORDER BY m.timestamp DESC
      LIMIT ?
    `,
    )
    .all(limit) as Array<{
    chatroom_id: string;
    sender: string;
    sender_name: string | null;
    content: string;
    time: string;
    timestamp: number;
    type: string;
  }>;

  const cfg = readConfig();
  const nameMap = new Map<string, string>();
  if (cfg.source === 'lark') {
    try {
      const chats = await larkAllChats();
      for (const c of chats) nameMap.set(c.chat_id, c.name);
    } catch {
      // ignore
    }
  }

  return rows.map((r) => ({
    chat: nameMap.get(r.chatroom_id) || r.chatroom_id,
    chat_type: 'group',
    is_group: true,
    last_msg_type: r.type,
    last_sender: r.sender_name || r.sender,
    summary: r.content,
    time: r.time,
    timestamp: r.timestamp,
    unread: 0,
    username: r.chatroom_id,
  }));
}

export async function loadSessionsSafe(limit = 500): Promise<Session[]> {
  const cfg = readConfig();
  if (cfg.demoMode || cfg.source === 'lark') {
    return await listLocalSessionsFallback(limit);
  }
  // Default to local DB fallback for any non-lark source
  return listLocalSessionsFallback(limit);
}
