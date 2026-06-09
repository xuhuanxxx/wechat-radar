import { NextResponse } from 'next/server';
import { larkAllChats, larkDoctor, larkAvailable } from '@/lib/lark';
import { readConfig } from '@/lib/config';
import { withCache, CK } from '@/lib/cache';

export const dynamic = 'force-dynamic';

export async function GET() {
  const available = await larkAvailable();
  if (!available) {
    return NextResponse.json(
      { ok: false, error: 'lark-cli 未安装', available: false },
      { status: 503 },
    );
  }

  const doctor = await larkDoctor();
  if (!doctor.authenticated) {
    return NextResponse.json(
      { ok: false, error: doctor.error || 'lark-cli 未登录', available: true, authenticated: false },
      { status: 401 },
    );
  }

  try {
    const result = await withCache(CK.larkChats(), 60, async () => {
      const chats = await larkAllChats();
      const cfg = readConfig();
      const filter = cfg.larkChatFilter;
      const enriched = chats.map((c) => ({
        id: c.chat_id,
        name: c.name,
        member_count: Number(c.user_count || c.memberCount || 0),
        filtered:
          filter?.mode === 'allowlist'
            ? filter.allowlist.includes(c.chat_id)
            : filter?.mode === 'blocklist'
              ? !filter.blocklist.includes(c.chat_id)
              : true,
      }));
      return { ok: true, chats: enriched, filter };
    });
    return NextResponse.json(result);
  } catch (e) {
    return NextResponse.json(
      { ok: false, error: e instanceof Error ? e.message : 'unknown', available: true, authenticated: true },
      { status: 500 },
    );
  }
}
