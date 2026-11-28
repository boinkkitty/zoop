import { Link } from "@tanstack/react-router";
import { NOTEBOOKS, type Notebook } from "../page/notebooks";
import styles from "./NotebookIndex.module.css";

// The index inset in the sheet's right edge: one vertical label per Notebook of this Day.
// Each is a link to its own URL, so back/forward moves between Notebooks.
export function NotebookIndex({
  day,
  active,
}: {
  day: string;
  active: Notebook;
}) {
  return (
    <nav className={styles.index} aria-label="Notebooks" data-coach="notebooks">
      {NOTEBOOKS.map(({ slug, name }) => (
        <Link
          key={slug}
          to="/day/$day/$notebook"
          params={{ day, notebook: slug }}
          className={styles.tab}
          aria-current={slug === active ? "page" : undefined}
        >
          {name}
        </Link>
      ))}
    </nav>
  );
}
