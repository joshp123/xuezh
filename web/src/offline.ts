import type {
  Card,
  Filters,
  OfflineDeckCard,
  OfflineDeckSnapshot,
  OfflineReviewEvent,
  OfflineSaveProgress,
  OfflineSaveState,
  OfflineStorageInfo,
  PracticeCard,
  PracticeCategory,
  PracticePreview,
  PracticeSource,
  ReviewAnswer,
  ReviewedCard,
  ReviewSessionState,
  ScoreBuckets
} from "./types";
import { categoryKey, sourceLabel } from "./utils";

const dbName = "xuezh-offline-v1";
const kvStore = "kv";
const eventStore = "events";
const appCacheName = "xuezh-app-v1";
const audioCacheName = "xuezh-audio-v1";

export type OfflineSessionStore = {
  session: ReviewSessionState;
  undo: Array<{ event_id: string; session_before: ReviewSessionState; deck_before: OfflineDeckSnapshot }>;
};

export async function registerOfflineApp() {
  if ("serviceWorker" in navigator) {
    await navigator.serviceWorker.register("/sw.js");
  }
}

export async function saveOfflineDeck(snapshot: OfflineDeckSnapshot, onProgress?: (progress: OfflineSaveProgress) => void, cacheAudio = true) {
  await putKV("deck", snapshot);
  await cacheAppShell();
  const previous = await loadOfflineSaveState();
  let audioSaved = cacheAudio ? 0 : previous?.audio_saved ?? 0;
  let audioMissing = cacheAudio ? snapshot.audio_paths.length : previous?.audio_missing ?? snapshot.audio_paths.length;
  if (cacheAudio && "caches" in window) {
    const cache = await caches.open(audioCacheName);
    const total = snapshot.audio_paths.length;
    let done = 0;
    let saved = 0;
    let missing = 0;
    for (const path of snapshot.audio_paths) {
      const request = `/${path}`;
      try {
        const cached = await cache.match(request);
        if (!cached) await cache.add(request);
        saved++;
      } catch {
        // Missing audio should not block text/card offline use.
        missing++;
      } finally {
        done++;
        onProgress?.({ done, total, saved, missing });
      }
    }
    audioSaved = saved;
    audioMissing = missing;
  }
  const state: OfflineSaveState = {
    saved_at: new Date().toISOString(),
    card_count: snapshot.cards.length,
    audio_total: snapshot.audio_paths.length,
    audio_saved: audioSaved,
    audio_missing: audioMissing,
    storage: await inspectOfflineStorage(cacheAudio)
  };
  await putKV("offline_state", state);
  return state;
}

async function cacheAppShell() {
  if (!("caches" in window)) return;
  const response = await fetch("/offline/app-shell", { cache: "no-store" });
  if (!response.ok) throw new Error(await response.text());
  const body = (await response.json()) as { assets?: string[] };
  const assets = body.assets ?? ["/xuezh"];
  const cache = await caches.open(appCacheName);
  const missing: string[] = [];
  for (const asset of assets) {
    try {
      const assetResponse = await fetch(asset, { cache: "no-store" });
      if (!assetResponse.ok) throw new Error(`${assetResponse.status}`);
      await cache.put(asset, assetResponse.clone());
    } catch {
      if (!(await cache.match(asset))) missing.push(asset);
    }
  }
  const requiredMissing = missing.filter(isRequiredAppAsset);
  if (requiredMissing.length > 0) {
    throw new Error(`Could not save app for offline use (${requiredMissing.length} files missing).`);
  }
}

function isRequiredAppAsset(asset: string) {
  return asset === "/xuezh" || asset === "/index.html" || asset.startsWith("/assets/");
}

export async function loadOfflineDeck() {
  return await getKV<OfflineDeckSnapshot>("deck");
}

export async function loadOfflineSaveState() {
  return await getKV<OfflineSaveState>("offline_state");
}

export async function inspectOfflineStorage(requestPersistence = false): Promise<OfflineStorageInfo> {
  const storage = navigator.storage;
  let persisted: boolean | null = null;
  let usage: number | null = null;
  let quota: number | null = null;
  if (storage?.persisted) {
    try {
      persisted = await storage.persisted();
    } catch {
      persisted = null;
    }
  }
  if (requestPersistence && storage?.persist) {
    try {
      persisted = await storage.persist();
    } catch {
      // Leave the prior persisted value in place.
    }
  }
  if (storage?.estimate) {
    try {
      const estimate = await storage.estimate();
      usage = typeof estimate.usage === "number" ? estimate.usage : null;
      quota = typeof estimate.quota === "number" ? estimate.quota : null;
    } catch {
      usage = null;
      quota = null;
    }
  }
  return { persisted, usage_bytes: usage, quota_bytes: quota };
}

export async function saveOfflineSession(session: OfflineSessionStore | null) {
  if (session) await putKV("session", session);
  else await deleteKV("session");
}

export async function loadOfflineSession() {
  return await getKV<OfflineSessionStore>("session");
}

export async function appendOfflineEvent(event: OfflineReviewEvent) {
  const db = await openOfflineDB();
  await requestDone(db.transaction(eventStore, "readwrite").objectStore(eventStore).put(event));
}

export async function deleteOfflineEvent(eventID: string) {
  const db = await openOfflineDB();
  await requestDone(db.transaction(eventStore, "readwrite").objectStore(eventStore).delete(eventID));
}

export async function pendingOfflineEvents() {
  const db = await openOfflineDB();
  const events = await requestDone<OfflineReviewEvent[]>(db.transaction(eventStore).objectStore(eventStore).getAll());
  return events.sort((left, right) => {
    const time = Date.parse(left.answered_at) - Date.parse(right.answered_at);
    return time !== 0 ? time : left.event_id.localeCompare(right.event_id);
  });
}

export async function markOfflineEventsSynced(eventIDs: string[]) {
  for (const eventID of eventIDs) await deleteOfflineEvent(eventID);
}

export function offlineOverview(snapshot: OfflineDeckSnapshot) {
  const totals = new Map<string, { source: string; label: string; total_count: number }>();
  for (const card of snapshot.cards) {
    const row = totals.get(card.source) ?? { source: card.source, label: sourceLabel(card.source), total_count: 0 };
    row.total_count++;
    totals.set(card.source, row);
  }
  return { generated_at: snapshot.generated_at, sources: [...totals.values()] };
}

export function offlinePracticePreview(snapshot: OfflineDeckSnapshot, filters: Filters, now = new Date()): PracticePreview {
  const sourceMap = new Map<string, PracticeSource>();
  const categoryMap = new Map<string, PracticeCategory>();
  const categoryOrder: string[] = [];
  const cards: PracticeCard[] = [];
  for (const deckCard of snapshot.cards) {
    const card = practiceCardFromDeck(deckCard, snapshot, filters, now);
    const source = ensureSource(sourceMap, deckCard.source);
    const key = categoryKey(deckCard);
    const category = ensureCategory(categoryMap, categoryOrder, deckCard);
    const matches = matchesFilters(card, filters);
    updateTotals(source, card, deckCard, snapshot, matches);
    updateTotals(category, card, deckCard, snapshot, matches);
    if (matches) cards.push(card);
  }
  cards.sort(practiceCardCompare);
  return {
    generated_at: new Date().toISOString(),
    sources: [sourceMap.get("hellochinese"), sourceMap.get("travel_survival")].filter(Boolean) as PracticeSource[],
    categories: categoryOrder.map((key) => categoryMap.get(key)!).filter(Boolean),
    cards
  };
}

export async function startOfflineReviewSession(cardIDs: string[], snapshot: OfflineDeckSnapshot, limit: number) {
  const cardsByID = new Map(snapshot.cards.map((card) => [card.item_id, card]));
  const cards = cardIDs.map((id) => cardsByID.get(id)).filter(Boolean).map((card) => cardToReviewCard(card!));
  const picked = cards.slice(0, limit === 0 ? cards.length : limit);
  const now = new Date().toISOString();
  const session: ReviewSessionState = {
    id: `offline-${crypto.randomUUID()}`,
    status: picked.length > 0 ? "active" : "done",
    card: picked[0] ?? null,
    queue: picked.slice(1),
    retry_queue: [],
    reviewed_cards: [],
    repeat_item_ids: [],
    revealed: false,
    round: 1,
    was_retry: false,
    shown_at: picked.length > 0 ? now : null,
    updated_at: now
  };
  const store = { session, undo: [] };
  await saveOfflineSession(store);
  return store;
}

export async function revealOfflineSession(store: OfflineSessionStore) {
  const next = cloneStore(store);
  next.session.revealed = true;
  next.session.updated_at = new Date().toISOString();
  await saveOfflineSession(next);
  return next;
}

export async function toggleOfflineRepeat(store: OfflineSessionStore) {
  const next = cloneStore(store);
  const id = next.session.card?.item_id;
  if (!id) return next;
  next.session.repeat_item_ids = next.session.repeat_item_ids.includes(id)
    ? next.session.repeat_item_ids.filter((value) => value !== id)
    : [...next.session.repeat_item_ids, id];
  next.session.updated_at = new Date().toISOString();
  await saveOfflineSession(next);
  return next;
}

export async function gradeOfflineSession(store: OfflineSessionStore, snapshot: OfflineDeckSnapshot, grade: ReviewAnswer) {
  if (!store.session.card || !store.session.revealed) return { store, snapshot };
  const now = new Date().toISOString();
  const current = store.session.card;
  const event: OfflineReviewEvent = {
    event_id: `offline-${crypto.randomUUID()}`,
    session_id: store.session.id,
    item_id: current.item_id,
    grade,
    shown_at: store.session.shown_at ?? "",
    answered_at: now,
    elapsed_ms: store.session.shown_at ? Math.max(0, Date.parse(now) - Date.parse(store.session.shown_at)) : 0,
    round: store.session.round,
    was_retry: store.session.was_retry
  };
  const nextSnapshot = applyLocalScore(snapshot, event);
  const nextStore = cloneStore(store);
  nextStore.undo.unshift({ event_id: event.event_id, session_before: cloneSession(store.session), deck_before: snapshot });
  const repeat = nextStore.session.repeat_item_ids.includes(current.item_id);
  nextStore.session.reviewed_cards = [{ card: current, grade, repeat }, ...nextStore.session.reviewed_cards];
  nextStore.session.repeat_item_ids = nextStore.session.repeat_item_ids.filter((id) => id !== current.item_id);
  const retry = grade === "incorrect" || repeat ? [current] : [];
  advanceOfflineSession(nextStore.session, retry, now);
  await appendOfflineEvent(event);
  await putKV("deck", nextSnapshot);
  await saveOfflineSession(nextStore);
  return { store: nextStore, snapshot: nextSnapshot };
}

export async function undoOfflineSession(store: OfflineSessionStore) {
  const last = store.undo[0];
  if (!last) return { store, snapshot: await loadOfflineDeck() };
  await deleteOfflineEvent(last.event_id);
  await putKV("deck", last.deck_before);
  const next = { session: cloneSession(last.session_before), undo: store.undo.slice(1) };
  await saveOfflineSession(next);
  return { store: next, snapshot: last.deck_before };
}

function advanceOfflineSession(session: ReviewSessionState, retry: Card[], now: string) {
  session.retry_queue = [...session.retry_queue, ...retry];
  const next = session.queue[0];
  if (next) {
    session.card = next;
    session.queue = session.queue.slice(1);
    session.revealed = false;
    session.was_retry = false;
    session.shown_at = now;
  } else if (session.retry_queue.length > 0) {
    session.card = session.retry_queue[0];
    session.retry_queue = session.retry_queue.slice(1);
    session.revealed = false;
    session.was_retry = true;
    session.round++;
    session.shown_at = now;
  } else {
    session.card = null;
    session.status = "done";
    session.revealed = false;
    session.shown_at = null;
  }
  session.updated_at = now;
}

function applyLocalScore(snapshot: OfflineDeckSnapshot, event: OfflineReviewEvent): OfflineDeckSnapshot {
  const settings = localScoringSettings(snapshot);
  const cards = snapshot.cards.map((card) => {
    if (card.item_id !== event.item_id) return card;
    if (settings.score_once_per_day && card.last_reviewed_at && sameDay(card.last_reviewed_at, event.answered_at)) return card;
    const difficulty = clamp(
      (card.difficulty || 100) + (event.grade === "correct" ? settings.correct_difficulty_change : settings.incorrect_difficulty_change),
      settings.min_difficulty,
      settings.max_difficulty
    );
    const score = event.grade === "correct"
      ? clamp(
        Math.max(settings.correct_initial_score, Math.round(Math.max(card.score ?? settings.min_score, settings.min_score) * difficulty / settings.difficulty_divisor * settings.correct_multiplier / 100)),
        settings.min_score,
        settings.max_score
      )
      : clamp(settings.incorrect_score, settings.min_score, settings.max_score);
    const nextDue = new Date(Date.parse(event.answered_at) + (score / settings.points_per_day) * 24 * 60 * 60 * 1000).toISOString();
    return {
      ...card,
      score,
      difficulty,
      correct_count: card.correct_count + (event.grade === "correct" ? 1 : 0),
      incorrect_count: card.incorrect_count + (event.grade === "incorrect" ? 1 : 0),
      reviewed_count: card.reviewed_count + 1,
      first_reviewed_at: card.first_reviewed_at ?? event.answered_at,
      last_reviewed_at: event.answered_at,
      due_at: nextDue
    };
  });
  return { ...snapshot, generated_at: new Date().toISOString(), cards, settings };
}

function localScoringSettings(snapshot: OfflineDeckSnapshot): OfflineDeckSnapshot["settings"] {
  return {
    learned_score: snapshot.settings.learned_score || 200,
    min_score: snapshot.settings.min_score || 100,
    max_score: snapshot.settings.max_score || 51200,
    min_difficulty: snapshot.settings.min_difficulty || 50,
    max_difficulty: snapshot.settings.max_difficulty || 200,
    points_per_day: snapshot.settings.points_per_day || 100,
    score_once_per_day: snapshot.settings.score_once_per_day ?? true,
    incorrect_score: snapshot.settings.incorrect_score || 100,
    correct_initial_score: snapshot.settings.correct_initial_score || 600,
    correct_multiplier: snapshot.settings.correct_multiplier || 110,
    correct_difficulty_change: snapshot.settings.correct_difficulty_change || 4,
    incorrect_difficulty_change: snapshot.settings.incorrect_difficulty_change || -10,
    difficulty_divisor: snapshot.settings.difficulty_divisor || 40
  };
}

function practiceCardFromDeck(card: OfflineDeckCard, snapshot: OfflineDeckSnapshot, filters: Filters, now: Date): PracticeCard {
  return {
    item_id: card.item_id,
    source: card.source,
    source_label: sourceLabel(card.source),
    category: card.category,
    learning_order: card.learning_order,
    word: card.word,
    sentence_hanzi: card.sentence_hanzi,
    score: card.score,
    correct_count: card.correct_count,
    incorrect_count: card.incorrect_count,
    reviewed_count: card.reviewed_count,
    last_reviewed_at: card.last_reviewed_at,
    not_learned: card.score === null ? filters.include_no_score : card.score < filters.score_below,
    due: Boolean(card.due_at && Date.parse(card.due_at) <= now.getTime()),
    got_wrong: card.incorrect_count > filters.misses_more_than
  };
}

function updateTotals(row: PracticeSource | PracticeCategory, card: PracticeCard, deckCard: OfflineDeckCard, snapshot: OfflineDeckSnapshot, matches: boolean) {
  row.total_count++;
  if (deckCard.score === null || deckCard.score < snapshot.settings.learned_score) row.not_learned_count++;
  if (card.due) row.due_count++;
  if (card.got_wrong) row.got_wrong_count++;
  addScoreBucket(row.score_buckets, deckCard.score);
  if (matches) row.practice_count++;
}

function matchesFilters(card: PracticeCard, filters: Filters) {
  if (!filters.include_not_learned && !filters.include_due && !filters.include_got_wrong) return true;
  return (filters.include_not_learned && card.not_learned) ||
    (filters.include_due && card.due) ||
    (filters.include_got_wrong && card.got_wrong);
}

function ensureSource(map: Map<string, PracticeSource>, source: string) {
  let row = map.get(source);
  if (!row) {
    row = { source, label: sourceLabel(source), total_count: 0, practice_count: 0, not_learned_count: 0, due_count: 0, got_wrong_count: 0, score_buckets: emptyBuckets() };
    map.set(source, row);
  }
  return row;
}

function ensureCategory(map: Map<string, PracticeCategory>, order: string[], card: OfflineDeckCard) {
  const key = categoryKey(card);
  let row = map.get(key);
  if (!row) {
    row = { source: card.source, source_label: sourceLabel(card.source), category: card.category, total_count: 0, practice_count: 0, not_learned_count: 0, due_count: 0, got_wrong_count: 0, score_buckets: emptyBuckets() };
    map.set(key, row);
    order.push(key);
  }
  return row;
}

function emptyBuckets(): ScoreBuckets {
  return { no_score: 0, score_under_100: 0, score_100_to_199: 0, score_200_to_599: 0, score_600_to_1599: 0, score_1600_to_6399: 0, score_6400_plus: 0 };
}

function addScoreBucket(buckets: ScoreBuckets, score: number | null) {
  if (score === null) buckets.no_score++;
  else if (score < 100) buckets.score_under_100++;
  else if (score < 200) buckets.score_100_to_199++;
  else if (score < 600) buckets.score_200_to_599++;
  else if (score < 1600) buckets.score_600_to_1599++;
  else if (score < 6400) buckets.score_1600_to_6399++;
  else buckets.score_6400_plus++;
}

function practiceCardCompare(left: PracticeCard, right: PracticeCard) {
  if (left.not_learned !== right.not_learned) return left.not_learned ? -1 : 1;
  if (left.due !== right.due) return left.due ? -1 : 1;
  if (left.incorrect_count !== right.incorrect_count) return right.incorrect_count - left.incorrect_count;
  if ((left.score === null) !== (right.score === null)) return left.score === null ? -1 : 1;
  const leftScore = left.score ?? Number.MAX_SAFE_INTEGER;
  const rightScore = right.score ?? Number.MAX_SAFE_INTEGER;
  if (leftScore !== rightScore) return leftScore - rightScore;
  if (left.source !== right.source) return left.source.localeCompare(right.source);
  return left.learning_order - right.learning_order;
}

function cardToReviewCard(card: OfflineDeckCard): Card {
  const { score, difficulty, correct_count, incorrect_count, reviewed_count, first_reviewed_at, last_reviewed_at, due_at, ...reviewCard } = card;
  return reviewCard;
}

function cloneStore(store: OfflineSessionStore): OfflineSessionStore {
  return structuredClone(store);
}

function cloneSession(session: ReviewSessionState): ReviewSessionState {
  return structuredClone(session);
}

function sameDay(left: string, right: string) {
  const a = new Date(left);
  const b = new Date(right);
  return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
}

function clamp(value: number, min: number, max: number) {
  return Math.max(min, Math.min(max, value));
}

async function openOfflineDB() {
  return await new Promise<IDBDatabase>((resolve, reject) => {
    const request = indexedDB.open(dbName, 1);
    request.onupgradeneeded = () => {
      const db = request.result;
      if (!db.objectStoreNames.contains(kvStore)) db.createObjectStore(kvStore);
      if (!db.objectStoreNames.contains(eventStore)) db.createObjectStore(eventStore, { keyPath: "event_id" });
    };
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error);
  });
}

async function putKV<T>(key: string, value: T) {
  const db = await openOfflineDB();
  await requestDone(db.transaction(kvStore, "readwrite").objectStore(kvStore).put(value, key));
}

async function getKV<T>(key: string) {
  const db = await openOfflineDB();
  return await requestDone<T | null>(db.transaction(kvStore).objectStore(kvStore).get(key));
}

async function deleteKV(key: string) {
  const db = await openOfflineDB();
  await requestDone(db.transaction(kvStore, "readwrite").objectStore(kvStore).delete(key));
}

async function requestDone<T = unknown>(request: IDBRequest<T>) {
  return await new Promise<T>((resolve, reject) => {
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error);
  });
}
