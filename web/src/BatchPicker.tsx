import { useMemo, useState } from "react";
import { State } from "./shared";
import type { Filters, OfflineSaveState, Overview, PracticeCard, PracticePreview, PracticeSource, ScoreBuckets } from "./types";
import { capOptions, defaultFilters } from "./types";
import { categoryKey, cleanCategoryName, groupCardsByCategory, learnedText, reviewText, scoreSegments, scoreText } from "./utils";

export function BatchPicker(props: {
  overview: Overview;
  preview: PracticePreview | null;
  filters: Filters;
  selected: Set<string>;
  openSources: Set<string>;
  openCategories: Set<string>;
  cap: number;
  batchSize: number;
  selectedCount: number;
  error: string | null;
  onFilters: (filters: Filters) => void;
  onToggleSourceOpen: (source: string) => void;
  onToggleCategoryOpen: (key: string) => void;
  onSelectCards: (ids: string[]) => void;
  onClearCards: (ids: string[]) => void;
  onClearAll: () => void;
  onCap: (value: number) => void;
  onStart: () => void;
  activeReview: boolean;
  onResume: () => void;
  offlineBusy: boolean;
  offlineSyncing: boolean;
  offlineMode: boolean;
  offlineSaveState: OfflineSaveState | null;
  offlineError: string | null;
  pendingOfflineCount: number;
  syncableOfflineCount: number;
  onPrepareOffline: () => void;
  onSyncOffline: () => void | Promise<void>;
}) {
  const [offlineOpen, setOfflineOpen] = useState(false);
  const preview = props.preview;
  const cardsByCategory = useMemo(() => groupCardsByCategory(preview?.cards ?? []), [preview]);
  const overviewSources = props.overview.sources ?? [];
  const previewSources = preview?.sources ?? [];
  const previewCategories = preview?.categories ?? [];
  const previewCards = preview?.cards ?? [];
  const sources = preview ? previewSources : overviewSources.map(emptyPracticeSource);

  return (
    <section className="batch">
      <header className="batchHeader">
        <div className="screenHeader split batchTitle">
          <h1>What to review</h1>
          <button className="offlineButton" type="button" onClick={() => setOfflineOpen(true)}>
            Offline
            {props.pendingOfflineCount > 0 && <span>{props.pendingOfflineCount}</span>}
          </button>
        </div>
        <PracticeFilters filters={props.filters} onChange={props.onFilters} />
      </header>
      <div className="categoryList">
        {!preview && <State title="Loading" body="Finding review cards." />}
        {preview && sources.map((source) => {
          const categories = previewCategories.filter((category) => category.source === source.source);
          const sourceCards = previewCards.filter((item) => item.source === source.source);
          const selectedInSource = sourceCards.filter((item) => props.selected.has(item.item_id)).length;
          const isOpen = props.openSources.has(source.source);
          return (
            <section className="categoryGroup" key={source.source}>
              <div className="categoryGroupHeader">
                <button className="sourceToggle" onClick={() => props.onToggleSourceOpen(source.source)}>
                  <span aria-hidden="true">{isOpen ? "▾" : "▸"}</span>
                  <span>
                    <strong>{source.label}</strong>
                    <em>{learnedText(source)}</em>
                    <ScoreStrip buckets={source.score_buckets} total={source.total_count} />
                  </span>
                </button>
                <div className="groupActions">
                  <button onClick={() => props.onSelectCards(sourceCards.map((item) => item.item_id))}>Select all</button>
                  <button onClick={() => props.onClearCards(sourceCards.map((item) => item.item_id))}>Clear</button>
                </div>
              </div>
              {isOpen && categories.map((category) => {
                const key = categoryKey(category);
                const cards = cardsByCategory.get(key) ?? [];
                const selectedInCategory = cards.filter((item) => props.selected.has(item.item_id)).length;
                const categoryOpen = props.openCategories.has(key);
                return (
                  <section className="categoryBlock" key={key}>
                    <label className="categoryRow">
                      <input
                        type="checkbox"
                        checked={cards.length > 0 && selectedInCategory === cards.length}
                        onChange={() => selectedInCategory === cards.length ? props.onClearCards(cards.map((item) => item.item_id)) : props.onSelectCards(cards.map((item) => item.item_id))}
                      />
                      <button type="button" className="categoryDisclosure" onClick={() => props.onToggleCategoryOpen(key)}>{categoryOpen ? "▾" : "▸"}</button>
                      <span className="categoryName">{cleanCategoryName(category.category)}</span>
                      <span className="categoryStats">
                        <strong>{learnedText(category)}</strong>
                        <ScoreStrip buckets={category.score_buckets} total={category.total_count} />
                      </span>
                    </label>
                    {categoryOpen && <div className="cardPreviewList">{cards.map((item) => <PracticeCardRow card={item} selected={props.selected.has(item.item_id)} key={item.item_id} />)}</div>}
                  </section>
                );
              })}
              {isOpen && sourceCards.length === 0 && <div className="emptyGroup">No cards match the current filters.</div>}
              {isOpen && selectedInSource > 0 && <div className="selectedHint">{selectedInSource} selected here</div>}
            </section>
          );
        })}
      </div>
      <footer className="batchFooter">
        <div className="selectionSummary">
          <span><strong>{props.selectedCount}</strong> selected</span>
          <span className="footerLinks">
            {props.activeReview && <button type="button" onClick={props.onResume}>Resume review</button>}
            <button type="button" onClick={props.onClearAll}>Clear all</button>
          </span>
        </div>
        <label>Cards this round <select value={props.cap} onChange={(event) => props.onCap(Number(event.target.value))}>{capOptions.map((option) => <option value={option} key={option}>{option === 0 ? "all" : `${option}`}</option>)}</select></label>
        <button className="primary" disabled={props.batchSize === 0} onClick={props.onStart}>Start review</button>
      </footer>
      {props.error && <div className="errorText">{props.error}</div>}
      {offlineOpen && (
        <OfflineSheet
          state={props.offlineSaveState}
          error={props.offlineError}
          pendingCount={props.pendingOfflineCount}
          activeReview={props.activeReview}
          offlineMode={props.offlineMode}
          busy={props.offlineBusy}
          syncing={props.offlineSyncing}
          syncableCount={props.syncableOfflineCount}
          onClose={() => setOfflineOpen(false)}
          onPrepare={props.onPrepareOffline}
          onSync={props.onSyncOffline}
        />
      )}
    </section>
  );
}

function OfflineSheet(props: {
  state: OfflineSaveState | null;
  error: string | null;
  pendingCount: number;
  syncableCount: number;
  activeReview: boolean;
  offlineMode: boolean;
  busy: boolean;
  syncing: boolean;
  onClose: () => void;
  onPrepare: () => void;
  onSync: () => void | Promise<void>;
}) {
  const state = props.state;
  const syncHint = props.activeReview
    ? "Older answers sync while you review; the latest answer stays local so Undo still works."
    : "Answers sync when this device is online.";
  return (
    <div className="offlineOverlay" role="presentation" onClick={props.onClose}>
      <section className="offlineSheet" role="dialog" aria-modal="true" aria-labelledby="offline-title" onClick={(event) => event.stopPropagation()}>
        <header className="offlineSheetHeader">
          <div>
            <h2 id="offline-title">Offline</h2>
            <p>{props.offlineMode ? "Using the saved deck on this device." : state ? "Full deck is saved on this device." : "Save the full deck and audio for travel."}</p>
          </div>
          <button type="button" onClick={props.onClose}>Close</button>
        </header>
        <dl className="offlineFacts">
          <div><dt>Cards</dt><dd>{state ? `${state.card_count} saved` : "Not saved yet"}</dd></div>
          <div><dt>Audio</dt><dd>{audioSavedText(state)}</dd></div>
          <div><dt>Storage</dt><dd>{storageText(state)}</dd></div>
          <div><dt>Answers</dt><dd>{props.pendingCount === 0 ? "Synced" : `${props.pendingCount} waiting`}</dd></div>
        </dl>
        {state && <p className="offlineHint">Last saved {relativeSavedAt(state.saved_at)}. {syncHint}</p>}
        {props.activeReview && props.pendingCount > 0 && props.syncableCount === 0 && <p className="offlineHint">Answer one more card, or finish this review, to sync the latest answer.</p>}
        {props.error && <p className="offlineError">Sync problem: {props.error}</p>}
        <div className="offlineActions">
          <button type="button" className="primary" onClick={props.onPrepare} disabled={props.busy}>
            {props.busy ? "Saving…" : state ? "Refresh offline copy" : "Save offline"}
          </button>
          <button type="button" onClick={() => { void props.onSync(); }} disabled={props.syncing || props.syncableCount === 0}>
            {props.syncing ? "Syncing…" : "Sync now"}
          </button>
        </div>
      </section>
    </div>
  );
}

function audioSavedText(state: OfflineSaveState | null) {
  if (!state) return "Not saved yet";
  if (state.audio_total === 0) return "No audio in deck";
  if (state.audio_missing === 0) return `${state.audio_saved}/${state.audio_total} saved`;
  return `${state.audio_saved}/${state.audio_total} saved, ${state.audio_missing} missing`;
}

function storageText(state: OfflineSaveState | null) {
  if (!state) return "Unknown";
  const storage = state.storage;
  const persistence = storage.persisted === true ? "Protected by browser" : storage.persisted === false ? "Stored by browser" : "Unknown";
  if (!storage.usage_bytes || !storage.quota_bytes) return persistence;
  return `${persistence} · ${formatBytes(storage.usage_bytes)} used`;
}

function relativeSavedAt(value: string) {
  const time = Date.parse(value);
  if (!Number.isFinite(time)) return value;
  const seconds = Math.max(0, Math.round((Date.now() - time) / 1000));
  if (seconds < 60) return "just now";
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.round(minutes / 60);
  if (hours < 48) return `${hours}h ago`;
  return new Date(value).toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

function formatBytes(value: number) {
  if (value < 1024 * 1024) return `${Math.round(value / 1024)} KB`;
  return `${Math.round(value / (1024 * 1024))} MB`;
}

function ScoreStrip(props: { buckets: ScoreBuckets; total: number }) {
  const segments = scoreSegments(props.buckets);
  if (props.total <= 0 || segments.length === 0) return null;
  const label = segments.map((segment) => `${segment.label}: ${segment.count}`).join(", ");
  return (
    <span className="scoreStrip" aria-label={`Score distribution. ${label}`}>
      {segments.map((segment) => (
        <span
          className={segment.className}
          key={segment.key}
          style={{ flexGrow: segment.count }}
          title={`${segment.label}: ${segment.count}`}
        />
      ))}
    </span>
  );
}

function PracticeFilters(props: { filters: Filters; onChange: (filters: Filters) => void }) {
  const update = (patch: Partial<Filters>) => props.onChange({ ...props.filters, ...patch });
  return (
    <div className="practiceFilters">
      <label><input type="checkbox" checked={props.filters.include_not_learned} onChange={(event) => update({ include_not_learned: event.target.checked })} /> Not learned yet</label>
      <label><input type="checkbox" checked={props.filters.include_due} onChange={(event) => update({ include_due: event.target.checked })} /> Due for review</label>
      <label><input type="checkbox" checked={props.filters.include_got_wrong} onChange={(event) => update({ include_got_wrong: event.target.checked })} /> Got wrong before</label>
      <details className="advancedFilters">
        <summary>Advanced</summary>
        <label>Score below <input type="number" min="1" value={props.filters.score_below} onChange={(event) => update({ score_below: Number(event.target.value) || 200 })} /></label>
        <label>Wrong more than <input type="number" min="0" value={props.filters.misses_more_than} onChange={(event) => update({ misses_more_than: Number(event.target.value) || 0 })} /></label>
        <label><input type="checkbox" checked={props.filters.include_no_score} onChange={(event) => update({ include_no_score: event.target.checked })} /> Include never reviewed</label>
        <button type="button" onClick={() => props.onChange(defaultFilters)}>Reset</button>
      </details>
    </div>
  );
}

function PracticeCardRow(props: { card: PracticeCard; selected: boolean }) {
  return (
    <div className={props.selected ? "practiceCard selected" : "practiceCard"}>
      <div>
        <strong>{props.card.word}</strong>
        <span>{props.card.sentence_hanzi}</span>
      </div>
      <div className="reasonChips">
        {props.card.not_learned && <span>not learned</span>}
        {props.card.due && <span>due</span>}
        {props.card.got_wrong && <span>wrong {props.card.incorrect_count} {props.card.incorrect_count === 1 ? "time" : "times"}</span>}
      </div>
      <small>{scoreText(props.card)} · {reviewText(props.card)}</small>
    </div>
  );
}

function emptyPracticeSource(source: Overview["sources"][number]): PracticeSource {
  return {
    ...source,
    practice_count: 0,
    not_learned_count: 0,
    due_count: 0,
    got_wrong_count: 0,
    score_buckets: {
      no_score: 0,
      score_under_100: 0,
      score_100_to_199: 0,
      score_200_to_599: 0,
      score_600_to_1599: 0,
      score_1600_to_6399: 0,
      score_6400_plus: 0
    }
  };
}
