const appCache = "xuezh-app-v1";
const audioCache = "xuezh-audio-v1";
const shellPaths = new Set([
  "/",
  "/xuezh",
  "/index.html",
  "/manifest.webmanifest",
  "/sw.js",
  "/icon.svg",
]);
const shellFallbacks = ["/xuezh", "/", "/index.html"];

self.addEventListener("install", (event) => {
  event.waitUntil((async () => {
    await refreshAppShellCache().catch(() => cacheFallbackShell());
    self.skipWaiting();
  })());
});

self.addEventListener("activate", (event) => {
  event.waitUntil((async () => {
    for (const name of await caches.keys()) {
      if (name !== appCache && name !== audioCache) await caches.delete(name);
    }
    await self.clients.claim();
  })());
});

self.addEventListener("fetch", (event) => {
  const url = new URL(event.request.url);
  if (event.request.method !== "GET" || url.origin !== location.origin) return;
  if (url.pathname.startsWith("/api/") || url.pathname.startsWith("/offline/")) return;
  if (url.pathname.startsWith("/artifacts/")) {
    event.respondWith(cacheFirst(event.request, audioCache));
    return;
  }
  if (isAppShellRequest(event.request, url)) {
    event.respondWith(appShellFirst(event.request, event));
  }
});

function isAppShellRequest(request, url) {
  return request.mode === "navigate" || url.pathname.startsWith("/assets/") || shellPaths.has(url.pathname);
}

async function refreshAppShellCache() {
  const response = await fetch("/offline/app-shell", { cache: "no-store" });
  if (!response.ok) throw new Error("app shell manifest failed");
  const body = await response.json();
  const cache = await caches.open(appCache);
  await Promise.all((body.assets || ["/xuezh"]).map((asset) => {
    return refreshCacheEntry(cache, asset).catch(() => undefined);
  }));
}

async function cacheFallbackShell() {
  const cache = await caches.open(appCache);
  await Promise.all(shellFallbacks.map((path) => {
    return refreshCacheEntry(cache, path).catch(() => undefined);
  }));
}

async function appShellFirst(request, event) {
  const cache = await caches.open(appCache);
  const cached = await cache.match(request) || (request.mode === "navigate" ? await fallbackShell(cache) : undefined);
  const refresh = refreshCacheEntry(cache, request).catch(() => undefined);
  if (cached) {
    event.waitUntil(refresh);
    return cached;
  }
  await refresh;
  return await cache.match(request) || await fallbackShell(cache) || Response.error();
}

async function refreshCacheEntry(cache, request) {
  const response = await fetch(request, { cache: "no-store" });
  if (!response.ok) throw new Error(`bad response ${response.status}`);
  await cache.put(request, response.clone());
  return response;
}

async function fallbackShell(cache) {
  for (const path of shellFallbacks) {
    const response = await cache.match(path);
    if (response) return response;
  }
  return undefined;
}

async function cacheFirst(request, cacheName) {
  const cache = await caches.open(cacheName);
  const cached = await cache.match(request);
  if (cached) return cached;
  const response = await fetch(request);
  if (response.ok) await cache.put(request, response.clone());
  return response;
}
