/**
 * Client-side API wrapper that routes ALL requests through the Next.js
 * catch-all proxy at /api/*.
 *
 * The data service URL is stored in localStorage as `lark-radar-data-url`.
 * Every request goes to the same-origin `/api/*` path with the target
 * data service URL in the `X-Data-Api-Url` header. The Next.js proxy
 * forwards the request to the actual Go data service.
 *
 * This ensures the web frontend never directly accesses data services
 * and works correctly in containerized deployments.
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
 * Build the API URL. Always returns a same-origin `/api/*` path.
 * The proxy route reads X-Data-Api-Url header to forward to Go.
 */
export function buildApiUrl(path: string): string {
  // Always use same-origin path; proxy handles forwarding
  return path.startsWith('/') ? path : `/${path}`;
}

/**
 * Get default fetch init options.
 * Always includes X-Data-Api-Url header so the proxy knows where
 * to forward the request.
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
 * Low-level fetch wrapper. Always requests same-origin /api/*
 * which is proxied to the Go data service.
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
