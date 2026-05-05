import { useEffect, useRef, useState, type RefObject } from "react";
import { ReviewCard } from "../ReviewSession";
import type { Card, ReviewAnswer, ReviewedCard } from "../types";
import "./design-system.css";
import "./design-system-review.css";

const scores = [
  { className: "dsScoreNone", width: 10 },
  { className: "dsScoreLow", width: 12 },
  { className: "dsScoreLearned", width: 58 },
  { className: "dsScoreStrong", width: 20 }
];

const fixtureAudio = "data:audio/ogg;base64,";

const normalCard = card({
  item_id: "ds-normal",
  category: "Habits",
  learning_order: 661,
  word: "擦",
  pinyin: "cā",
  meaning: "to wipe; to rub",
  sentence_hanzi: "我把窗户擦了。",
  sentence_pinyin: "wǒ bǎ chuānghu cā le",
  sentence_meaning: "I cleaned the window."
});

const longSentenceCard = card({
  item_id: "ds-long",
  category: "Going Abroad",
  learning_order: 604,
  word: "就",
  pinyin: "jiù",
  meaning: "as soon as; right after",
  sentence_hanzi: "我到了中国就跟你联系。",
  sentence_pinyin: "wǒ dào le zhōng guó jiù gēn nǐ lián xì",
  sentence_meaning: "I will contact you as soon as I arrive in China."
});

const longAnswerCard = card({
  item_id: "ds-answer",
  category: "Dating",
  learning_order: 517,
  word: "愿意",
  pinyin: "yuànyì",
  meaning: "to be willing (to do something); to want",
  sentence_hanzi: "她愿意当我的女朋友吗？",
  sentence_pinyin: "tā yuànyì dāng wǒ de nǚ péngyou ma",
  sentence_meaning: "Is she willing to be my girlfriend?"
});

const reviewed = [
  reviewedCard(card({
    item_id: "ds-last",
    category: "Money",
    learning_order: 48,
    word: "放",
    pinyin: "fàng",
    meaning: "to put",
    sentence_hanzi: "把钱放这里。",
    sentence_pinyin: "bǎ qián fàng zhèlǐ",
    sentence_meaning: "Put the money here."
  }), "incorrect"),
  reviewedCard(normalCard, "correct"),
  reviewedCard(longSentenceCard, "correct")
];

export function DesignSystemPage() {
  const hash = window.location.hash;
  const checks = useDesignSystemChecks(hash);
  const full = fullSpecimen(hash);
  if (full) {
    return (
      <main className={`designSystem designSystemSingle ds${titleCase(full.device)}Specimen`}>
        <ReviewSpecimen {...full} fullSize />
      </main>
    );
  }

  return (
    <main className="designSystem">
      <header className="dsIntro">
        <a href="/" className="dsTextButton">Back to app</a>
        <div>
          <p>XUEZH UI</p>
          <h1>Design system</h1>
          <span>Minimal components for card picking and sentence review.</span>
        </div>
      </header>

      <section className="dsSection dsChecks">
        <div>
          <h2>Regression checks</h2>
          <p className="dsSectionNote">These run in-browser against the production review components below. They catch clipped controls, horizontal overflow, and prompt movement between reveal states.</p>
        </div>
        <div className="dsCheckList">
          {checks.map((check) => (
            <span className={check.pass ? "dsCheckResult pass" : "dsCheckResult fail"} key={check.name}>
              <strong>{check.pass ? "Pass" : "Fail"}</strong>
              {check.name}
              {check.detail && <small>{check.detail}</small>}
            </span>
          ))}
        </div>
      </section>

      <section className="dsSection">
        <h2>Type roles</h2>
        <p className="dsSectionNote">Keep the product to three weights: regular for reading, medium for controls and numbers, bold for screen titles and Chinese prompts.</p>
        <div className="dsTypeGrid">
          <div><small>Page title</small><strong className="dsPageTitle">What to review</strong></div>
          <div><small>Chinese prompt</small><strong className="dsChineseType">擦</strong></div>
          <div><small>Body / action</small><span>Food Menu Core</span></div>
          <div><small>Meta</small><em>43/53 learned</em></div>
          <div><small>Pinyin</small><span className="dsPinyinType">cā</span></div>
          <div><small>Session facts</small><span>✓ 7 · × 2</span></div>
        </div>
      </section>

      <section className="dsSection">
        <h2>Controls</h2>
        <div className="dsControlGrid">
          <button className="dsButton dsPrimary">Start review</button>
          <button className="dsButton">Cards</button>
          <button className="dsTextButton">Show sentence pinyin</button>
          <label className="dsToggle"><input type="checkbox" defaultChecked /> Not learned</label>
          <label className="dsToggle"><input type="checkbox" defaultChecked /> Due for review</label>
          <label className="dsToggle"><input type="checkbox" /> Got wrong before</label>
        </div>
      </section>

      <section className="dsSection">
        <h2>Category progress</h2>
        <div className="dsStickyHeaderExample">
          <div>
            <strong>HelloChinese</strong>
            <span>655/689 learned</span>
            <LearningBar />
          </div>
          <div>
            <button className="dsButton">Select all</button>
            <button className="dsButton">Clear</button>
          </div>
        </div>
        <div className="dsRows">
          <CategoryRow name="Daily Life" learned="7/11 learned" selected />
          <CategoryRow name="Habits" learned="12/21 learned" selected />
          <CategoryRow name="Must Know - Restaurant Vendor Flow" learned="18/52 learned" selected />
        </div>
      </section>

      <section className="dsSection">
        <h2>Flashcard states</h2>
        <p className="dsSectionNote">These are the production review card in desktop and phone frames. If either frame differs from the app, the design system is wrong.</p>
        <h3 className="dsSubhead">Desktop</h3>
        <div className="dsFlashGrid dsDesktopGrid">
          <ReviewSpecimen label="Before reveal" card={normalCard} revealed={false} device="desktop" href="#design-system-before" />
          <ReviewSpecimen label="After reveal" card={normalCard} revealed device="desktop" href="#design-system-after" />
          <ReviewSpecimen label="Long answer" card={longAnswerCard} revealed device="desktop" href="#design-system-long-answer" />
        </div>
        <h3 className="dsSubhead">Phone, 393 x 852</h3>
        <div className="dsFlashGrid dsPhoneGrid">
          <ReviewSpecimen label="Before reveal" card={normalCard} revealed={false} device="phone" href="#design-system-phone-before" />
          <ReviewSpecimen label="After reveal" card={normalCard} revealed device="phone" href="#design-system-phone-after" />
          <ReviewSpecimen label="Long sentence" card={longSentenceCard} revealed device="phone" href="#design-system-phone-long" />
          <ReviewSpecimen label="Long answer" card={longAnswerCard} revealed device="phone" href="#design-system-phone-long-answer" />
        </div>
      </section>

      <section className="dsSection">
        <h2>History sheet</h2>
        <div className="dsHistory">
          <header><strong>History</strong><button className="dsButton">Close</button></header>
          <p>This session · ✓ 7 · × 2</p>
          <article><strong>擦</strong><span>我把窗户擦了。<small>I cleaned the window.</small></span><em>Correct</em></article>
          <article><strong>扔</strong><span>她把衣服扔在床上边了。<small>She threw the clothes on the bed.</small></span><em className="bad">Incorrect</em></article>
        </div>
      </section>
    </main>
  );
}

function fullSpecimen(hash: string) {
  if (hash === "#design-system-before") return { label: "Before reveal", card: normalCard, revealed: false, device: "desktop" as const };
  if (hash === "#design-system-after") return { label: "After reveal", card: normalCard, revealed: true, device: "desktop" as const };
  if (hash === "#design-system-long") return { label: "Long sentence", card: longSentenceCard, revealed: true, device: "desktop" as const };
  if (hash === "#design-system-long-answer") return { label: "Long answer", card: longAnswerCard, revealed: true, device: "desktop" as const };
  if (hash === "#design-system-phone-before") return { label: "Before reveal", card: normalCard, revealed: false, device: "phone" as const };
  if (hash === "#design-system-phone-after") return { label: "After reveal", card: normalCard, revealed: true, device: "phone" as const };
  if (hash === "#design-system-phone-long") return { label: "Long sentence", card: longSentenceCard, revealed: true, device: "phone" as const };
  if (hash === "#design-system-phone-long-answer") return { label: "Long answer", card: longAnswerCard, revealed: true, device: "phone" as const };
  return null;
}

type DesignCheck = { name: string; pass: boolean; detail?: string };

function useDesignSystemChecks(hash: string) {
  const [checks, setChecks] = useState<DesignCheck[]>([]);

  useEffect(() => {
    if (hash !== "" && hash !== "#design-system") return;
    let cancelled = false;

    function measure() {
      const frames = [...document.querySelectorAll<HTMLElement>(".dsReviewSpecimen")];
      const phoneFrames = [...document.querySelectorAll<HTMLElement>(".dsReviewSpecimen.dsPhone")];
      const overflowing = frames.filter((frame) => frame.scrollWidth > frame.clientWidth + 1 || frame.scrollHeight > frame.clientHeight + 1);
      const clippedControls = phoneFrames.flatMap((frame) => [...frame.querySelectorAll<HTMLElement>("button, .sessionPanel")])
        .filter((el) => el.scrollWidth > el.clientWidth + 1 || el.scrollHeight > el.clientHeight + 1);
      const before = document.querySelector<HTMLElement>('[data-ds-device="phone"][data-ds-state="before-reveal"] .sentence');
      const after = document.querySelector<HTMLElement>('[data-ds-device="phone"][data-ds-state="after-reveal"] .sentence');
      const promptDelta = before && after ? Math.abs(relativeTopInFrame(before) - relativeTopInFrame(after)) : 0;

      const next = [
        { name: "No review-frame overflow", pass: overflowing.length === 0, detail: overflowing.length ? `${overflowing.length} frame(s)` : undefined },
        { name: "Phone controls do not wrap or clip", pass: clippedControls.length === 0, detail: clippedControls.length ? `${clippedControls.length} control(s)` : undefined },
        { name: "Prompt stays stable after reveal", pass: promptDelta <= 2, detail: promptDelta > 2 ? `${Math.round(promptDelta)}px movement` : undefined }
      ];

      if (!cancelled) setChecks(next);
    }

    const raf = window.requestAnimationFrame(measure);
    window.addEventListener("resize", measure);
    return () => {
      cancelled = true;
      window.cancelAnimationFrame(raf);
      window.removeEventListener("resize", measure);
    };
  }, [hash]);

  return checks;
}

function relativeTopInFrame(element: HTMLElement) {
  const frame = element.closest<HTMLElement>(".dsReviewSpecimen");
  return element.getBoundingClientRect().top - (frame?.getBoundingClientRect().top ?? 0);
}

function ReviewSpecimen(props: { label: string; card: Card; revealed: boolean; device?: "desktop" | "phone"; href?: string; fullSize?: boolean }) {
  const audioRef = useRef<HTMLAudioElement | null>(null);
  const device = props.device ?? "phone";
  const state = props.label.toLowerCase().replaceAll(" ", "-");
  return (
    <article className={`${props.fullSize ? "dsFlashExample dsFlashExampleFull" : "dsFlashExample"} ds${titleCase(device)}`} data-ds-device={device} data-ds-state={state}>
      {!props.fullSize && (
        <div className="dsStateHeader">
          <div>
            <div className="dsStateLabel">{props.label}</div>
            <span>{device === "phone" ? "393 x 852 CSS px" : "desktop frame"}</span>
          </div>
          {props.href && <a className="dsTextButton" href={props.href}>Open full-size</a>}
        </div>
      )}
      <div className={`dsReviewSpecimen ds${titleCase(device)}`}>
        <ReviewCard
          card={props.card}
          revealed={props.revealed}
          audioPath={fixtureAudio}
          error={null}
          queueCount={8}
          reviewedCards={reviewed}
          willRepeat={false}
          onBack={() => undefined}
          onToggleRepeat={() => undefined}
          onUndoLast={() => undefined}
          onOpenHistory={() => undefined}
          onReplay={() => undefined}
          onReveal={() => undefined}
          onGrade={() => undefined}
          audioRef={audioRef as RefObject<HTMLAudioElement | null>}
          initialSentencePinyinVisible={props.revealed && props.card === longSentenceCard}
        />
      </div>
    </article>
  );
}

function titleCase(value: "desktop" | "phone") {
  return value.charAt(0).toUpperCase() + value.slice(1);
}

function LearningBar() {
  return (
    <span className="dsLearningBar" aria-label="Learning progress">
      {scores.map((score) => <i className={score.className} style={{ width: `${score.width}%` }} key={score.className} />)}
    </span>
  );
}

function CategoryRow(props: { name: string; learned: string; selected?: boolean }) {
  return (
    <div className="dsCategoryRow">
      <input type="checkbox" checked={props.selected ?? false} readOnly aria-label={`Select ${props.name}`} />
      <button className="dsDisclosure">▸</button>
      <strong>{props.name}</strong>
      <div>
        <span>{props.learned}</span>
        <LearningBar />
      </div>
    </div>
  );
}

function card(input: Omit<Card, "source" | "sentence_audio_paths">): Card {
  return { ...input, source: "hellochinese", sentence_audio_paths: {} };
}

function reviewedCard(cardValue: Card, grade: ReviewAnswer): ReviewedCard {
  return {
    card: cardValue,
    grade,
    repeat: false
  };
}
