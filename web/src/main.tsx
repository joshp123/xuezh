import { StrictMode, useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { ReactNode } from "react";
import { createRoot } from "react-dom/client";
import "./styles.css";

type Card = {
  item_id: string;
  learning_order: number;
  word: string;
  pinyin: string;
  meaning: string;
  sentence_hanzi: string;
  sentence_pinyin: string;
  sentence_meaning: string;
  sentence_audio_paths: Record<string, string>;
  status: string;
};

const grades = [
  ["1", "again", "Again"],
  ["2", "hard", "Hard"],
  ["3", "good", "Good"],
  ["4", "easy", "Easy"]
] as const;

function App() {
  const [card, setCard] = useState<Card | null>(null);
  const [revealed, setRevealed] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [audioPath, setAudioPath] = useState<string | null>(null);
  const [audioIndex, setAudioIndex] = useState(0);
  const [lastGrade, setLastGrade] = useState<string | null>(null);
  const audioRef = useRef<HTMLAudioElement | null>(null);

  const loadNext = useCallback(async () => {
    setLoading(true);
    setError(null);
    setLastGrade(null);
    try {
      const response = await fetch("/api/cram/next");
      if (!response.ok) throw new Error(await response.text());
      const body = await response.json();
      setCard(body.card);
      setRevealed(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadNext();
  }, [loadNext]);

  const audioChoices = useMemo(() => {
    if (!card) return null;
    const paths = Object.entries(card.sentence_audio_paths ?? {})
      .sort(([left], [right]) => left.localeCompare(right))
      .map(([, path]) => path);
    if (paths.length === 0) return null;
    const offset = Math.floor(Math.random() * paths.length);
    return [...paths.slice(offset), ...paths.slice(0, offset)];
  }, [card?.item_id]);

  useEffect(() => {
    setAudioIndex(0);
  }, [audioChoices]);

  useEffect(() => {
    if (!audioChoices) {
      setAudioPath(null);
      return;
    }
    setAudioPath(`/${audioChoices[audioIndex % audioChoices.length]}`);
  }, [audioChoices, audioIndex]);

  const playCurrentAudio = useCallback(() => {
    if (!audioRef.current) return;
    audioRef.current.currentTime = 0;
    void audioRef.current.play();
  }, []);

  const replay = useCallback(() => {
    if (!audioChoices || audioChoices.length <= 1) {
      playCurrentAudio();
      return;
    }
    setAudioIndex((index) => (index + 1) % audioChoices.length);
  }, [audioChoices, playCurrentAudio]);

  useEffect(() => {
    if (audioPath) playCurrentAudio();
  }, [audioPath, playCurrentAudio]);

  const grade = useCallback(
    async (value: string) => {
      if (!card || !revealed) return;
      setError(null);
      try {
        const response = await fetch("/api/cram/grade", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ item_id: card.item_id, grade: value })
        });
        if (!response.ok) throw new Error(await response.text());
        setLastGrade(value);
        await loadNext();
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      }
    },
    [card, revealed, loadNext]
  );

  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === " " || event.key === "0" || event.key === "Enter") {
        event.preventDefault();
        if (!revealed) setRevealed(true);
      } else if (event.key.toLowerCase() === "r" || event.key === "5") {
        replay();
      } else {
        const match = grades.find(([key]) => key === event.key);
        if (match) void grade(match[1]);
      }
    }
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [grade, replay, revealed]);

  return (
    <main className="app">
      {loading ? (
        <State title="Loading next card" body="Preparing the review queue." />
      ) : error ? (
        <State title="Something broke" body={error} action={<button onClick={loadNext}>Retry</button>} />
      ) : !card ? (
        <State title="Nothing due" body="Import the corpus or come back when the next item is due." action={<button onClick={loadNext}>Check again</button>} />
      ) : (
        <section className="review" aria-live="polite">
          <div className="meta">
            <span>Card {card.learning_order}</span>
          </div>
          <div className="studyStack">
            <section className="target">
              <div className="targetWord">{card.word}</div>
              <div className="sentence">{highlightTarget(card.sentence_hanzi, card.word)}</div>
            </section>
            <div className="audioRow">
              <button onClick={replay} disabled={!audioPath}>
                <kbd>5</kbd> Replay
              </button>
              {!audioPath && <span>No audio</span>}
            </div>

            {audioPath && <audio ref={audioRef} src={audioPath} preload="auto" />}

            {revealed ? (
              <section className="answer shown">
                <div className="meaning">{card.meaning}</div>
                <div className="pinyin">{card.pinyin}</div>
                <div className="sentenceMeaning">{card.sentence_meaning}</div>
              </section>
            ) : (
              <section className="answer prompt" aria-hidden="true" />
            )}
          </div>

          <div className={revealed ? "controls controlsGrades" : "controls controlsReveal"}>
            {!revealed ? (
              <button className="primary revealButton" onClick={() => setRevealed(true)}>
                <kbd>0</kbd> Reveal
              </button>
            ) : (
              grades.map(([key, value, label]) => (
                <button key={value} onClick={() => void grade(value)}>
                  <kbd>{key}</kbd> {label}
                </button>
              ))
            )}
          </div>
          {lastGrade && <div className="saved">Saved {lastGrade}</div>}
        </section>
      )}
    </main>
  );
}

function highlightTarget(sentence: string, word: string) {
  const index = sentence.indexOf(word);
  if (index < 0 || sentence.indexOf(word, index + word.length) >= 0) return sentence;
  return (
    <>
      {sentence.slice(0, index)}
      <mark>{word}</mark>
      {sentence.slice(index + word.length)}
    </>
  );
}

function State(props: { title: string; body: string; action?: ReactNode }) {
  return (
    <section className="state">
      <h2>{props.title}</h2>
      <p>{props.body}</p>
      {props.action}
    </section>
  );
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>
);
