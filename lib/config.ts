import { existsSync, readFileSync, writeFileSync, mkdirSync } from 'node:fs';
import { homedir } from 'node:os';
import { dirname, join } from 'node:path';

export const DATA_DIR =
  process.env.WECHAT_RADAR_DATA_DIR ||
  join(homedir(), '.wechat-radar');

const CONFIG_PATH = join(DATA_DIR, 'config.json');

export type DataSource = 'wechat' | 'lark' | 'demo';

export interface LarkChatFilter {
  mode: 'all' | 'allowlist' | 'blocklist';
  allowlist: string[];
  blocklist: string[];
}

export interface Config {
  myNicknames: string[];
  defaultRange: 'day' | 'week' | 'month' | 'quarter' | 'year';
  rescanConcurrency: number;
  privacyConfirmed: boolean;
  setupCompleted: boolean;
  demoMode: boolean;
  defaultSyncDays: number;
  source: DataSource;
  larkChatFilter: LarkChatFilter;
  port: number;
  larkCliPath: string;
  openApiKey: string;
  autoSyncInterval: number; // minutes, 0 = manual only
}

function envNames(): string[] {
  return (process.env.WECHAT_RADAR_MY_NAMES || '')
    .split(',')
    .map((name) => name.trim())
    .filter(Boolean);
}

const DEFAULTS: Config = {
  myNicknames: envNames(),
  defaultRange: 'week',
  rescanConcurrency: 5,
  privacyConfirmed: false,
  setupCompleted: false,
  demoMode: process.env.WECHAT_RADAR_DEMO === '1',
  defaultSyncDays: 7,
  source: process.env.WECHAT_RADAR_DEMO === '1' ? 'demo' : 'wechat',
  larkChatFilter: { mode: 'all', allowlist: [], blocklist: [] },
  port: 3456,
  larkCliPath: '',
  openApiKey: '',
  autoSyncInterval: 0,
};

export function readConfig(): Config {
  if (!existsSync(CONFIG_PATH)) {
    mkdirSync(dirname(CONFIG_PATH), { recursive: true });
    writeFileSync(CONFIG_PATH, JSON.stringify(DEFAULTS, null, 2), 'utf-8');
    return DEFAULTS;
  }
  try {
    const raw = readFileSync(CONFIG_PATH, 'utf-8');
    const parsed = JSON.parse(raw) as Partial<Config>;
    const merged = { ...DEFAULTS, ...parsed };
    if (envNames().length > 0) merged.myNicknames = envNames();
    if (process.env.WECHAT_RADAR_DEMO === '1') merged.demoMode = true;
    if (!merged.larkChatFilter) merged.larkChatFilter = DEFAULTS.larkChatFilter;
    if (!merged.port || merged.port < 1024 || merged.port > 65535) merged.port = DEFAULTS.port;
    return merged;
  } catch {
    return DEFAULTS;
  }
}

export function writeConfig(patch: Partial<Config>): Config {
  const cur = readConfig();
  const merged = { ...cur, ...patch };
  mkdirSync(dirname(CONFIG_PATH), { recursive: true });
  writeFileSync(CONFIG_PATH, JSON.stringify(merged, null, 2), 'utf-8');
  return merged;
}

export function configStatus() {
  const cfg = readConfig();
  return {
    dataDir: DATA_DIR,
    configPath: CONFIG_PATH,
    configured: cfg.setupCompleted && cfg.privacyConfirmed && (cfg.demoMode || cfg.myNicknames.length > 0),
    config: cfg,
  };
}
