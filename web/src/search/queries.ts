import {
  infiniteQueryOptions,
  keepPreviousData,
  useInfiniteQuery,
} from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { type SearchResults, search } from "../api";

// Search waits this long after typing stops before asking the server.
export const DEBOUNCE_MS = 200;

// A query's results, 20 at a time by cursor. When the query changes the old request
// loses its observer and TanStack Query aborts it through the signal.
export const searchQuery = (q: string) =>
  infiniteQueryOptions({
    queryKey: ["search", q],
    queryFn: ({ pageParam, signal }) => search(q, pageParam, signal),
    initialPageParam: null as string | null,
    getNextPageParam: (last: SearchResults) => last.nextCursor ?? undefined,
    enabled: q.trim() !== "",
    placeholderData: keepPreviousData, // old results stay, dimmed, while a new query loads
  });

// useSearch searches for q from its first character, once typing has paused.
export function useSearch(q: string) {
  const [settled, setSettled] = useState(q);
  useEffect(() => {
    const t = setTimeout(() => setSettled(q), DEBOUNCE_MS);
    return () => clearTimeout(t);
  }, [q]);
  const query = useInfiniteQuery(searchQuery(settled));
  const results = query.data?.pages.flatMap((p) => p.results) ?? [];
  return {
    settled,
    results,
    // Typing ahead of the request, or the request still out: what's shown is for an older query.
    stale: settled !== q || query.isPlaceholderData,
    loading: settled !== q || query.isFetching,
    firstLoad:
      results.length === 0 &&
      (settled !== q || query.isPending || query.isPlaceholderData),
    hasMore: query.hasNextPage,
    loadMore: () => {
      if (query.hasNextPage && !query.isFetchingNextPage)
        void query.fetchNextPage();
    },
  };
}
