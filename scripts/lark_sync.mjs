#!/usr/bin/env node
// Lark sync CLI script.
// Can be run manually or scheduled via cron.
// Environment:
//   WECHAT_RADAR_LARK_API_URL - base URL of the running Next.js app (default: http://localhost:3000)
//   WECHAT_RADAR_LARK_CHAT_ID - optional, sync only one chat
//   WECHAT_RADAR_LARK_DAYS_BACK - how many days back to fetch on first sync (default: 7)

async function main() {
  const baseUrl = process.env.WECHAT_RADAR_LARK_API_URL || 'http://localhost:3000';
  const chatId = process.env.WECHAT_RADAR_LARK_CHAT_ID || undefined;
  const daysBack = Number(process.env.WECHAT_RADAR_LARK_DAYS_BACK || 7);

  console.log(`[${new Date().toISOString()}] Lark sync started -> ${baseUrl}/api/lark/sync`);

  const res = await fetch(`${baseUrl}/api/lark/sync`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({
      chat_id: chatId,
      days_back: Number.isFinite(daysBack) && daysBack > 0 ? daysBack : 7,
    }),
  });

  const text = await res.text();
  let json;
  try {
    json = JSON.parse(text);
  } catch {
    json = { ok: false, error: `Non-JSON response (${res.status}): ${text.slice(0, 200)}` };
  }

  console.log(`[${new Date().toISOString()}] Lark sync finished: status=${res.status} ok=${json.ok}`);
  if (json.synced) {
    for (const [cid, meta] of Object.entries(json.synced)) {
      console.log(`  ${cid}: inserted=${meta.inserted} skipped=${meta.skipped} error=${meta.error ?? 'none'}`);
    }
  }
  if (!res.ok || !json.ok) {
    console.error(`Error: ${json.error ?? 'unknown'}`);
    process.exit(1);
  }
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
