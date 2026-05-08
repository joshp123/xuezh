export type OfflineSyncEvent = { event_id: string };

export type OfflineSyncStore = {
  session: { status: "active" | "done"; card: unknown | null };
  undo: Array<{ event_id: string }>;
} | null | undefined;

export function heldUndoEventID(store: OfflineSyncStore) {
  if (!store || store.session.status !== "active" || !store.session.card) return null;
  return store.undo[0]?.event_id ?? null;
}

export function syncableOfflineEvents<T extends OfflineSyncEvent>(events: T[], store: OfflineSyncStore) {
  const heldID = heldUndoEventID(store);
  if (!heldID) return events;
  return events.filter((event) => event.event_id !== heldID);
}

