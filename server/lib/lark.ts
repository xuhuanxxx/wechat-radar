import { execFile } from 'node:child_process';
import { promisify } from 'node:util';

const run = promisify(execFile);

const DEFAULT_OPTS = {
  maxBuffer: 64 * 1024 * 1024,
  timeout: 60_000,
} as const;

export interface LarkChat {
  chat_id: string;
  name: string;
  description?: string;
  chat_type?: string;
  chat_mode?: string;
  user_count?: string;
  add_member_permission?: string;
  // CLI pretty/table output may use camelCase or snake_case
  memberCount?: number;
  chatId?: string;
}

export interface LarkSender {
  id: string;
  id_type?: string;
  name?: string;
  // pretty output sometimes flattens
  sender_id?: string;
  sender_name?: string;
}

export interface LarkMessage {
  message_id: string;
  chat_id?: string;
  sender?: LarkSender;
  create_time?: string; // ms timestamp as string
  update_time?: string;
  msg_type?: string;
  content?: string; // JSON string for text/post/card
  body?: {
    content?: string;
  };
  mentions?: Array<{
    key: string;
    id: { open_id: string; user_id?: string };
    name: string;
    tenant_key?: string;
  }>;
  parent_id?: string;
  thread_id?: string;
  // pretty output may flatten
  messageId?: string;
  msgType?: string;
  createTime?: string;
}

export interface LarkChatListResponse {
  ok: boolean;
  data?: {
    items?: LarkChat[];
    chats?: LarkChat[];
    has_more?: boolean;
    page_token?: string;
  };
  chats?: LarkChat[];
  error?: { type: string; message: string };
}

export interface LarkMessagesResponse {
  ok: boolean;
  data?: {
    items?: LarkMessage[];
    messages?: LarkMessage[];
    has_more?: boolean;
    page_token?: string;
  };
  messages?: LarkMessage[];
  error?: { type: string; message: string };
}

async function larkRaw(args: string[], opts = DEFAULT_OPTS): Promise<string> {
  const { stdout } = await run('lark-cli', args, opts);
  return stdout;
}

async function larkJson<T>(args: string[], opts = DEFAULT_OPTS): Promise<T> {
  const stdout = await larkRaw([...args, '--json'], opts);
  return JSON.parse(stdout) as T;
}

export async function larkAvailable(): Promise<boolean> {
  try {
    await run('lark-cli', ['--version'], { timeout: 5_000 });
    return true;
  } catch {
    return false;
  }
}

export async function larkDoctor(): Promise<{
  ok: boolean;
  configured: boolean;
  authenticated: boolean;
  identity?: string;
  error?: string;
}> {
  try {
    const out = await larkRaw(['doctor']);
    const parsed = JSON.parse(out) as {
      ok?: boolean;
      checks?: Array<{ name: string; status: string; message?: string }>;
    };
    const checks = parsed.checks ?? [];
    const configOk = checks.some((c) => c.name === 'config_file' && c.status === 'pass');
    const appOk = checks.some((c) => c.name === 'app_resolved' && c.status === 'pass');
    const userOk = checks.some((c) => c.name === 'user_identity' && c.status === 'pass');
    return {
      ok: parsed.ok ?? false,
      configured: configOk && appOk,
      authenticated: userOk,
      error: parsed.ok ? undefined : out.slice(0, 200),
    };
  } catch (e) {
    return {
      ok: false,
      configured: false,
      authenticated: false,
      error: e instanceof Error ? e.message : 'unknown',
    };
  }
}

export async function larkChatList(pageToken?: string, pageSize = 100): Promise<LarkChat[]> {
  const args = ['im', '+chat-list', '--as', 'user', '--page-size', String(pageSize)];
  if (pageToken) args.push('--page-token', pageToken);
  const res = await larkJson<LarkChatListResponse>(args);
  if (!res.ok && res.error) {
    throw new Error(`lark chat-list failed: ${res.error.type} ${res.error.message}`);
  }
  const items = (res.data?.items ?? res.data?.chats ?? res.chats ?? []) as LarkChat[];
  // Normalize field names
  return items.map((c) => ({
    chat_id: c.chat_id || c.chatId || '',
    name: c.name || '未命名群',
    description: c.description,
    chat_type: c.chat_type || c.chat_mode,
    user_count: c.user_count,
    memberCount: c.memberCount,
  }));
}

export async function larkChatMessages(
  chatId: string,
  opts: {
    start?: string; // ISO8601
    end?: string; // ISO8601
    pageToken?: string;
    pageSize?: number;
    sort?: 'asc' | 'desc';
  } = {},
): Promise<{ messages: LarkMessage[]; hasMore: boolean; pageToken?: string }> {
  const args = [
    'im',
    '+chat-messages-list',
    '--as',
    'user',
    '--chat-id',
    chatId,
    '--page-size',
    String(opts.pageSize ?? 50),
    '--sort',
    opts.sort ?? 'desc',
    '--no-reactions',
  ];
  if (opts.start) args.push('--start', opts.start);
  if (opts.end) args.push('--end', opts.end);
  if (opts.pageToken) args.push('--page-token', opts.pageToken);

  const res = await larkJson<LarkMessagesResponse>(args);
  if (!res.ok && res.error) {
    throw new Error(`lark chat-messages-list failed: ${res.error.type} ${res.error.message}`);
  }
  const items = (res.data?.items ?? res.data?.messages ?? res.messages ?? []) as LarkMessage[];
  return {
    messages: items.map(normalizeMessage),
    hasMore: res.data?.has_more ?? false,
    pageToken: res.data?.page_token,
  };
}

function normalizeMessage(m: LarkMessage): LarkMessage {
  return {
    ...m,
    message_id: m.message_id || m.messageId || '',
    msg_type: m.msg_type || m.msgType || 'unknown',
    create_time: m.create_time || m.createTime,
    sender: m.sender
      ? {
          id: m.sender.id || (m.sender as unknown as { sender_id?: string }).sender_id || '',
          name: m.sender.name || (m.sender as unknown as { sender_name?: string }).sender_name || '',
        }
      : {
          id: (m as unknown as { sender_id?: string }).sender_id || '',
          name: (m as unknown as { sender_name?: string }).sender_name || '',
        },
  };
}

export async function larkAllChats(): Promise<LarkChat[]> {
  const out: LarkChat[] = [];
  let pageToken: string | undefined;
  do {
    const batch = await larkChatList(pageToken, 100);
    out.push(...batch);
    pageToken = batch.length === 100 ? undefined : undefined; // lark-cli shortcut may not expose page_token; stop after one batch for MVP
  } while (pageToken);
  return out;
}

export async function larkAllMessages(
  chatId: string,
  opts: { start?: string; end?: string; maxPages?: number } = {},
): Promise<LarkMessage[]> {
  const out: LarkMessage[] = [];
  let pageToken: string | undefined;
  let pages = 0;
  do {
    const batch = await larkChatMessages(chatId, {
      ...opts,
      pageToken,
      pageSize: 50,
      sort: 'desc',
    });
    out.push(...batch.messages);
    pageToken = batch.pageToken;
    pages++;
  } while (pageToken && pages < (opts.maxPages ?? 4));
  return out;
}
