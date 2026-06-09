import { NextRequest, NextResponse } from 'next/server';
import { writeConfig, readConfig, type LarkChatFilter } from '@/lib/config';
import { z } from 'zod';

export const dynamic = 'force-dynamic';

const FilterSchema = z.object({
  mode: z.enum(['all', 'allowlist', 'blocklist']),
  allowlist: z.array(z.string()).default([]),
  blocklist: z.array(z.string()).default([]),
});

export async function GET() {
  const cfg = readConfig();
  return NextResponse.json({ ok: true, filter: cfg.larkChatFilter });
}

export async function POST(req: NextRequest) {
  const body = await req.json().catch(() => null);
  const parsed = FilterSchema.safeParse(body);
  if (!parsed.success) {
    return NextResponse.json({ ok: false, error: parsed.error.message }, { status: 400 });
  }
  const filter: LarkChatFilter = {
    mode: parsed.data.mode,
    allowlist: parsed.data.allowlist,
    blocklist: parsed.data.blocklist,
  };
  writeConfig({ larkChatFilter: filter });
  return NextResponse.json({ ok: true, filter });
}
