import { createServer, IncomingMessage, ServerResponse } from 'node:http';
import { URL } from 'node:url';
import { readConfig } from './lib/config';
import { cacheStats } from './lib/cache';

// Route handlers
import { handleSync, handleSyncStream } from './routes/sync';
import { handleStats } from './routes/stats';
import { handleSessions } from './routes/sessions';
import { handleGroupDetail } from './routes/messages';
import { handleMentions } from './routes/mentions';
import { handleTopics, handleTopicDetail, handleTopicBuild } from './routes/topics';
import { handleLinks } from './routes/links';
import { handleSearch } from './routes/search';
import { handleGroups, handleGroupTags, handleAIClassify } from './routes/groups';
import { handleSetup } from './routes/setup';
import { handleLarkChats, handleLarkFilter } from './routes/lark';
import { handleDates } from './routes/dates';
import { handleDbInfo } from './routes/dbinfo';
import { handleMessageLinksRaw, handleMessageLinksResolve, handleMessageLinksBackfill } from './routes/message-links';

const cfg = readConfig();
const PORT = cfg.port || 3456;

const ROUTES: Array<{
  method: string;
  pattern: RegExp;
  handler: (req: IncomingMessage, res: ServerResponse, params: Record<string, string>, searchParams: URLSearchParams) => Promise<void> | void;
  stream?: boolean;
}> = [
  { method: 'POST', pattern: /^\/api\/lark\/sync$/, handler: handleSync },
  { method: 'GET', pattern: /^\/api\/stats$/, handler: handleStats },
  { method: 'GET', pattern: /^\/api\/sessions$/, handler: handleSessions },
  { method: 'GET', pattern: /^\/api\/group\/([^/]+)$/, handler: handleGroupDetail },
  { method: 'GET', pattern: /^\/api\/mentions$/, handler: handleMentions },
  { method: 'POST', pattern: /^\/api\/mentions$/, handler: handleMentions },
  { method: 'GET', pattern: /^\/api\/topics$/, handler: handleTopics },
  { method: 'GET', pattern: /^\/api\/topics\/([^/]+)$/, handler: handleTopicDetail },
  { method: 'POST', pattern: /^\/api\/topics\/build$/, handler: handleTopicBuild },
  { method: 'GET', pattern: /^\/api\/topics\/links$/, handler: handleLinks },
  { method: 'GET', pattern: /^\/api\/search$/, handler: handleSearch },
  { method: 'GET', pattern: /^\/api\/groups$/, handler: handleGroups },
  { method: 'POST', pattern: /^\/api\/groups$/, handler: handleGroups },
  { method: 'DELETE', pattern: /^\/api\/groups$/, handler: handleGroups },
  { method: 'GET', pattern: /^\/api\/group-tags$/, handler: handleGroupTags },
  { method: 'POST', pattern: /^\/api\/group-tags$/, handler: handleGroupTags },
  { method: 'GET', pattern: /^\/api\/ai-classify$/, handler: handleAIClassify },
  { method: 'POST', pattern: /^\/api\/ai-classify$/, handler: handleAIClassify },
  { method: 'GET', pattern: /^\/api\/setup$/, handler: handleSetup },
  { method: 'POST', pattern: /^\/api\/setup$/, handler: handleSetup },
  { method: 'GET', pattern: /^\/api\/lark\/chats$/, handler: handleLarkChats },
  { method: 'GET', pattern: /^\/api\/lark\/filter$/, handler: handleLarkFilter },
  { method: 'POST', pattern: /^\/api\/lark\/filter$/, handler: handleLarkFilter },
  { method: 'GET', pattern: /^\/api\/dates$/, handler: handleDates },
  { method: 'GET', pattern: /^\/api\/dbinfo$/, handler: handleDbInfo },
  { method: 'GET', pattern: /^\/api\/message-links\/raw$/, handler: handleMessageLinksRaw },
  { method: 'POST', pattern: /^\/api\/message-links\/resolve$/, handler: handleMessageLinksResolve },
  { method: 'POST', pattern: /^\/api\/message-links\/backfill$/, handler: handleMessageLinksBackfill },
];

function parseBody(req: IncomingMessage): Promise<unknown> {
  return new Promise((resolve) => {
    let data = '';
    req.on('data', (chunk) => { data += chunk; });
    req.on('end', () => {
      try { resolve(JSON.parse(data)); } catch { resolve({}); }
    });
    req.on('error', () => resolve({}));
  });
}

function setCors(res: ServerResponse) {
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Methods', 'GET, POST, DELETE, OPTIONS');
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type');
}

const server = createServer(async (req, res) => {
  setCors(res);

  if (req.method === 'OPTIONS') {
    res.writeHead(204);
    res.end();
    return;
  }

  const url = new URL(req.url || '/', `http://${req.headers.host}`);
  const pathname = url.pathname;

  // Health check
  if (pathname === '/health') {
    res.writeHead(200, { 'content-type': 'application/json' });
    res.end(JSON.stringify({ ok: true, cache: cacheStats() }));
    return;
  }

  const route = ROUTES.find((r) => r.method === req.method && r.pattern.test(pathname));
  if (!route) {
    res.writeHead(404, { 'content-type': 'application/json' });
    res.end(JSON.stringify({ ok: false, error: 'not found' }));
    return;
  }

  // Extract path params
  const match = pathname.match(route.pattern);
  const params: Record<string, string> = {};
  if (match && match[1]) params.id = decodeURIComponent(match[1]);

  try {
    // Attach parsed body for POST handlers
    if (req.method === 'POST' || req.method === 'DELETE') {
      (req as any).body = await parseBody(req);
    }
    await route.handler(req, res, params, url.searchParams);
  } catch (e) {
    console.error(`[server] ${req.method} ${pathname} error:`, e);
    if (!res.headersSent) {
      res.writeHead(500, { 'content-type': 'application/json' });
      res.end(JSON.stringify({ ok: false, error: e instanceof Error ? e.message : 'unknown' }));
    }
  }
});

server.listen(PORT, () => {
  console.log(`[LarkRadar Data Service] listening on http://localhost:${PORT}`);
});
