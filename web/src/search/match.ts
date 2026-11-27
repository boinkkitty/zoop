// The word-start rule search matches by, kept in step with internal/search: a query is
// its runs of letters and digits, lowercased, and a word in the text matches when it
// starts with any of them. The client marks matches itself with it, on the opened Page.

const WORD = /[\p{L}\p{N}]+/gu;

// queryWords is what the server searches for when the user types q.
export const queryWords = (q: string): string[] =>
  q.toLowerCase().match(WORD) ?? [];

export type Range = { from: number; to: number };

// matches finds each whole word of text that starts with one of words, in order.
export function matches(text: string, words: readonly string[]): Range[] {
  if (words.length === 0) return [];
  const out: Range[] = [];
  for (const m of text.matchAll(WORD)) {
    const w = m[0].toLowerCase();
    if (words.some((q) => w.startsWith(q)))
      out.push({ from: m.index, to: m.index + m[0].length });
  }
  return out;
}
