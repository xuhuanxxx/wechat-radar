import { NextRequest, NextResponse } from 'next/server';
import { countMentions, listMentions, markMentionsSeen } from '@/lib/mentions';
import { loadSessionsSafe } from '@/lib/sessions';
import { withCache, CK } from '@/lib/cache';

export const dynamic = 'force-dynamic';

export async function GET(req: NextRequest) {
  const url = new URL(req.url);
  const limit = Math.min(Math.max(Number(url.searchParams.get('limit') ?? 1000), 1), 5000);

  const result = await withCache(`${CK.mentions()}:${limit}`, 30, async () => {
    const sessions = await loadSessionsSafe(500);
    const nameByChatroom = new Map<string, string>();
    for (const s of sessions) nameByChatroom.set(s.username, s.chat);

    const items = listMentions(limit).map((m) => ({
      ...m,
      chat_name: nameByChatroom.get(m.chatroom_id) ?? m.chatroom_id,
    }));

    return { ok: true, total: countMentions(), items };
  });

  return NextResponse.json(result);
}

export async function POST(req: NextRequest) {
  const body = (await req.json().catch(() => ({}))) as { chatroom_id?: string };
  markMentionsSeen(body.chatroom_id);
  return NextResponse.json({ ok: true });
}
