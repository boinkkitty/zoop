// The v1 Notebooks, in index order. Kept apart from the editor so the router can use them without TipTap.
export const NOTEBOOKS = [
  { slug: "my-journal", name: "My Journal" },
  { slug: "i-learned", name: "I Learned" },
] as const;

export type Notebook = (typeof NOTEBOOKS)[number]["slug"];

export const isNotebook = (slug: string): slug is Notebook =>
  NOTEBOOKS.some((n) => n.slug === slug);
