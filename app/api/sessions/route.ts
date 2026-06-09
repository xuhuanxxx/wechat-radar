import { NextResponse } from 'next/server';
import { loadSessionsSafe } from '@/lib/sessions';
import { listGroups, listAllTags, listFavorites } from '@/lib/groups';
import { effectiveGroupIds } from '@/lib/group-classifier';
import { withCache, CK } from '@/lib/cache';

export const dynamic = 'force-dynamic';

export async function GET() {
  try {
    const result = await withCache(CK.sessions(), 30, async () => {
      const sessions = await loadSessionsSafe(500);

      const groups = listGroups();
      const tags = listAllTags();
      const favorites = new Set(listFavorites());

      const tagsByChatroom = new Map<string, number[]>();
      for (const t of tags) {
        const arr = tagsByChatroom.get(t.chatroom_id) ?? [];
        arr.push(t.group_id);
        tagsByChatroom.set(t.chatroom_id, arr);
      }

      const groupsList = sessions.filter((s) => s.is_group);

      const enriched = groupsList.map((s) => {
        const groupIds = effectiveGroupIds(
          s.chat,
          s.summary,
          tagsByChatroom.get(s.username) ?? [],
          groups,
        );
        return {
          chatroom_id: s.username,
          name: s.chat,
          last_msg_type: s.last_msg_type,
          last_sender: s.last_sender,
          summary: s.summary,
          time: s.time,
          timestamp: s.timestamp,
          unread: s.unread,
          is_favorite: favorites.has(s.username),
          group_ids: groupIds,
        };
      });

      const memberCounts = new Map<number, number>();
      for (const g of enriched) {
        for (const groupId of g.group_ids) {
          memberCounts.set(groupId, (memberCounts.get(groupId) ?? 0) + 1);
        }
      }
      const categories = groups.map((g) => ({
        ...g,
        member_count: memberCounts.get(g.id) ?? 0,
      }));

      return {
        ok: true,
        total: groupsList.length,
        groups: enriched,
        categories,
      };
    });

    return NextResponse.json(result);
  } catch (e) {
    const message = e instanceof Error ? e.message : 'unknown error';
    return NextResponse.json({ ok: false, error: message }, { status: 500 });
  }
}


