import { StrictMode, Suspense, lazy, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createRoot } from "react-dom/client";
import { BatchPicker } from "./BatchPicker";
import { ReviewCard, SessionHistoryDialog } from "./ReviewSession";
import {
  gradeOfflineSession,
  inspectOfflineStorage,
  loadOfflineDeck,
  loadOfflineSaveState,
  loadOfflineSession,
  markOfflineEventsSynced,
  offlineOverview,
  offlinePracticePreview,
  pendingOfflineEvents,
  registerOfflineApp,
  revealOfflineSession,
  saveOfflineDeck,
  startOfflineReviewSession,
  toggleOfflineRepeat,
  undoOfflineSession
} from "./offline";
import { syncableOfflineEvents } from "./offlineSyncRules";
import { Shell, State } from "./shared";
import type { OfflineSessionStore } from "./offline";
import type { Card, Filters, OfflineDeckSnapshot, OfflineSaveState, OfflineSyncResult, Overview, PracticePreview, ReviewAnswer, ReviewSessionState, View } from "./types";
import { defaultFilters } from "./types";
import { addSet, hashOffset, removeSet, toggleSet } from "./utils";
import "./styles.css";

const DesignSystemPage = lazy(() => import("./dev/DesignSystemPage").then((module) => ({ default: module.DesignSystemPage })));

function App() {
  const [hash, setHash] = useState(location.hash);

  useEffect(() => {
    const setViewportHeight = () => {
      const height = window.visualViewport?.height ?? window.innerHeight;
      document.documentElement.style.setProperty("--app-height", `${height}px`);
    };
    const onHashChange = () => setHash(location.hash);
    setViewportHeight();
    window.addEventListener("hashchange", onHashChange);
    window.addEventListener("resize", setViewportHeight);
    window.visualViewport?.addEventListener("resize", setViewportHeight);
    window.visualViewport?.addEventListener("scroll", setViewportHeight);
    return () => {
      window.removeEventListener("hashchange", onHashChange);
      window.removeEventListener("resize", setViewportHeight);
      window.visualViewport?.removeEventListener("resize", setViewportHeight);
      window.visualViewport?.removeEventListener("scroll", setViewportHeight);
    };
  }, []);

  if (hash.startsWith("#design-system")) {
    return (
      <Suspense fallback={<Shell><State title="Loading" body="Preparing design system." /></Shell>}>
        <DesignSystemPage />
      </Suspense>
    );
  }

  return <CramApp />;
}

function CramApp() {
  const [overview, setOverview] = useState<Overview | null>(null);
  const [preview, setPreview] = useState<PracticePreview | null>(null);
  const [filters, setFilters] = useState<Filters>(defaultFilters);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [selectionTouched, setSelectionTouched] = useState(false);
  const [openSources, setOpenSources] = useState<Set<string>>(new Set());
  const [openCategories, setOpenCategories] = useState<Set<string>>(new Set());
  const [view, setView] = useState<View>("batch");
  const [cap, setCap] = useState(100);
  const [session, setSession] = useState<ReviewSessionState | null>(null);
  const [sessionLoaded, setSessionLoaded] = useState(false);
  const [historyOpen, setHistoryOpen] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [offlineDeck, setOfflineDeck] = useState<OfflineDeckSnapshot | null>(null);
  const [offlineStore, setOfflineStore] = useState<OfflineSessionStore | null>(null);
  const [offlineMode, setOfflineMode] = useState(false);
  const [offlineBusy, setOfflineBusy] = useState(false);
  const [offlineSyncing, setOfflineSyncing] = useState(false);
  const [offlineSaveState, setOfflineSaveState] = useState<OfflineSaveState | null>(null);
  const [offlineError, setOfflineError] = useState<string | null>(null);
  const [pendingOfflineCount, setPendingOfflineCount] = useState(0);
  const [syncableOfflineCount, setSyncableOfflineCount] = useState(0);
  const [audioPath, setAudioPath] = useState<string | null>(null);
  const [audioIndex, setAudioIndex] = useState(0);
  const audioRef = useRef<HTMLAudioElement | null>(null);

  const card = session?.card ?? null;
  const queue = session?.queue ?? [];
  const retryQueue = session?.retry_queue ?? [];
  const reviewedCards = session?.reviewed_cards ?? [];
  const repeatCards = useMemo(() => new Set(session?.repeat_item_ids ?? []), [session?.repeat_item_ids]);
  const revealed = session?.revealed ?? false;
  const shownAt = session?.shown_at ?? null;
  const hasActiveReview = Boolean(session?.status === "active" && card);

  const resumeOfflineStore = useCallback((store: OfflineSessionStore, deck: OfflineDeckSnapshot) => {
    setOfflineDeck(deck);
    setOfflineStore(store);
    setOfflineMode(true);
    setOverview(offlineOverview(deck));
    setPreview(offlinePracticePreview(deck, filters));
    setSession(store.session);
    setView("review");
  }, [filters]);

  const loadOfflineFallback = useCallback(async () => {
    const deck = await loadOfflineDeck();
    if (!deck) return false;
    const store = await loadOfflineSession();
    setOfflineDeck(deck);
    setOfflineStore(store);
    setOfflineMode(true);
    setOverview(offlineOverview(deck));
    setPreview(offlinePracticePreview(deck, filters));
    const pending = await pendingOfflineEvents();
    setPendingOfflineCount(pending.length);
    setSyncableOfflineCount(syncableOfflineEvents(pending, store).length);
    if (store?.session.status === "active" && store.session.card) {
      resumeOfflineStore(store, deck);
    }
    return true;
  }, [filters, resumeOfflineStore]);

  const loadActiveOfflineStore = useCallback(async () => {
    const localStore = await loadOfflineSession();
    const localDeck = await loadOfflineDeck();
    if (localStore?.session.status === "active" && localStore.session.card && localDeck) {
      resumeOfflineStore(localStore, localDeck);
      return true;
    }
    return false;
  }, [resumeOfflineStore]);

  const refreshOverview = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      if (await loadActiveOfflineStore()) return;
      const response = await fetch("/api/cram/overview");
      if (!response.ok) throw new Error(await response.text());
      setOverview((await response.json()) as Overview);
      setOfflineMode(false);
    } catch (err) {
      if (!(await loadOfflineFallback())) {
        setError(err instanceof Error ? err.message : String(err));
      }
    } finally {
      setLoading(false);
    }
  }, [loadActiveOfflineStore, loadOfflineFallback]);

  const refreshPreview = useCallback(async () => {
    setError(null);
    if (offlineMode && offlineDeck) {
      setPreview(offlinePracticePreview(offlineDeck, filters));
      return;
    }
    try {
      const response = await fetch("/api/cram/preview", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(filters)
      });
      if (!response.ok) throw new Error(await response.text());
      setPreview((await response.json()) as PracticePreview);
    } catch (err) {
      if (!(await loadOfflineFallback())) {
        setError(err instanceof Error ? err.message : String(err));
      }
    }
  }, [filters, loadOfflineFallback, offlineDeck, offlineMode]);

  const loadActiveSession = useCallback(async () => {
    setError(null);
    try {
      if (await loadActiveOfflineStore()) return;
      const response = await fetch("/api/cram/session");
      if (!response.ok) throw new Error(await response.text());
      const body = (await response.json()) as { session: ReviewSessionState | null };
      if (body.session?.status === "active" && body.session.card) {
        setSession(body.session);
        setView("review");
      }
    } catch (err) {
      if (!(await loadOfflineFallback())) {
        setError(err instanceof Error ? err.message : String(err));
      }
    } finally {
      setSessionLoaded(true);
    }
  }, [loadActiveOfflineStore, loadOfflineFallback]);

  const syncOfflineAnswers = useCallback(async () => {
    const events = await pendingOfflineEvents();
    setPendingOfflineCount(events.length);
    const localStore = offlineStore ?? await loadOfflineSession();
    const eventsToSync = syncableOfflineEvents(events, localStore);
    setSyncableOfflineCount(eventsToSync.length);
    if (!navigator.onLine) return;
    if (eventsToSync.length === 0) return;
    setOfflineSyncing(true);
    setOfflineError(null);
    try {
      const response = await fetch("/api/cram/offline/sync", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ events: eventsToSync })
      });
      if (!response.ok) throw new Error(await response.text());
      const result = (await response.json()) as OfflineSyncResult;
      await markOfflineEventsSynced([...result.applied_event_ids, ...result.skipped_event_ids]);
      const remaining = await pendingOfflineEvents();
      setPendingOfflineCount(remaining.length);
      const remainingSyncable = syncableOfflineEvents(remaining, localStore);
      setSyncableOfflineCount(remainingSyncable.length);
      if (remaining.length === 0) {
        const deckResponse = await fetch("/api/cram/offline/deck");
        if (deckResponse.ok) {
          const deck = (await deckResponse.json()) as OfflineDeckSnapshot;
          const state = await saveOfflineDeck(deck, undefined, false);
          setOfflineSaveState(state);
          setOfflineDeck(deck);
          if (offlineMode) {
            setOverview(offlineOverview(deck));
            setPreview(offlinePracticePreview(deck, filters));
          }
        }
      }
    } catch (err) {
      setOfflineError(err instanceof Error ? err.message : String(err));
    } finally {
      setOfflineSyncing(false);
    }
  }, [filters, offlineMode, offlineStore]);

  const prepareOffline = useCallback(async () => {
    setOfflineBusy(true);
    setOfflineError(null);
    setError(null);
    try {
      await registerOfflineApp();
      const response = await fetch("/api/cram/offline/deck");
      if (!response.ok) throw new Error(await response.text());
      const deck = (await response.json()) as OfflineDeckSnapshot;
      const state = await saveOfflineDeck(deck, (progress) => {
        if (progress.done !== progress.total && progress.done % 25 !== 0) return;
        setOfflineSaveState({
          saved_at: new Date().toISOString(),
          card_count: deck.cards.length,
          audio_total: progress.total,
          audio_saved: progress.saved,
          audio_missing: progress.missing,
          storage: { persisted: null, usage_bytes: null, quota_bytes: null }
        });
      });
      setOfflineDeck(deck);
      setOfflineSaveState(state);
      setPreview(offlinePracticePreview(deck, filters));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setOfflineError(err instanceof Error ? err.message : String(err));
    } finally {
      setOfflineBusy(false);
    }
  }, [filters]);

  useEffect(() => { void refreshOverview(); }, [refreshOverview]);
  useEffect(() => { void loadActiveSession(); }, [loadActiveSession]);
  useEffect(() => {
    void registerOfflineApp().catch(() => undefined);
    void loadOfflineDeck().then((deck) => {
      if (deck) {
        setOfflineDeck(deck);
      }
    });
    void loadOfflineSaveState().then(setOfflineSaveState);
    void inspectOfflineStorage().then((storage) => {
      setOfflineSaveState((current) => current ? { ...current, storage } : current);
    });
    void Promise.all([pendingOfflineEvents(), loadOfflineSession()]).then(([events, store]) => {
      setPendingOfflineCount(events.length);
      setSyncableOfflineCount(syncableOfflineEvents(events, store).length);
    });
  }, []);
  useEffect(() => {
    const onOnline = () => { void syncOfflineAnswers().catch((err) => setError(err instanceof Error ? err.message : String(err))); };
    const onFocus = () => { void syncOfflineAnswers().catch(() => undefined); };
    const onVisible = () => {
      if (document.visibilityState === "visible") void syncOfflineAnswers().catch(() => undefined);
    };
    window.addEventListener("online", onOnline);
    window.addEventListener("focus", onFocus);
    document.addEventListener("visibilitychange", onVisible);
    void syncOfflineAnswers().catch(() => undefined);
    return () => {
      window.removeEventListener("online", onOnline);
      window.removeEventListener("focus", onFocus);
      document.removeEventListener("visibilitychange", onVisible);
    };
  }, [syncOfflineAnswers]);
  useEffect(() => {
    if (pendingOfflineCount === 0) return;
    const interval = window.setInterval(() => {
      void syncOfflineAnswers().catch(() => undefined);
    }, 30_000);
    return () => window.clearInterval(interval);
  }, [pendingOfflineCount, syncOfflineAnswers]);
  useEffect(() => {
    if (offlineMode && offlineStore?.session.status !== "active") {
      void syncOfflineAnswers().catch((err) => setError(err instanceof Error ? err.message : String(err)));
    }
  }, [offlineMode, offlineStore?.session.status, syncOfflineAnswers]);
  useEffect(() => { if (view === "batch") void refreshPreview(); }, [refreshPreview, view]);
  useEffect(() => {
    if (!preview) return;
    const visible = new Set(preview.cards.map((item) => item.item_id));
    setSelected((current) => !selectionTouched ? visible : new Set([...current].filter((id) => visible.has(id))));
  }, [preview, selectionTouched]);

  const selectedCards = useMemo(() => preview?.cards.filter((item) => selected.has(item.item_id)) ?? [], [preview, selected]);
  const batchLimit = cap === 0 ? selectedCards.length : cap;
  const batchSize = Math.min(batchLimit, selectedCards.length);

  const audioChoices = useMemo(() => {
    if (!card) return null;
    const paths = Object.entries(card.sentence_audio_paths ?? {})
      .sort(([left], [right]) => left.localeCompare(right))
      .map(([, path]) => path);
    if (paths.length === 0) return null;
    const offset = hashOffset(card.item_id, paths.length);
    return [...paths.slice(offset), ...paths.slice(0, offset)];
  }, [card?.item_id]);

  const playCurrentAudio = useCallback(() => {
    if (!audioRef.current) return;
    audioRef.current.currentTime = 0;
    void audioRef.current.play().catch(() => {
      // Browsers block autoplay until the first user gesture. Replay still works.
    });
  }, []);

  useEffect(() => setAudioIndex(0), [audioChoices]);
  useEffect(() => setAudioPath(audioChoices ? `/${audioChoices[audioIndex % audioChoices.length]}` : null), [audioChoices, audioIndex]);
  useEffect(() => { if (audioPath) playCurrentAudio(); }, [audioPath, playCurrentAudio]);

  const replay = useCallback(() => {
    if (!audioChoices || audioChoices.length <= 1) playCurrentAudio();
    else setAudioIndex((index) => (index + 1) % audioChoices.length);
  }, [audioChoices, playCurrentAudio]);

  const applySession = useCallback((next: ReviewSessionState) => {
    setSession(next);
    setHistoryOpen(false);
    if (next.status === "active" && next.card) setView("review");
    else {
      setView("done");
      if (offlineMode && offlineDeck) {
        setOverview(offlineOverview(offlineDeck));
        setPreview(offlinePracticePreview(offlineDeck, filters));
      } else {
        void refreshOverview();
      }
    }
  }, [filters, offlineDeck, offlineMode, refreshOverview]);

  const openBatch = () => setView("batch");
  const resumeReview = () => { if (hasActiveReview) setView("review"); };

  const startBatch = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      if (offlineMode && offlineDeck) {
        const store = await startOfflineReviewSession(selectedCards.map((item) => item.item_id), offlineDeck, batchLimit);
        setOfflineStore(store);
        applySession(store.session);
        return;
      }
      const response = await fetch("/api/cram/session/start", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ limit: batchLimit, card_ids: selectedCards.map((item) => item.item_id) })
      });
      if (!response.ok) throw new Error(await response.text());
      const body = (await response.json()) as { session: ReviewSessionState };
      applySession(body.session);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, [applySession, batchLimit, offlineDeck, offlineMode, selectedCards]);

  const reveal = useCallback(async () => {
    if (!session || revealed) return;
    setError(null);
    try {
      if (offlineMode && offlineStore) {
        const store = await revealOfflineSession(offlineStore);
        setOfflineStore(store);
        setSession(store.session);
        return;
      }
      const response = await fetch("/api/cram/session/reveal", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ session_id: session.id })
      });
      if (!response.ok) throw new Error(await response.text());
      const body = (await response.json()) as { session: ReviewSessionState };
      setSession(body.session);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }, [offlineMode, offlineStore, revealed, session]);

  const toggleRepeat = useCallback(async () => {
    if (!session) return;
    setError(null);
    try {
      if (offlineMode && offlineStore) {
        const store = await toggleOfflineRepeat(offlineStore);
        setOfflineStore(store);
        setSession(store.session);
        return;
      }
      const response = await fetch("/api/cram/session/repeat", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ session_id: session.id })
      });
      if (!response.ok) throw new Error(await response.text());
      const body = (await response.json()) as { session: ReviewSessionState };
      setSession(body.session);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }, [offlineMode, offlineStore, session]);

  const undoLastGrade = useCallback(async () => {
    if (!session || reviewedCards.length === 0) return;
    setError(null);
    try {
      if (offlineMode && offlineStore) {
        const { store, snapshot } = await undoOfflineSession(offlineStore);
        setOfflineStore(store);
        setSession(store.session);
        if (snapshot) {
          setOfflineDeck(snapshot);
          setPreview(offlinePracticePreview(snapshot, filters));
        }
        const events = await pendingOfflineEvents();
        setPendingOfflineCount(events.length);
        setSyncableOfflineCount(syncableOfflineEvents(events, store).length);
        return;
      }
      const response = await fetch("/api/cram/session/undo", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ session_id: session.id })
      });
      if (!response.ok) throw new Error(await response.text());
      const body = (await response.json()) as { session: ReviewSessionState };
      applySession(body.session);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }, [applySession, filters, offlineMode, offlineStore, reviewedCards.length, session]);

  const grade = useCallback(async (value: ReviewAnswer) => {
    if (!session || !card || !revealed) return;
    const answeredAt = new Date().toISOString();
    const elapsedMs = shownAt ? Date.parse(answeredAt) - Date.parse(shownAt) : 0;
    setError(null);
    try {
      if (offlineMode && offlineStore && offlineDeck) {
        const { store, snapshot } = await gradeOfflineSession(offlineStore, offlineDeck, value);
        setOfflineStore(store);
        setOfflineDeck(snapshot);
        setPreview(offlinePracticePreview(snapshot, filters));
        applySession(store.session);
        const events = await pendingOfflineEvents();
        setPendingOfflineCount(events.length);
        setSyncableOfflineCount(syncableOfflineEvents(events, store).length);
        return;
      }
      const response = await fetch("/api/cram/session/grade", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ item_id: card.item_id, grade: value, session_id: session.id, shown_at: shownAt, answered_at: answeredAt, elapsed_ms: elapsedMs })
      });
      if (!response.ok) throw new Error(await response.text());
      const body = (await response.json()) as { session: ReviewSessionState };
      applySession(body.session);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }, [applySession, card, filters, offlineDeck, offlineMode, offlineStore, revealed, session, shownAt]);

  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      if (view !== "review") return;
      if (event.key === "Enter" || event.key === "0") {
        event.preventDefault();
        if (!revealed) void reveal();
      } else if (event.key.toLowerCase() === "r" || event.key === "5") replay();
      else if (event.key === "1") void grade("incorrect");
      else if (event.key === "2") void grade("correct");
    }
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [grade, replay, reveal, revealed, view]);

  if ((loading && !overview) || !sessionLoaded) return <Shell><State title="Loading" body="Preparing cards." /></Shell>;
  if (error && !overview) return <Shell><State title="Something broke" body={error} action={<button onClick={refreshOverview}>Retry</button>} /></Shell>;

  return (
    <Shell>
      {view === "batch" && overview && (
        <BatchPicker
          overview={overview}
          preview={preview}
          filters={filters}
          selected={selected}
          openSources={openSources}
          openCategories={openCategories}
          cap={cap}
          batchSize={batchSize}
          selectedCount={selectedCards.length}
          error={error}
          onFilters={setFilters}
          onToggleSourceOpen={(source) => setOpenSources((current) => toggleSet(current, source))}
          onToggleCategoryOpen={(key) => setOpenCategories((current) => toggleSet(current, key))}
          onSelectCards={(ids) => { setSelectionTouched(true); setSelected((current) => addSet(current, ids)); }}
          onClearCards={(ids) => { setSelectionTouched(true); setSelected((current) => removeSet(current, ids)); }}
          onClearAll={() => { setSelectionTouched(true); setSelected(new Set()); }}
          onCap={setCap}
          onStart={startBatch}
          activeReview={hasActiveReview}
          onResume={resumeReview}
          offlineBusy={offlineBusy}
          offlineSyncing={offlineSyncing}
          offlineMode={offlineMode}
          offlineSaveState={offlineSaveState}
          offlineError={offlineError}
          pendingOfflineCount={pendingOfflineCount}
          syncableOfflineCount={syncableOfflineCount}
          onPrepareOffline={prepareOffline}
          onSyncOffline={() => syncOfflineAnswers()}
        />
      )}
      {view === "review" && card && (
        <ReviewCard
          card={card}
          revealed={revealed}
          audioPath={audioPath}
          error={error}
          queueCount={queue.length + retryQueue.length}
          reviewedCards={reviewedCards}
          willRepeat={repeatCards.has(card.item_id)}
          onBack={openBatch}
          onToggleRepeat={toggleRepeat}
          onUndoLast={undoLastGrade}
          onOpenHistory={() => setHistoryOpen(true)}
          onReplay={replay}
          onReveal={reveal}
          onGrade={grade}
          audioRef={audioRef}
        />
      )}
      {historyOpen && <SessionHistoryDialog reviewedCards={reviewedCards} onClose={() => setHistoryOpen(false)} />}
      {view === "done" && <State title="Review complete" body="Saved your answers." action={<button onClick={() => { void refreshOverview(); openBatch(); }}>Pick more cards</button>} />}
    </Shell>
  );
}

createRoot(document.getElementById("root")!).render(<StrictMode><App /></StrictMode>);
