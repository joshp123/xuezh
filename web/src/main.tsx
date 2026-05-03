import { StrictMode, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createRoot } from "react-dom/client";
import { BatchPicker } from "./BatchPicker";
import { DesignSystemPage } from "./DesignSystemPage";
import { ReviewCard, SessionHistoryDialog } from "./ReviewSession";
import { Shell, State } from "./shared";
import type { Card, Filters, Overview, PracticePreview, ReviewAnswer, ReviewSessionState, View } from "./types";
import { defaultFilters } from "./types";
import { addSet, hashOffset, removeSet, toggleSet } from "./utils";
import "./styles.css";

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

  if (hash.startsWith("#design-system")) return <DesignSystemPage />;

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

  const refreshOverview = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch("/api/cram/overview");
      if (!response.ok) throw new Error(await response.text());
      setOverview((await response.json()) as Overview);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, []);

  const refreshPreview = useCallback(async () => {
    setError(null);
    try {
      const response = await fetch("/api/cram/preview", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(filters)
      });
      if (!response.ok) throw new Error(await response.text());
      setPreview((await response.json()) as PracticePreview);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }, [filters]);

  const loadActiveSession = useCallback(async () => {
    setError(null);
    try {
      const response = await fetch("/api/cram/session");
      if (!response.ok) throw new Error(await response.text());
      const body = (await response.json()) as { session: ReviewSessionState | null };
      if (body.session?.status === "active" && body.session.card) {
        setSession(body.session);
        setView("review");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSessionLoaded(true);
    }
  }, []);

  useEffect(() => { void refreshOverview(); }, [refreshOverview]);
  useEffect(() => { void loadActiveSession(); }, [loadActiveSession]);
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
    void audioRef.current.play();
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
      void refreshOverview();
    }
  }, [refreshOverview]);

  const openBatch = () => setView("batch");
  const resumeReview = () => { if (hasActiveReview) setView("review"); };

  const startBatch = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
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
  }, [applySession, batchLimit, selectedCards]);

  const reveal = useCallback(async () => {
    if (!session || revealed) return;
    setError(null);
    try {
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
  }, [revealed, session]);

  const toggleRepeat = useCallback(async () => {
    if (!session) return;
    setError(null);
    try {
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
  }, [session]);

  const undoLastGrade = useCallback(async () => {
    if (!session || reviewedCards.length === 0) return;
    setError(null);
    try {
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
  }, [applySession, reviewedCards.length, session]);

  const grade = useCallback(async (value: ReviewAnswer) => {
    if (!session || !card || !revealed) return;
    const answeredAt = new Date().toISOString();
    const elapsedMs = shownAt ? Date.parse(answeredAt) - Date.parse(shownAt) : 0;
    setError(null);
    try {
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
  }, [applySession, card, revealed, session, shownAt]);

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
