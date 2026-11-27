import { infiniteQueryOptions, queryOptions } from "@tanstack/react-query";
import { getDaysSummary, getMonth, type RailMonth } from "../api";

// Everything the rail shows; a save invalidates it all.
export const railKey = ["rail"] as const;

// The rail's Days, a month per page, starting at anchor: today's month, or the
// month the scrubber jumped to. Newer months load above it, older ones below.
export const railQuery = (anchor: string) =>
  infiniteQueryOptions({
    queryKey: [...railKey, "months", anchor],
    queryFn: ({ pageParam }) => getMonth(pageParam),
    initialPageParam: anchor,
    getNextPageParam: (last: RailMonth) => last.olderMonth ?? undefined,
    getPreviousPageParam: (first: RailMonth) => first.newerMonth ?? undefined,
  });

export const summaryQuery = queryOptions({
  queryKey: [...railKey, "summary"],
  queryFn: getDaysSummary,
});
