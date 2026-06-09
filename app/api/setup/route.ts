import { NextRequest, NextResponse } from 'next/server';
import { z } from 'zod';
import { DATA_DIR, configStatus, writeConfig, type DataSource, type LarkChatFilter } from '@/lib/config';
import { seedDemoData } from '@/lib/demo-data';
import { larkAvailable, larkDoctor } from '@/lib/lark';

export const dynamic = 'force-dynamic';

const SetupSchema = z.object({
  myNicknames: z.array(z.string()).default([]),
  privacyConfirmed: z.boolean(),
  demoMode: z.boolean().default(false),
  defaultSyncDays: z.number().int().min(1).max(365).default(7),
  source: z.enum(['lark', 'demo']).default('lark'),
  larkChatFilter: z
    .object({
      mode: z.enum(['all', 'allowlist', 'blocklist']),
      allowlist: z.array(z.string()).default([]),
      blocklist: z.array(z.string()).default([]),
    })
    .optional(),
  port: z.number().int().min(1024).max(65535).optional(),
  larkCliPath: z.string().optional(),
  openApiKey: z.string().optional(),
  autoSyncInterval: z.number().int().min(0).max(1440).optional(),
});

export async function GET() {
  const [larkInstalled, larkDoc] = await Promise.all([
    larkAvailable(),
    larkDoctor(),
  ]);
  return NextResponse.json({
    ok: true,
    ...configStatus(),
    dataDir: DATA_DIR,
    checks: {
      larkInstalled,
      larkAuthenticated: larkDoc.authenticated,
      larkError: larkDoc.error ?? null,
    },
  });
}

export async function POST(req: NextRequest) {
  const body = await req.json().catch(() => null);
  const parsed = SetupSchema.safeParse(body);
  if (!parsed.success) {
    return NextResponse.json({ ok: false, error: parsed.error.message }, { status: 400 });
  }
  const names = parsed.data.myNicknames.map((name) => name.trim()).filter(Boolean);
  const effectiveSource: DataSource = parsed.data.demoMode ? 'demo' : 'lark';

  if (effectiveSource !== 'demo' && names.length === 0) {
    return NextResponse.json(
      { ok: false, error: '请至少填写一个自己的昵称（用于 @ 检测）' },
      { status: 400 },
    );
  }

  const patch: Parameters<typeof writeConfig>[0] = {
    myNicknames: names,
    privacyConfirmed: parsed.data.privacyConfirmed,
    demoMode: parsed.data.demoMode,
    defaultSyncDays: parsed.data.defaultSyncDays,
    source: effectiveSource,
    setupCompleted: true,
  };
  if (parsed.data.larkChatFilter) {
    patch.larkChatFilter = parsed.data.larkChatFilter as LarkChatFilter;
  }
  if (parsed.data.port !== undefined) patch.port = parsed.data.port;
  if (parsed.data.larkCliPath !== undefined) patch.larkCliPath = parsed.data.larkCliPath;
  if (parsed.data.openApiKey !== undefined) patch.openApiKey = parsed.data.openApiKey;
  if (parsed.data.autoSyncInterval !== undefined) patch.autoSyncInterval = parsed.data.autoSyncInterval;
  const config = writeConfig(patch);
  const demo = parsed.data.demoMode ? seedDemoData() : null;
  return NextResponse.json({ ok: true, configured: true, config, demo });
}
