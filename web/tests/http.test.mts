import assert from "node:assert/strict";
import test from "node:test";
import { fetchWithTimeout } from "../src/http.ts";

const originalFetch = globalThis.fetch;

test.afterEach(() => {
  globalThis.fetch = originalFetch;
});

test("fetchWithTimeout returns successful responses", async () => {
  globalThis.fetch = async () => new Response("ok", { status: 200 });

  const response = await fetchWithTimeout("/ok", {}, 50);

  assert.equal(response.status, 200);
  assert.equal(await response.text(), "ok");
});

test("fetchWithTimeout aborts hung requests", async () => {
  globalThis.fetch = async (_input, init) => new Promise<Response>((_resolve, reject) => {
    init?.signal?.addEventListener("abort", () => {
      const err = new Error("aborted");
      err.name = "AbortError";
      reject(err);
    });
  });

  await assert.rejects(
    () => fetchWithTimeout("/hang", {}, 5),
    /Network timed out/
  );
});
