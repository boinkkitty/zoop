import { describe, expect, it } from "vitest";
import { matches, queryWords } from "./match";

// What a search marks in a piece of text, as the server's 'simple' word-start rule matches it.
const marked = (q: string, text: string) =>
  matches(text, queryWords(q)).map(({ from, to }) => text.slice(from, to));

describe("the word-start rule", () => {
  it("marks every whole word that starts with a query word, in any case", () => {
    expect(marked("rai", "Rain all morning, then rainbows")).toEqual([
      "Rain",
      "rainbows",
    ]);
    expect(marked("BREAD mu", "Baked bread with Mum")).toEqual([
      "bread",
      "Mum",
    ]);
  });

  it("never marks the middle of a word", () => {
    expect(marked("um", "Mum hummed")).toEqual([]);
    expect(marked("read", "Baked bread")).toEqual([]);
  });

  it("takes query punctuation and syntax as word breaks, as the server does", () => {
    expect(queryWords("yeast, bread!")).toEqual(["yeast", "bread"]);
    expect(queryWords("it's")).toEqual(["it", "s"]);
    expect(queryWords("&:*()<->")).toEqual([]);
    expect(marked("café", "It's a café: tea & cake!")).toEqual(["café"]);
    expect(marked("tea & cake", "It's a café: tea & cake!")).toEqual([
      "tea",
      "cake",
    ]);
  });

  it("marks date words like any other text", () => {
    expect(marked("nov 2026", "Thursday, 26 November 2026")).toEqual([
      "November",
      "2026",
    ]);
  });
});
