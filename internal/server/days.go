package server

import (
	"net/http"
	"time"

	"github.com/boinkkitty/zoop/internal/days"
	"github.com/boinkkitty/zoop/internal/pages"
)

type islandResponse struct {
	Day      string            `json:"day"`
	Written  bool              `json:"written"`
	Title    string            `json:"title"`
	Glimpses map[string]string `json:"glimpses"`
}

type monthResponse struct {
	Month      string           `json:"month"`
	Days       []islandResponse `json:"days"`
	NewerMonth *string          `json:"newerMonth"`
	OlderMonth *string          `json:"olderMonth"`
}

const monthLayout = "2006-01"

func getMonth(ps *pages.Service, rep *reporter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		month, err := time.Parse(monthLayout, r.URL.Query().Get("month"))
		if err != nil {
			writeError(w, http.StatusNotFound, "not_found", "No such month.")
			return
		}
		m, err := ps.Month(r.Context(), userID(r), month)
		if err != nil {
			writePageError(w, r, rep, err)
			return
		}
		res := monthResponse{
			Month:      m.Month.Format(monthLayout),
			Days:       make([]islandResponse, len(m.Islands)),
			NewerMonth: monthOrNull(m.Newer),
			OlderMonth: monthOrNull(m.Older),
		}
		for i, isl := range m.Islands {
			res.Days[i] = islandResponse{Day: days.Format(isl.Day), Written: isl.Written, Title: isl.Title, Glimpses: isl.Glimpses}
		}
		writeJSON(w, http.StatusOK, res)
	}
}

func monthOrNull(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(monthLayout)
	return &s
}

type summaryResponse struct {
	FirstDay string               `json:"firstDay"`
	Today    string               `json:"today"`
	Months   []monthCountResponse `json:"months"`
}

type monthCountResponse struct {
	Month       string `json:"month"`
	WrittenDays int    `json:"writtenDays"`
}

func getSummary(ps *pages.Service, rep *reporter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sum, err := ps.Summary(r.Context(), userID(r))
		if err != nil {
			writePageError(w, r, rep, err)
			return
		}
		res := summaryResponse{
			FirstDay: days.Format(sum.FirstDay),
			Today:    days.Format(sum.Today),
			Months:   make([]monthCountResponse, len(sum.Months)),
		}
		for i, m := range sum.Months {
			res.Months[i] = monthCountResponse{Month: m.Month.Format(monthLayout), WrittenDays: m.WrittenDays}
		}
		writeJSON(w, http.StatusOK, res)
	}
}
