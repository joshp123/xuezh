import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import vm from "node:vm";

const origin = "https://josh-mbp.tailb7ad2a.ts.net";

class MemoryCache {
  constructor(fetcher) {
    this.fetcher = fetcher;
    this.entries = new Map();
  }

  key(input) {
    if (typeof input === "string") return new URL(input, origin).href;
    return input.url;
  }

  async match(input) {
    return this.entries.get(this.key(input));
  }

  async put(input, response) {
    this.entries.set(this.key(input), response.clone());
  }

  async add(input) {
    const response = await this.fetcher(input);
    if (!response.ok) throw new Error(`bad response ${response.status}`);
    await this.put(input, response);
  }
}

function makeHarness(fetcher) {
  const handlers = {};
  const stores = new Map();
  const caches = {
    async open(name) {
      if (!stores.has(name)) stores.set(name, new MemoryCache(fetcher));
      return stores.get(name);
    },
    async keys() {
      return [...stores.keys()];
    },
    async delete(name) {
      return stores.delete(name);
    },
  };
  const self = {
    clients: { claim: async () => undefined },
    skipWaiting: () => undefined,
    addEventListener(type, handler) {
      handlers[type] = handler;
    },
  };
  return { caches, handlers, self, stores };
}

async function loadServiceWorker(fetcher) {
  const harness = makeHarness(fetcher);
  const source = await readFile(new URL("../public/sw.js", import.meta.url), "utf8");
  vm.runInNewContext(source, {
    caches: harness.caches,
    console,
    Error,
    fetch: fetcher,
    location: { origin },
    Response,
    self: harness.self,
    URL,
  });
  return harness;
}

async function runLifecycle(handler) {
  const waits = [];
  handler({
    waitUntil(promise) {
      waits.push(Promise.resolve(promise));
    },
  });
  await Promise.all(waits);
}

function request(path, mode = "same-origin") {
  return { method: "GET", mode, url: new URL(path, origin).href };
}

function dispatch(fetchHandler, req) {
  let responsePromise;
  const waits = [];
  fetchHandler({
    request: req,
    respondWith(promise) {
      responsePromise = Promise.resolve(promise);
    },
    waitUntil(promise) {
      waits.push(Promise.resolve(promise).catch(() => undefined));
    },
  });
  return { responsePromise, waits };
}

function textResponse(body, init) {
  return new Response(body, init);
}

async function cacheShell(harness, path, body) {
  const cache = await harness.caches.open("xuezh-app-v1");
  await cache.put(path, textResponse(body));
}

test("install falls back to the cached shell when the app-shell manifest fails", async () => {
  const harness = await loadServiceWorker((input) => {
    const path = new URL(typeof input === "string" ? input : input.url, origin).pathname;
    if (path === "/offline/app-shell") throw new Error("manifest down");
    if (path === "/xuezh") return textResponse("fallback-shell");
    return textResponse("missing", { status: 404 });
  });
  await runLifecycle(harness.handlers.install);
  const cache = await harness.caches.open("xuezh-app-v1");
  assert.equal(await (await cache.match("/xuezh")).text(), "fallback-shell");
});

test("cached navigation shell wins when the network hangs", async () => {
  const harness = await loadServiceWorker(() => new Promise(() => undefined));
  await cacheShell(harness, "/xuezh", "cached-shell");
  const { responsePromise } = dispatch(harness.handlers.fetch, request("/xuezh", "navigate"));
  const response = await Promise.race([
    responsePromise,
    new Promise((_, reject) => setTimeout(() => reject(new Error("timed out")), 50)),
  ]);
  assert.equal(await response.text(), "cached-shell");
});

test("cached asset wins over a bad network response", async () => {
  const harness = await loadServiceWorker(() => textResponse("bad gateway", { status: 502 }));
  await cacheShell(harness, "/assets/index.js", "cached-js");
  const { responsePromise } = dispatch(harness.handlers.fetch, request("/assets/index.js"));
  const response = await responsePromise;
  assert.equal(await response.text(), "cached-js");
});

test("offline app-shell manifest requests are not intercepted", async () => {
  const harness = await loadServiceWorker(() => textResponse("network"));
  assert.equal(dispatch(harness.handlers.fetch, request("/offline/app-shell")).responsePromise, undefined);
});

test("API requests are not intercepted by the app-shell cache", async () => {
  const harness = await loadServiceWorker(() => textResponse("network"));
  assert.equal(dispatch(harness.handlers.fetch, request("/api/cram/overview")).responsePromise, undefined);
});

test("audio artifacts stay cache-first", async () => {
  const harness = await loadServiceWorker(() => {
    throw new Error("network down");
  });
  const cache = await harness.caches.open("xuezh-audio-v1");
  await cache.put("/artifacts/sentence.ogg", textResponse("cached-audio"));
  const { responsePromise } = dispatch(harness.handlers.fetch, request("/artifacts/sentence.ogg"));
  const response = await responsePromise;
  assert.equal(await response.text(), "cached-audio");
});
