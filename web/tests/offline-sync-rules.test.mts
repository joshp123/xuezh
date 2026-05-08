import assert from "node:assert/strict";
import test from "node:test";
import { heldUndoEventID, syncableOfflineEvents, type OfflineSyncStore } from "../src/offlineSyncRules.ts";

const events = (...ids: string[]) => ids.map((event_id) => ({ event_id }));

function store(status: "active" | "done", undoIDs: string[]): OfflineSyncStore {
  return {
    session: { status, card: status === "active" ? { item_id: "current" } : null },
    undo: undoIDs.map((event_id) => ({ event_id }))
  };
}

test("no active review means every pending event can sync", () => {
  assert.deepEqual(syncableOfflineEvents(events("a", "b"), store("done", ["b"])), events("a", "b"));
});

test("active review with no undo target syncs every pending event", () => {
  assert.deepEqual(syncableOfflineEvents(events("a", "b"), store("active", [])), events("a", "b"));
});

test("active review holds exactly the latest undo event", () => {
  assert.equal(heldUndoEventID(store("active", ["c", "b", "a"])), "c");
  assert.deepEqual(syncableOfflineEvents(events("a", "b", "c"), store("active", ["c", "b", "a"])), events("a", "b"));
});

test("single latest pending event stays local during active review", () => {
  assert.deepEqual(syncableOfflineEvents(events("a"), store("active", ["a"])), []);
});

test("missing undo event does not block older pending events", () => {
  assert.deepEqual(syncableOfflineEvents(events("a", "b"), store("active", ["c", "b", "a"])), events("a", "b"));
});

