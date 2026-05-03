import { useRef, type RefObject } from "react";
import { ReviewCard } from "./ReviewSession";
import type { Card, ReviewAnswer, ReviewedCard } from "./types";

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
  category: "Catching a Flight",
  learning_order: 613,
  word: "极了",
  pinyin: "jíle",
  meaning: "extremely",
  sentence_hanzi: "我觉得坐飞机方便极了。",
  sentence_pinyin: "wǒ juéde zuò fēijī fāngbiàn jíle",
  sentence_meaning: "I think taking a plane is extremely convenient."
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
  const full = fullSpecimen(hash);
  if (full) {
    return (
      <main className="designSystem designSystemSingle dsPhoneSpecimen">
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
          <label className="dsToggle"><input type="checkbox" defaultChecked /> Due today</label>
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
        <p className="dsSectionNote">These are real review cards in a fixed 393 x 852 phone frame. If this differs from the app, the design system is wrong.</p>
        <div className="dsFlashGrid">
          <ReviewSpecimen label="Before reveal" card={normalCard} revealed={false} href="#design-system-before" />
          <ReviewSpecimen label="After reveal" card={normalCard} revealed href="#design-system-after" />
          <ReviewSpecimen label="Long sentence" card={longSentenceCard} revealed href="#design-system-long" />
          <ReviewSpecimen label="Long answer" card={longAnswerCard} revealed href="#design-system-long-answer" />
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
  if (hash === "#design-system-before") return { label: "Before reveal", card: normalCard, revealed: false };
  if (hash === "#design-system-after") return { label: "After reveal", card: normalCard, revealed: true };
  if (hash === "#design-system-long") return { label: "Long sentence", card: longSentenceCard, revealed: true };
  if (hash === "#design-system-long-answer") return { label: "Long answer", card: longAnswerCard, revealed: true };
  return null;
}

function ReviewSpecimen(props: { label: string; card: Card; revealed: boolean; href?: string; fullSize?: boolean }) {
  const audioRef = useRef<HTMLAudioElement | null>(null);
  return (
    <article className={props.fullSize ? "dsFlashExample dsFlashExampleFull" : "dsFlashExample"}>
      {!props.fullSize && (
        <div className="dsStateHeader">
          <div>
            <div className="dsStateLabel">{props.label}</div>
            <span>393 x 852 CSS px</span>
          </div>
          {props.href && <a className="dsTextButton" href={props.href}>Open full-size</a>}
        </div>
      )}
      <div className="dsReviewSpecimen">
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
        />
      </div>
    </article>
  );
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
      <span className={props.selected ? "dsCheck checked" : "dsCheck"}>{props.selected ? "✓" : ""}</span>
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
