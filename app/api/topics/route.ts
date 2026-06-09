import { NextRequest, NextResponse } from 'next/server';
import { listTopics } from '@/lib/topics';
import { todayStr } from '@/lib/range';
import { withCache, CK } from '@/lib/cache';

export const dynamic = 'force-dynamic';

export async function GET(req: NextRequest) {
  const url = new URL(req.url);
  const date = url.searchParams.get('date') ?? todayStr();
  const result = await withCache(CK.topics(date), 60, async () => {
    const topics = listTopics(date);
    return { ok: true, date, topics };
  });
  return NextResponse.json(result);
}
