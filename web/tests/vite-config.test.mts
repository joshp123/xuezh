import assert from "node:assert/strict";
import test from "node:test";
import config from "../vite.config.ts";

test("production builds retain previous hashed assets for installed PWAs", () => {
  assert.equal(config.build?.emptyOutDir, false);
});
