import { NextRequest } from 'next/server';
import { syncLarkMessages } from '@/lib/lark-sync';

export const dynamic = 'force-dynamic';
export const maxDuration = 300;

export async function POST(req: NextRequest) {
  const body = (await req.json().catch(() => ({}))) as {
    chat_id?: string;
    days_back?: number;
    stream?: boolean;
  };

  // Non-streaming mode for simple requests
  if (!body.stream) {
    try {
      const result = await syncLarkMessages({
        chatId: body.chat_id,
        daysBack: body.days_back,
      });
      if (!result.ok) {
        return new Response(
          JSON.stringify({ ok: false, error: result.error, synced: result.synced }),
          { status: 400, headers: { 'content-type': 'application/json' } },
        );
      }
      return new Response(JSON.stringify({ ok: true, synced: result.synced }), {
        headers: { 'content-type': 'application/json' },
      });
    } catch (e) {
      return new Response(
        JSON.stringify({ ok: false, error: e instanceof Error ? e.message : 'unknown' }),
        { status: 500, headers: { 'content-type': 'application/json' } },
      );
    }
  }

  // Streaming mode with SSE
  const stream = new ReadableStream({
    async start(controller) {
      const enc = new TextEncoder();
      const send = (obj: unknown) =>
        controller.enqueue(enc.encode(`data: ${JSON.stringify(obj)}\n\n`));

      send({ type: 'start' });

      try {
        const result = await syncLarkMessages({
          chatId: body.chat_id,
          daysBack: body.days_back,
          onProgress: (chatId, info) => {
            send({ type: 'progress', chatId, ...info });
          },
        });

        send({ type: 'finished', ok: result.ok, synced: result.synced });
      } catch (e) {
        send({
          type: 'error',
          error: e instanceof Error ? e.message : 'unknown',
        });
      } finally {
        controller.close();
      }
    },
  });

  return new Response(stream, {
    headers: {
      'Content-Type': 'text/event-stream',
      'Cache-Control': 'no-cache, no-transform',
      Connection: 'keep-alive',
    },
  });
}
