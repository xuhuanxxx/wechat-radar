/**
 * Client-side API wrapper that routes requests to a user-configured data service.
 *
 * The data service URL is stored in localStorage as `lark-radar-data-url`.
 * When configured, requests go directly to the data service (CORS must be enabled).
 * When not configured, requests fall back to same-origin `/api/*`.
 *
 * For the catch-all proxy route to work, the client also sends the target URL
 * in the `X-Data-Api-Url` header so the server can proxy if needed.
 */

const STORAGE_KEY = 'lark-radar-data-url';

export function getDataApiUrl(): string {
  if (typeof window === 'undefined') return '';
  return localStorage.getItem(STORAGE_KEY) || '';
}

export function setDataApiUrl(url: string): void {
  localStorage.setItem(STORAGE_KEY, url);
}

export function clearDataApiUrl(): void {
  localStorage.removeItem(STORAGE_KEY);
}

/**
 * Build the full API URL for a given path.
 * If a data service URL is configured, prepend it to the path.
 * Otherwise, return the path as-is (same-origin).
 */
export function buildApiUrl(path: string): string {
  const base = getDataApiUrl();
  if (base) {
    const cleanBase = base.replace(/\/$/, '');
    const cleanPath = path.startsWith('/') ? path : `/${path}`;
    return `${cleanBase}${cleanPath}`;
  }
  return path;
}

/**
 * Get default fetch init options.
 * When a data URL is configured, we add the `X-Data-Api-Url` header
 * so the catch-all proxy can forward requests if the direct call fails.
 */
function defaultInit(): RequestInit {
  const base = getDataApiUrl();
  const init: RequestInit = {};
  if (base) {
    init.headers = { 'X-Data-Api-Url': base };
  }
  return init;
}

function mergeInit(base: RequestInit, extra?: RequestInit): RequestInit {
  if (!extra) return base;
  return {
    ...base,
    ...extra,
    headers: {
      ...(base.headers as Record<string, string>),
      ...(extra.headers as Record<string, string>),
    },
  };
}

/**
 * Low-level fetch wrapper that routes to the configured data service.
 */
export async function apiFetch(path: string, init?: RequestInit): Promise<Response> {
  const url = buildApiUrl(path);
  const merged = mergeInit(defaultInit(), init);
  return fetch(url, merged);
}

/**
 * Typed GET helper.
 */
export async function apiGet<T>(path: string): Promise<T> {
  const r = await apiFetch(path, { cache: 'no-store' });
  if (!r.ok) throw new Error(`${path} failed: ${r.status}`);
  return r.json() as Promise<T>;
}

/**
 * Typed POST helper.
 */
export async function apiPost<T>(path: string, body: unknown): Promise<T> {
  const r = await apiFetch(path, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify(body),
  });
  if (!r.ok) throw new Error(`${path} failed: ${r.status}`);
  return r.json() as Promise<T>;
}

/**
 * Typed PUT helper.
 */
export async function apiPut<T>(path: string, body: unknown): Promise<T> {
  const r = await apiFetch(path, {
    method: 'PUT',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify(body),
  });
  if (!r.ok) throw new Error(`${path} failed: ${r.status}`);
  return r.json() as Promise<T>;
}

/**
 * Typed DELETE helper.
 */
export async function apiDelete<T>(path: string): Promise<T> {
  const r = await apiFetch(path, { method: 'DELETE' });
  if (!r.ok) throw new Error(`${path} failed: ${r.status}`);
  return r.json() as Promise<T>;
}
