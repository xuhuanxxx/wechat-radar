import { NextRequest, NextResponse } from 'next/server';

export const dynamic = 'force-dynamic';

const ALLOWED_METHODS = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'OPTIONS'];

/**
 * Catch-all proxy route that forwards unmatched /api/* requests to a
 * user-configured data service. The target URL is read from the
 * `X-Data-Api-Url` request header (set by the client-side api-client).
 *
 * This enables the web frontend to work even when the data service runs
 * on a different origin and direct CORS calls are blocked.
 */
// Default data service URL from environment (set at container build/run time)
const DEFAULT_DATA_API_URL = process.env.DATA_API_URL || '';

async function proxyRequest(req: NextRequest, method: string) {
  // Priority: request header > environment variable
  const dataApiUrl = req.headers.get('x-data-api-url') || DEFAULT_DATA_API_URL;

  if (!dataApiUrl) {
    return NextResponse.json(
      {
        ok: false,
        error:
          'Data service URL not configured. Please set the data service URL in Settings.',
      },
      { status: 503 },
    );
  }

  // Validate the dataApiUrl is a valid HTTP(S) URL
  let targetBase: URL;
  try {
    targetBase = new URL(dataApiUrl);
  } catch {
    return NextResponse.json(
      { ok: false, error: 'Invalid data service URL' },
      { status: 400 },
    );
  }

  if (targetBase.protocol !== 'http:' && targetBase.protocol !== 'https:') {
    return NextResponse.json(
      { ok: false, error: 'Data service URL must use http or https' },
      { status: 400 },
    );
  }

  // Build target URL: preserve path and query string
  const { pathname, search } = new URL(req.url);
  const targetUrl = new URL(`${pathname}${search}`, targetBase);

  // Forward relevant headers (omit host and connection-related ones)
  const forwardHeaders = new Headers();
  req.headers.forEach((value, key) => {
    const lower = key.toLowerCase();
    if (
      lower === 'host' ||
      lower === 'connection' ||
      lower === 'content-length' ||
      lower === 'x-data-api-url'
    ) {
      return;
    }
    forwardHeaders.set(key, value);
  });

  // Read body for non-GET/HEAD requests
  let body: BodyInit | undefined;
  if (method !== 'GET' && method !== 'HEAD') {
    const contentType = req.headers.get('content-type') || '';
    if (contentType.includes('application/json')) {
      try {
        body = JSON.stringify(await req.json());
      } catch {
        body = await req.text();
      }
    } else if (contentType.includes('multipart/form-data') || contentType.includes('application/x-www-form-urlencoded')) {
      body = await req.formData();
    } else {
      body = await req.text();
    }
  }

  try {
    // Use AbortController for long-running requests (sync can take 30s+)
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), 120_000); // 120s timeout

    const upstream = await fetch(targetUrl.toString(), {
      method,
      headers: forwardHeaders,
      body,
      redirect: 'manual',
      signal: controller.signal,
    });

    clearTimeout(timeoutId);

    // Build response, forwarding status and headers
    const responseHeaders = new Headers();
    upstream.headers.forEach((value, key) => {
      // Skip headers that Next.js manages
      if (['content-encoding', 'transfer-encoding'].includes(key.toLowerCase())) return;
      responseHeaders.set(key, value);
    });

    // Add CORS headers so the browser allows the response
    responseHeaders.set('Access-Control-Allow-Origin', '*');
    responseHeaders.set('Access-Control-Allow-Methods', ALLOWED_METHODS.join(', '));
    responseHeaders.set('Access-Control-Allow-Headers', '*');
    
    // Prevent browser from caching error responses
    responseHeaders.set('Cache-Control', 'no-store, no-cache, must-revalidate, proxy-revalidate');
    responseHeaders.set('Pragma', 'no-cache');
    responseHeaders.set('Expires', '0');

    return new NextResponse(upstream.body, {
      status: upstream.status,
      statusText: upstream.statusText,
      headers: responseHeaders,
    });
  } catch (e) {
    const message = e instanceof Error ? e.message : 'Proxy request failed';
    return NextResponse.json(
      { ok: false, error: message },
      { status: 502 },
    );
  }
}

export async function GET(req: NextRequest) {
  return proxyRequest(req, 'GET');
}

export async function POST(req: NextRequest) {
  return proxyRequest(req, 'POST');
}

export async function PUT(req: NextRequest) {
  return proxyRequest(req, 'PUT');
}

export async function DELETE(req: NextRequest) {
  return proxyRequest(req, 'DELETE');
}

export async function PATCH(req: NextRequest) {
  return proxyRequest(req, 'PATCH');
}

export async function OPTIONS() {
  return new NextResponse(null, {
    status: 204,
    headers: {
      'Access-Control-Allow-Origin': '*',
      'Access-Control-Allow-Methods': ALLOWED_METHODS.join(', '),
      'Access-Control-Allow-Headers': '*',
    },
  });
}
