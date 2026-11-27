package server

import (
	"errors"
	"net/http"

	"github.com/boinkkitty/zoop/internal/days"
	"github.com/boinkkitty/zoop/internal/search"
)

type segmentResponse struct {
	Text  string `json:"text"`
	Match bool   `json:"match"`
}

type searchResultResponse struct {
	Day      string            `json:"day"`
	Notebook string            `json:"notebook"`
	Title    string            `json:"title"`
	Snippet  []segmentResponse `json:"snippet"`
}

type searchResponse struct {
	Results    []searchResultResponse `json:"results"`
	NextCursor *string                `json:"nextCursor"`
}

func getSearch(s *search.Service, rep *reporter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		got, err := s.Search(r.Context(), userID(r), r.URL.Query().Get("q"), r.URL.Query().Get("cursor"))
		if errors.Is(err, search.ErrBadCursor) {
			writeError(w, http.StatusBadRequest, "validation", "That search cursor isn't valid.")
			return
		}
		if err != nil {
			rep.internal(w, r, err)
			return
		}
		res := searchResponse{Results: make([]searchResultResponse, len(got.Results))}
		for i, g := range got.Results {
			seg := make([]segmentResponse, len(g.Snippet))
			for j, s := range g.Snippet {
				seg[j] = segmentResponse(s)
			}
			res.Results[i] = searchResultResponse{Day: days.Format(g.Day), Notebook: g.Notebook, Title: g.Title, Snippet: seg}
		}
		if got.Next != "" {
			res.NextCursor = &got.Next
		}
		writeJSON(w, http.StatusOK, res)
	}
}
