import type { PracticeCard, PracticeCategory, PracticeSource, ReviewAnswer, ScoreBuckets } from "./types";

export function categoryKey(category: Pick<PracticeCategory, "source" | "category">) {
  return `${category.source}\u0000${category.category}`;
}

export function toggleSet(current: Set<string>, key: string) {
  const next = new Set(current);
  if (next.has(key)) next.delete(key);
  else next.add(key);
  return next;
}

export function addSet(current: Set<string>, ids: string[]) {
  const next = new Set(current);
  for (const id of ids) next.add(id);
  return next;
}

export function removeSet(current: Set<string>, ids: string[]) {
  const next = new Set(current);
  for (const id of ids) next.delete(id);
  return next;
}

export function groupCardsByCategory(cards: PracticeCard[]) {
  const groups = new Map<string, PracticeCard[]>();
  for (const card of cards) {
    const key = categoryKey(card);
    groups.set(key, [...(groups.get(key) ?? []), card]);
  }
  return groups;
}

export function scoreText(card: PracticeCard) {
  return card.score === null ? "no Pleco score" : `score ${card.score}`;
}

export function reviewText(card: PracticeCard) {
  const date = card.last_reviewed_at ? new Date(card.last_reviewed_at).toLocaleDateString(undefined, { month: "short", day: "numeric" }) : "never reviewed";
  return card.reviewed_count > 0 ? `reviewed ${card.reviewed_count} · last ${date}` : date;
}

export function learnedText(row: Pick<PracticeSource, "total_count" | "not_learned_count">) {
  const learned = Math.max(0, row.total_count - row.not_learned_count);
  return `${learned}/${row.total_count} learned`;
}

export function scoreSegments(buckets: ScoreBuckets) {
  return [
    { key: "no-score", label: "no score", count: buckets.no_score, className: "scoreNone" },
    { key: "under-100", label: "<100", count: buckets.score_under_100, className: "scoreVeryLow" },
    { key: "100-199", label: "100-199", count: buckets.score_100_to_199, className: "scoreLow" },
    { key: "200-599", label: "200-599", count: buckets.score_200_to_599, className: "scoreLearning" },
    { key: "600-1599", label: "600-1599", count: buckets.score_600_to_1599, className: "scoreSolid" },
    { key: "1600-6399", label: "1600-6399", count: buckets.score_1600_to_6399, className: "scoreStrong" },
    { key: "6400+", label: "6400+", count: buckets.score_6400_plus, className: "scoreVeryStrong" }
  ].filter((segment) => segment.count > 0);
}

export function hashOffset(value: string, modulo: number) {
  let hash = 0;
  for (const char of value) hash = (hash * 31 + char.charCodeAt(0)) >>> 0;
  return hash % modulo;
}

export function cleanCategoryName(value: string) {
  return value.replace(/^Travel Survival\//, "");
}

export function sourceLabel(source: string) {
  if (source === "hellochinese") return "HelloChinese";
  if (source === "travel_survival") return "Travel Survival";
  return source;
}

export function gradeLabel(grade: ReviewAnswer) {
  return grade === "incorrect" ? "Incorrect" : "Correct";
}
