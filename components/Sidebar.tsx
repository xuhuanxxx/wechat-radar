'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import {
  Star,
  Folder,
  Inbox,
  LayoutDashboard,
  Sparkles,
  Link2,
} from 'lucide-react';
import ThemeToggle from './ThemeToggle';

type Category = {
  id: number;
  name: string;
  color: string;
  emoji: string | null;
  member_count?: number;
};

type SidebarData = {
  ok: boolean;
  total: number;
  categories: Category[];
};

export default function Sidebar() {
  const pathname = usePathname();
  const [data, setData] = useState<SidebarData | null>(null);
  const [unsorted, setUnsorted] = useState(0);
  const [favorites, setFavorites] = useState(0);

  useEffect(() => {
    const load = async () => {
      try {
        const r = await fetch('/api/sessions');
        const j = await r.json();
        setData(j);
      } catch {}
      try {
        const r = await fetch('/api/stats?range=week');
        const j = await r.json();
        if (j.ok && j.sidebar_counts) {
          setUnsorted(j.sidebar_counts.unsorted);
          setFavorites(j.sidebar_counts.favorites);
        }
      } catch {}
    };
    load();
    const id = setInterval(load, 30_000);
    return () => clearInterval(id);
  }, []);

  return (
    <aside className="sidebar flex h-full w-56 flex-col border-r border-[var(--border-soft)] bg-[var(--bg-1)]">
      <div className="flex items-center gap-2 px-4 py-3">
        <div className="flex size-8 items-center justify-center rounded-lg bg-[var(--accent)] text-white text-[14px] font-bold">
          📡
        </div>
        <div>
          <div className="text-[13px] font-semibold leading-tight">Lark Radar</div>
          <div className="text-[10px] text-[var(--text-3)] leading-tight">飞书群聊情报站</div>
        </div>
      </div>

      <nav className="flex flex-col gap-0.5 px-2">
        <NavItem href="/" icon={<LayoutDashboard size={15} />} label="Dashboard" active={pathname === '/'} />
        <NavItem href="/groups" icon={<Folder size={15} />} label="Groups" active={pathname === '/groups'} badge={unsorted > 0 ? unsorted : undefined} />
        <NavItem href="/topics" icon={<Sparkles size={15} />} label="Topics" active={pathname === '/topics'} />
        <NavItem href="/links" icon={<Link2 size={15} />} label="Links" active={pathname === '/links'} />
        <NavItem href="/mentions" icon={<Inbox size={15} />} label="Mentions" active={pathname === '/mentions'} />
        <NavItem href="/signals" icon={<Star size={15} />} label="Signals" active={pathname === '/signals'} badge={favorites > 0 ? favorites : undefined} />
      </nav>

      <div className="mt-auto px-2 pb-2">
        <NavItem href="/setup" icon={<span className="text-[12px]">⚙️</span>} label="Setup" active={pathname === '/setup'} />
        <div className="mt-2 px-3">
          <ThemeToggle />
        </div>
      </div>

      <div className="flex-1 overflow-y-auto px-2 pb-2">
        {(data?.categories ?? []).map((c) => (
          <CategoryItem key={c.id} category={c} />
        ))}
      </div>

      <div className="border-t border-[var(--border-soft)] px-4 py-2 text-[11px] text-[var(--text-3)]">
        <span className="inline-block size-2 rounded-full bg-[var(--accent)] mr-1.5 align-middle" />
        Lark Radar
      </div>
    </aside>
  );
}

function NavItem({
  href,
  icon,
  label,
  active,
  badge,
}: {
  href: string;
  icon: React.ReactNode;
  label: string;
  active?: boolean;
  badge?: number;
}) {
  return (
    <Link
      href={href}
      className={`flex items-center gap-2 rounded-md px-3 py-1.5 text-[13px] transition-colors ${
        active
          ? 'bg-[var(--accent-soft)] text-[var(--accent)] font-medium'
          : 'text-[var(--text-2)] hover:bg-[var(--bg-2)]'
      }`}
    >
      {icon}
      <span className="flex-1">{label}</span>
      {badge !== undefined && badge > 0 && (
        <span className="ml-auto rounded-full bg-[var(--accent)] px-1.5 py-0.5 text-[10px] text-white font-medium">
          {badge}
        </span>
      )}
    </Link>
  );
}

function CategoryItem({ category }: { category: Category }) {
  return (
    <Link
      href={`/groups/${category.id}`}
      className="flex items-center gap-2 rounded-md px-3 py-1 text-[12px] text-[var(--text-2)] hover:bg-[var(--bg-2)]"
    >
      <span
        className="inline-block size-2.5 rounded-full"
        style={{ backgroundColor: category.color }}
      />
      <span className="truncate">{category.emoji ? `${category.emoji} ` : ''}{category.name}</span>
    </Link>
  );
}
