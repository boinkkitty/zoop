// Kept apart from the editor so the router can load a Page without pulling in TipTap.
import { queryOptions } from "@tanstack/react-query";
import { getPage } from "../api";

export const pageQuery = (day: string, notebook: string) =>
  queryOptions({
    queryKey: ["page", day, notebook],
    queryFn: () => getPage(day, notebook),
    // The open editor owns the Page's content; saves write their result back here.
    staleTime: Number.POSITIVE_INFINITY,
  });
