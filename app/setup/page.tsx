'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  CheckCircle2,
  Database,
  ShieldCheck,
  UserRound,
  Wrench,
  MessageCircle,
  Filter,
  Search,
} from 'lucide-react';

type SourceType = 'lark' | 'demo';

type LarkChatFilter = {
  mode: 'all' | 'allowlist' | 'blocklist';
  allowlist: string[];
  blocklist: string[];
};

type SetupStatus = {
  ok: boolean;
  dataDir: string;
  configured: boolean;
  config: {
    myNicknames: string[];
    demoMode: boolean;
    privacyConfirmed: boolean;
    defaultSyncDays: number;
    source: SourceType;
    larkChatFilter: LarkChatFilter;
    port: number;
    larkCliPath: string;
    openApiKey: string;
    autoSyncInterval: number;
  };
  checks: {
    larkInstalled: boolean;
    larkAuthenticated: boolean;
    larkError: string | null;
  };
};

type LarkChatItem = {
  id: string;
  name: string;
  member_count: number;
  filtered: boolean;
};

export default function SetupPage() {
  const [status, setStatus] = useState<SetupStatus | null>(null);
  const [names, setNames] = useState('');
  const [demoMode, setDemoMode] = useState(false);
  const [privacyConfirmed, setPrivacyConfirmed] = useState(false);
  const [defaultSyncDays, setDefaultSyncDays] = useState(7);
  const [source, setSource] = useState<SourceType>('lark');
  const [filter, setFilter] = useState<LarkChatFilter>({
    mode: 'all',
    allowlist: [],
    blocklist: [],
  });
  const [port, setPort] = useState(3456);
  const [larkCliPath, setLarkCliPath] = useState('');
  const [openApiKey, setOpenApiKey] = useState('');
  const [autoSyncInterval, setAutoSyncInterval] = useState(0);
  const [larkChats, setLarkChats] = useState<LarkChatItem[] | null>(null);
  const [larkChatsLoading, setLarkChatsLoading] = useState(false);
  const [larkChatsError, setLarkChatsError] = useState<string | null>(null);
  const [chatSearch, setChatSearch] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      const res = await fetch('/api/setup', { cache: 'no-store' });
      const json = (await res.json()) as SetupStatus;
      setStatus(json);
      setNames(json.config.myNicknames.join(', '));
      setDemoMode(json.config.demoMode);
      setPrivacyConfirmed(json.config.privacyConfirmed);
      setDefaultSyncDays(json.config.defaultSyncDays ?? 7);
      setSource(json.config.source ?? (json.config.demoMode ? 'demo' : 'lark'));
      setPort(json.config.port ?? 3456);
      setLarkCliPath(json.config.larkCliPath ?? '');
      setOpenApiKey(json.config.openApiKey ?? '');
      setAutoSyncInterval(json.config.autoSyncInterval ?? 0);
      setFilter(
        json.config.larkChatFilter ?? {
          mode: 'all',
          allowlist: [],
          blocklist: [],
        },
      );
    })();
  }, []);

  const loadLarkChats = useCallback(async () => {
    setLarkChatsLoading(true);
    setLarkChatsError(null);
    try {
      const res = await fetch('/api/lark/chats', { cache: 'no-store' });
      const json = await res.json();
      if (!res.ok || !json.ok) throw new Error(json.error || '获取群列表失败');
      setLarkChats(json.chats);
      if (json.filter) setFilter(json.filter);
    } catch (e) {
      setLarkChatsError(e instanceof Error ? e.message : 'unknown');
    } finally {
      setLarkChatsLoading(false);
    }
  }, []);

  const filteredChats = useMemo(() => {
    if (!larkChats) return [];
    if (!chatSearch.trim()) return larkChats;
    const q = chatSearch.trim().toLowerCase();
    return larkChats.filter((c) => c.name.toLowerCase().includes(q));
  }, [larkChats, chatSearch]);

  function toggleChat(chatId: string, checked: boolean) {
    setFilter((prev) => {
      if (prev.mode === 'allowlist') {
        const next = checked
          ? Array.from(new Set([...prev.allowlist, chatId]))
          : prev.allowlist.filter((id) => id !== chatId);
        return { ...prev, allowlist: next };
      }
      if (prev.mode === 'blocklist') {
        const next = checked
          ? Array.from(new Set([...prev.blocklist, chatId]))
          : prev.blocklist.filter((id) => id !== chatId);
        return { ...prev, blocklist: next };
      }
      return prev;
    });
  }

  async function submit() {
    setBusy(true);
    setError(null);
    try {
      const res = await fetch('/api/setup', {
        method: 'POST',
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
          myNicknames: names.split(',').map((name) => name.trim()).filter(Boolean),
          demoMode,
          privacyConfirmed,
          defaultSyncDays,
          source: demoMode ? 'demo' : 'lark',
          larkChatFilter: filter,
          port,
          larkCliPath,
          openApiKey,
          autoSyncInterval,
        }),
      });
      const json = await res.json();
      if (!res.ok || !json.ok) throw new Error(json.error ?? '保存失败');
      window.location.href = '/';
    } catch (e) {
      setError(e instanceof Error ? e.message : '保存失败');
    } finally {
      setBusy(false);
    }
  }

  const effectiveSource: SourceType = demoMode ? 'demo' : source;

  return (
    <main className="min-h-screen bg-[var(--bg)] px-6 py-8 text-[var(--text)]">
      <div className="mx-auto max-w-4xl">
        <div className="report-kicker">Lark Radar Setup</div>
        <h1 className="mt-2 text-[28px] font-semibold">配置雷达</h1>
        <p className="mt-2 text-[13px] leading-relaxed text-[var(--text-2)]">
          首次运行需要确认本地环境、填写你的昵称。所有数据默认保存在本机。
        </p>

        <div className="mt-6 grid grid-cols-1 gap-4 lg:grid-cols-2">
          <section className="card p-5 lg:col-span-2">
            <SectionTitle icon={<Database size={15} />} title="数据源" />
            <div className="mt-4 flex flex-wrap gap-3">
              <SourceButton
                active={effectiveSource === 'lark'}
                onClick={() => {
                  setDemoMode(false);
                  setSource('lark');
                  if (larkChats === null) loadLarkChats();
                }}
                label="飞书"
                sub="lark-cli + 开放平台"
              />
              <SourceButton
                active={effectiveSource === 'demo'}
                onClick={() => setDemoMode(true)}
                label="示例数据"
                sub="先体验功能"
              />
            </div>
          </section>

          <section className="card p-5">
            <SectionTitle icon={<Wrench size={15} />} title="环境检查" />
            {effectiveSource === 'lark' && (
              <>
                <CheckRow
                  label="lark-cli"
                  ok={status?.checks.larkInstalled ?? false}
                  detail={status?.checks.larkInstalled ? '已安装' : '未检测到 lark-cli 命令'}
                />
                <CheckRow
                  label="lark 登录"
                  ok={status?.checks.larkAuthenticated ?? false}
                  detail={
                    status?.checks.larkAuthenticated
                      ? '已登录'
                      : status?.checks.larkError || '未登录，请运行 lark-cli auth login --as user'
                  }
                />
              </>
            )}
            {effectiveSource === 'demo' && (
              <CheckRow label="示例数据" ok detail="无需本地环境即可体验" />
            )}
            <CheckRow label="数据目录" ok detail={status?.dataDir ?? '加载中'} />
          </section>

          <section className="card p-5">
            <SectionTitle icon={<UserRound size={15} />} title="你的昵称" />
            <label className="mt-3 block text-[12px] text-[var(--text-3)]">多个名称用英文逗号分隔</label>
            <input
              value={names}
              onChange={(e) => setNames(e.target.value)}
              placeholder="张三, San Zhang, zhangsan"
              className="control-surface mt-2 w-full rounded-md px-3 py-2 text-[13px] outline-none"
            />
            <p className="mt-2 text-[11px] text-[var(--text-3)]">用于识别 @我的、自己相关讨论和提醒。</p>
          </section>

          <section className="card p-5">
            <SectionTitle icon={<MessageCircle size={15} />} title="同步设置" />
            <label className="mt-4 block text-[12px] text-[var(--text-3)]">首次同步天数</label>
            <select
              value={defaultSyncDays}
              onChange={(e) => setDefaultSyncDays(Number(e.target.value))}
              className="control-surface mt-2 rounded-md px-3 py-2 text-[13px] outline-none"
            >
              <option value={1}>最近 1 天</option>
              <option value={7}>最近 7 天</option>
              <option value={30}>最近 30 天</option>
              <option value={365}>最近 365 天</option>
            </select>
            <label className="mt-4 block text-[12px] text-[var(--text-3)]">自动同步间隔</label>
            <select
              value={autoSyncInterval}
              onChange={(e) => setAutoSyncInterval(Number(e.target.value))}
              className="control-surface mt-2 rounded-md px-3 py-2 text-[13px] outline-none"
            >
              <option value={0}>手动同步</option>
              <option value={5}>每 5 分钟</option>
              <option value={15}>每 15 分钟</option>
              <option value={30}>每 30 分钟</option>
              <option value={60}>每 1 小时</option>
            </select>
          </section>

          <section className="card p-5">
            <SectionTitle icon={<Wrench size={15} />} title="高级设置" />
            <label className="mt-3 block text-[12px] text-[var(--text-3)]">服务端口</label>
            <input
              type="number"
              min={1024}
              max={65535}
              value={port}
              onChange={(e) => setPort(Number(e.target.value))}
              className="control-surface mt-2 w-full rounded-md px-3 py-2 text-[13px] outline-none"
            />
            <p className="mt-1 text-[11px] text-[var(--text-3)]">修改后需重启应用生效</p>

            <label className="mt-4 block text-[12px] text-[var(--text-3)]">lark-cli 路径（可选）</label>
            <input
              value={larkCliPath}
              onChange={(e) => setLarkCliPath(e.target.value)}
              placeholder="/usr/local/bin/lark-cli"
              className="control-surface mt-2 w-full rounded-md px-3 py-2 text-[13px] outline-none"
            />
            <p className="mt-1 text-[11px] text-[var(--text-3)]">留空使用 PATH 中的 lark-cli</p>

            <label className="mt-4 block text-[12px] text-[var(--text-3)]">OpenAPI Key（可选）</label>
            <input
              type="password"
              value={openApiKey}
              onChange={(e) => setOpenApiKey(e.target.value)}
              placeholder="sk-..."
              className="control-surface mt-2 w-full rounded-md px-3 py-2 text-[13px] outline-none"
            />
            <p className="mt-1 text-[11px] text-[var(--text-3)]">用于话题聚合和链接摘要（本地存储）</p>
          </section>

          <section className="card p-5">
            <SectionTitle icon={<ShieldCheck size={15} />} title="隐私确认" />
            <label className="mt-4 flex items-start gap-2 text-[13px] leading-relaxed">
              <input
                type="checkbox"
                checked={privacyConfirmed}
                onChange={(e) => setPrivacyConfirmed(e.target.checked)}
                className="mt-0.5"
              />
              <span>
                我确认本工具仅用于个人数据分析，所有消息内容仅保存在本地数据库，不会上传到任何第三方服务器。
              </span>
            </label>
          </section>

          {effectiveSource === 'lark' && (
            <section className="card p-5 lg:col-span-2">
              <SectionTitle icon={<Filter size={15} />} title="群聊过滤" />
              <div className="mt-3 flex flex-wrap gap-2">
                <FilterModeButton
                  active={filter.mode === 'all'}
                  onClick={() => setFilter((p) => ({ ...p, mode: 'all' }))}
                  label="全部群聊"
                />
                <FilterModeButton
                  active={filter.mode === 'allowlist'}
                  onClick={() => setFilter((p) => ({ ...p, mode: 'allowlist' }))}
                  label="仅同步选中"
                />
                <FilterModeButton
                  active={filter.mode === 'blocklist'}
                  onClick={() => setFilter((p) => ({ ...p, mode: 'blocklist' }))}
                  label="排除选中"
                />
              </div>

              <div className="mt-3">
                <div className="flex items-center gap-2">
                  <Search size={14} className="text-[var(--text-3)]" />
                  <input
                    value={chatSearch}
                    onChange={(e) => setChatSearch(e.target.value)}
                    placeholder="搜索群聊..."
                    className="control-surface flex-1 rounded-md px-3 py-1.5 text-[13px] outline-none"
                  />
                  <button
                    className="btn text-[12px]"
                    onClick={loadLarkChats}
                    disabled={larkChatsLoading}
                  >
                    {larkChatsLoading ? '加载中…' : '加载群列表'}
                  </button>
                  {(filter.allowlist.length > 0 || filter.blocklist.length > 0) && (
                    <button
                      className="btn text-[12px]"
                      onClick={() => setFilter({ mode: filter.mode, allowlist: [], blocklist: [] })}
                    >
                      清空
                    </button>
                  )}
                </div>

                {larkChatsLoading && (
                  <div className="mt-3 text-[12px] text-[var(--text-3)]">加载群列表…</div>
                )}
                {larkChatsError && (
                  <div className="mt-3 text-[12px] text-[var(--danger)]">{larkChatsError}</div>
                )}

                {!larkChatsLoading && !larkChatsError && larkChats !== null && filteredChats.length === 0 && (
                  <div className="mt-3 text-[12px] text-[var(--text-3)]">未找到群聊</div>
                )}

                {filteredChats.length > 0 && (
                  <div className="mt-3 max-h-64 overflow-auto rounded-md border border-[var(--border)]">
                    {filteredChats.map((chat) => {
                      const selected =
                        filter.mode === 'allowlist'
                          ? filter.allowlist.includes(chat.id)
                          : filter.blocklist.includes(chat.id);
                      return (
                        <label
                          key={chat.id}
                          className="flex cursor-pointer items-center justify-between gap-3 border-b border-[var(--border)] px-3 py-2 text-[13px] last:border-b-0 hover:bg-[var(--surface-hover)]"
                        >
                          <span className="flex min-w-0 flex-col">
                            <span className="truncate">{chat.name}</span>
                            <span className="text-[11px] text-[var(--text-3)]">
                              {chat.member_count} 人 · {chat.id}
                            </span>
                          </span>
                          <input
                            type="checkbox"
                            checked={selected}
                            onChange={(e) => toggleChat(chat.id, e.target.checked)}
                          />
                        </label>
                      );
                    })}
                  </div>
                )}

                <p className="mt-2 text-[11px] text-[var(--text-3)]">
                  {filter.mode === 'allowlist'
                    ? '只同步选中的群。'
                    : '同步所有群，但排除选中的群。'}
                </p>
              </div>
            </section>
          )}
        </div>

        {error && <div className="mt-4 text-[13px] text-[var(--danger)]">{error}</div>}

        <div className="mt-6 flex justify-end gap-2">
          <button className="btn" onClick={() => (window.location.href = '/')}>稍后再说</button>
          <button className="btn btn-primary" disabled={busy || !privacyConfirmed} onClick={submit}>
            {busy ? '保存中…' : '完成配置'}
          </button>
        </div>
      </div>
    </main>
  );
}

function SectionTitle({ icon, title }: { icon: React.ReactNode; title: string }) {
  return (
    <div className="flex items-center gap-1.5 text-[14px] font-semibold text-[var(--text)]">
      {icon}
      {title}
    </div>
  );
}

function CheckRow({ label, ok, detail }: { label: string; ok: boolean; detail: string }) {
  return (
    <div className="mt-3 flex items-center justify-between gap-3 text-[13px]">
      <span className="text-[var(--text-2)]">{label}</span>
      <span className="flex min-w-0 items-center gap-1.5 text-right text-[12px] text-[var(--text-3)]">
        <CheckCircle2 size={13} className={ok ? 'text-[var(--accent)]' : 'text-[var(--text-3)]'} />
        <span className="truncate">{detail}</span>
      </span>
    </div>
  );
}

function SourceButton({
  active,
  onClick,
  label,
  sub,
}: {
  active: boolean;
  onClick: () => void;
  label: string;
  sub: string;
}) {
  return (
    <button
      onClick={onClick}
      className={`flex flex-col items-start rounded-md border px-4 py-3 text-left transition ${
        active
          ? 'border-[var(--accent)] bg-[var(--accent)]/10'
          : 'border-[var(--border)] bg-[var(--surface)] hover:bg-[var(--surface-hover)]'
      }`}
    >
      <span className="text-[13px] font-medium">{label}</span>
      <span className="text-[11px] text-[var(--text-3)]">{sub}</span>
    </button>
  );
}

function FilterModeButton({
  active,
  onClick,
  label,
}: {
  active: boolean;
  onClick: () => void;
  label: string;
}) {
  return (
    <button
      onClick={onClick}
      className={`rounded-md border px-3 py-1.5 text-[12px] transition ${
        active
          ? 'border-[var(--accent)] bg-[var(--accent)]/10 text-[var(--accent)]'
          : 'border-[var(--border)] bg-[var(--surface)] hover:bg-[var(--surface-hover)]'
      }`}
    >
      {label}
    </button>
  );
}
