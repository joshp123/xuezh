const appCache = "xuezh-app-v1";
const audioCache = "xuezh-audio-v1";

self.addEventListener("install", (event) => {
  event.waitUntil((async () => {
    const cache = await caches.open(appCache);
    try {
      const response = await fetch("/offline/app-shell", { cache: "no-store" });
      const body = await response.json();
      await cache.addAll(body.assets || ["/"]);
    } catch {
      await cache.add("/");
    }
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
  if (url.pathname.startsWith("/api/")) return;
  if (url.pathname.startsWith("/artifacts/")) {
    event.respondWith(cacheFirst(event.request, audioCache));
    return;
  }
  event.respondWith(appShellFirst(event.request));
});

async function appShellFirst(request) {
  const cache = await caches.open(appCache);
  try {
    const response = await fetch(request);
    if (response.ok) await cache.put(request, response.clone());
    return response;
  } catch {
    return await cache.match(request) || await cache.match("/") || Response.error();
  }
}

async function cacheFirst(request, cacheName) {
  const cache = await caches.open(cacheName);
  const cached = await cache.match(request);
  if (cached) return cached;
  const response = await fetch(request);
  if (response.ok) await cache.put(request, response.clone());
  return response;
}
