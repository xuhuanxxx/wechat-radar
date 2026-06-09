import { NextResponse } from 'next/server';
import { db } from '@/lib/db';
import { withCacheSync, CK } from '@/lib/cache';

export const dynamic = 'force-dynamic';

export async function GET() {
  const result = withCacheSync(CK.stats('dates'), 60, () => {
    const rows = db()
      .prepare(
        `SELECT date, COUNT(*) AS count
         FROM messages
         GROUP BY date
         ORDER BY date DESC
         LIMIT 90`,
      )
      .all() as Array<{ date: string; count: number }>;
    return { ok: true, dates: rows };
  });
  return NextResponse.json(result);
}
