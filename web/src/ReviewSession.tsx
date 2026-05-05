import { useEffect, useState, type RefObject } from "react";
import type { Card, ReviewAnswer, ReviewedCard } from "./types";
import { cleanCategoryName, gradeLabel, sourceLabel } from "./utils";

export function ReviewCard(props: {
  card: Card;
  revealed: boolean;
  audioPath: string | null;
  error: string | null;
  queueCount: number;
  reviewedCards: ReviewedCard[];
  willRepeat: boolean;
  onBack: () => void;
  onToggleRepeat: () => void;
  onUndoLast: () => void;
  onOpenHistory: () => void;
  onReplay: () => void;
  onReveal: () => void;
  onGrade: (value: ReviewAnswer) => void;
  audioRef: RefObject<HTMLAudioElement | null>;
}) {
  const [showSentencePinyin, setShowSentencePinyin] = useState(false);
  useEffect(() => setShowSentencePinyin(false), [props.card.item_id]);
  const fullMeta = `${sourceLabel(props.card.source)} · ${cleanCategoryName(props.card.category)} · Card ${props.card.learning_order}`;
  const compactMeta = `${cleanCategoryName(props.card.category)} · ${props.card.learning_order}`;

  return (
    <section className="review" aria-live="polite">
      <div className="reviewTop">
        <button onClick={props.onBack}>Cards</button>
        <span title={fullMeta}>
          <span className="desktopMeta">{fullMeta}</span>
          <span className="mobileMeta">{compactMeta}</span>
        </span>
        <button className="audioButton" aria-label="Listen" title={props.audioPath ? "Listen" : "No audio"} onClick={props.onReplay} disabled={!props.audioPath}><kbd>5</kbd><span aria-hidden="true" className="playMark">▶</span><span aria-hidden="true" className="speechMark">🗣</span><span className="wideLabel">Listen</span></button>
        <button className={props.willRepeat ? "repeatButton active" : "repeatButton"} aria-pressed={props.willRepeat} onClick={props.onToggleRepeat}>
          <span className="wideLabel">{props.willRepeat ? "Will practice again" : "Practice again"}</span>
          <span className="narrowLabel">{props.willRepeat ? "Again ✓" : "Again"}</span>
        </button>
      </div>
      <div className="studyStack">
        <section className="target">
          <div className={props.revealed && props.card.pinyin ? "targetWordPinyin visible" : "targetWordPinyin"}>{props.card.pinyin || "\u00a0"}</div>
          <div className="targetWord">{props.card.word}</div>
          <div className="sentenceWrap">
            <div className="sentencePinyinSpace" aria-hidden="true" />
            <div className="sentenceFrame">
              <div className="sentence">{sentenceWithHighlight(props.card.sentence_hanzi, props.card.word)}</div>
              {showSentencePinyin && props.card.sentence_pinyin && (
                <div className="sentence sentenceRubyOverlay" aria-hidden="true">
                  {sentencePinyinOverlay(props.card.sentence_hanzi, props.card.word, props.card.sentence_pinyin)}
                </div>
              )}
            </div>
          </div>
        </section>
        {props.audioPath && <audio ref={props.audioRef} src={props.audioPath} preload="auto" />}
        {props.revealed ? (
          <section className="answer shown">
            <div className="meaning">{props.card.meaning}</div>
            <div className="sentenceMeaning">{props.card.sentence_meaning}</div>
            {props.card.sentence_pinyin && <button className="sentencePinyinToggle" type="button" onClick={() => setShowSentencePinyin((visible) => !visible)}>{showSentencePinyin ? "Hide sentence pinyin" : "Show sentence pinyin"}</button>}
          </section>
        ) : <section className="answer prompt" aria-hidden="true" />}
      </div>
      {!props.revealed ? <div className="controls controlsReveal"><button className="primary revealButton" onClick={props.onReveal}><kbd>Enter</kbd> Show answer</button></div> : <div className="controls controlsTwo"><button onClick={() => props.onGrade("incorrect")}><kbd>1</kbd> Incorrect</button><button className="primary" onClick={() => props.onGrade("correct")}><kbd>2</kbd> Correct</button></div>}
      <SessionPanel reviewedCards={props.reviewedCards} queueCount={props.queueCount} onUndoLast={props.onUndoLast} onOpenHistory={props.onOpenHistory} />
      {props.error && <div className="saved errorText">{props.error}</div>}
    </section>
  );
}

export function SessionHistoryDialog(props: { reviewedCards: ReviewedCard[]; onClose: () => void }) {
  return (
    <div className="historyOverlay" role="dialog" aria-modal="true" aria-labelledby="history-title">
      <section className="historyDialog">
        <header>
          <h2 id="history-title">Session history</h2>
          <button onClick={props.onClose}>Close</button>
        </header>
        {props.reviewedCards.length === 0 ? (
          <p>No cards reviewed yet.</p>
        ) : (
          <div className="historyList">
            {props.reviewedCards.map((item, index) => (
              <article className="historyItem" key={`${item.card.item_id}-${index}`}>
                <div>
                  <strong>{item.card.word}</strong>
                  <span>{item.card.sentence_hanzi}</span>
                  <em>{item.card.sentence_meaning || item.card.meaning}</em>
                </div>
                <small>{sourceLabel(item.card.source)} · {cleanCategoryName(item.card.category)}</small>
                <div className="historyBadges">
                  <span>{gradeLabel(item.grade)}</span>
                  {item.repeat && <span>practice again</span>}
                </div>
              </article>
            ))}
          </div>
        )}
      </section>
    </div>
  );
}

function SessionPanel(props: { reviewedCards: ReviewedCard[]; queueCount: number; onUndoLast: () => void; onOpenHistory: () => void }) {
  const last = props.reviewedCards[0];
  const correctCount = props.reviewedCards.filter((item) => item.grade === "correct").length;
  const incorrectCount = props.reviewedCards.filter((item) => item.grade === "incorrect").length;

  return (
    <aside className="sessionPanel">
      <div className="sessionCounts"><strong>{props.queueCount}</strong> left · ✓ <strong>{correctCount}</strong> · × <strong>{incorrectCount}</strong></div>
      {last && (
        <>
          <span className="lastReviewPill" title={`Last: ${last.card.word} ${gradeLabel(last.grade)}`}><em>Last</em><strong>{last.card.word}</strong><em>{last.grade === "correct" ? "✓" : "×"}</em>{last.repeat && <em className="attention">again</em>}</span>
          <button className="sessionAction" onClick={props.onUndoLast}>Undo</button>
          <button className="sessionAction" onClick={props.onOpenHistory}>History</button>
        </>
      )}
    </aside>
  );
}

function sentenceWithHighlight(sentence: string, word: string) {
  const targetStart = sentence.indexOf(word);
  const targetEnd = targetStart >= 0 && sentence.indexOf(word, targetStart + word.length) < 0 ? targetStart + word.length : -1;
  if (targetStart < 0 || targetEnd < 0) return sentence;
  const after = sentence.slice(targetEnd);
  const punctuation = after.match(/^[^\u3400-\u9fff]+/u)?.[0] ?? "";
  const remainingAfter = punctuation ? after.slice(punctuation.length) : after;
  return (
    <>
      {sentence.slice(0, targetStart)}
      <span className="noBreak"><mark>{sentence.slice(targetStart, targetEnd)}</mark>{punctuation}</span>
      {remainingAfter}
    </>
  );
}

function sentencePinyinOverlay(sentence: string, word: string, sentencePinyin: string) {
  const syllables = sentencePinyin
    .trim()
    .split(/\s+/)
    .flatMap(splitPinyinToken)
    .filter(Boolean);
  const chars = Array.from(sentence);
  const pinyinByChar = pinyinForSentenceChars(chars, syllables);
  const wordStart = sentence.indexOf(word);
  const wordIsUnique = wordStart >= 0 && sentence.indexOf(word, wordStart + word.length) < 0;
  const targetStart = wordIsUnique ? Array.from(sentence.slice(0, wordStart)).length : -1;
  const targetEnd = targetStart >= 0 ? targetStart + Array.from(word).length : -1;
  const nodes = [];

  for (let index = 0; index < chars.length; index++) {
    if (index === targetStart) {
      const targetNodes = [];
      for (; index < targetEnd; index++) {
        targetNodes.push(rubyChar(chars[index], pinyinByChar[index], index));
      }
      for (; index < chars.length && !isChineseChar(chars[index]); index++) {
        targetNodes.push(<span key={`${chars[index]}-${index}`}>{chars[index]}</span>);
      }
      nodes.push(<span className="noBreak" key="target-pinyin-run">{targetNodes}</span>);
      index--;
      continue;
    }
    nodes.push(rubyChar(chars[index], pinyinByChar[index], index));
  }

  return nodes;
}

function isChineseChar(value: string) {
  return /[\u3400-\u9fff]/u.test(value);
}

function pinyinForSentenceChars(chars: string[], syllables: string[]) {
  let syllableIndex = 0;
  return chars.map((char) => {
    if (!isChineseChar(char)) return "";
    return syllables[syllableIndex++] ?? "";
  });
}

function rubyChar(char: string, pinyin: string, index: number) {
  if (!isChineseChar(char) || !pinyin) return <span key={`${char}-${index}`}>{char}</span>;
  return (
    <ruby key={`${char}-${index}`}>
      {char}
      <rt>{pinyin}</rt>
    </ruby>
  );
}

const pinyinInitials = ["zh", "ch", "sh", "b", "p", "m", "f", "d", "t", "n", "l", "g", "k", "h", "j", "q", "x", "r", "z", "c", "s", "y", "w", ""];
const pinyinFinals = ["a", "o", "e", "ai", "ei", "ao", "ou", "an", "en", "ang", "eng", "ong", "i", "ia", "ie", "iao", "iu", "ian", "in", "iang", "ing", "iong", "u", "ua", "uo", "uai", "ui", "uan", "un", "uang", "ueng", "ue", "v", "ve", "van", "vn", "er"];
const pinyinSyllables = new Set(pinyinInitials.flatMap((initial) => pinyinFinals.map((final) => `${initial}${final}`)));

function splitPinyinToken(value: string) {
  const cleaned = value.replace(/[.,!?;:，。！？；：]+$/u, "");
  return cleaned.split(/['’·-]+/u).flatMap(splitPinyinPart);
}

function splitPinyinPart(value: string) {
  const chars = Array.from(value);
  const plain = chars.map(normalizePinyinChar).join("").toLowerCase();
  const syllables: string[] = [];
  let offset = 0;

  while (offset < chars.length) {
    let matchLength = 0;
    for (let length = Math.min(6, chars.length - offset); length > 0; length--) {
      if (pinyinSyllables.has(plain.slice(offset, offset + length))) {
        matchLength = length;
        break;
      }
    }
    if (matchLength === 0) {
      syllables.push(chars.slice(offset).join(""));
      break;
    }
    syllables.push(chars.slice(offset, offset + matchLength).join(""));
    offset += matchLength;
  }

  return syllables;
}

function normalizePinyinChar(value: string) {
  if ("ǖǘǚǜüǕǗǙǛÜ".includes(value)) return "v";
  return value.normalize("NFD").replace(/\p{M}/gu, "").toLowerCase();
}
