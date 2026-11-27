package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/boinkkitty/zoop/internal/days"
	"github.com/boinkkitty/zoop/internal/pages"
)

type pageResponse struct {
	Day           string          `json:"day"`
	Notebook      string          `json:"notebook"`
	Doc           json.RawMessage `json:"doc"`
	SchemaVersion int             `json:"schemaVersion"`
	Version       int             `json:"version"`
	UpdatedAt     *string         `json:"updatedAt"`
	Editable      bool            `json:"editable"`
	LocksAt       *string         `json:"locksAt"`
}

func toPageResponse(p pages.Page) pageResponse {
	return pageResponse{
		Day:           days.Format(p.Day),
		Notebook:      p.Notebook,
		Doc:           p.Doc,
		SchemaVersion: p.SchemaVersion,
		Version:       p.Version,
		UpdatedAt:     moment(p.UpdatedAt),
		Editable:      p.Editable,
		LocksAt:       moment(p.LocksAt),
	}
}

func getPage(ps *pages.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		day, ok := dayParam(w, r)
		if !ok {
			return
		}
		p, err := ps.Get(r.Context(), userID(r), day, chi.URLParam(r, "notebook"))
		if err != nil {
			writePageError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toPageResponse(p))
	}
}

// maxPageBody caps a save request: the whole doc is sent every time.
const maxPageBody = 512 << 10

type savePageRequest struct {
	Doc           json.RawMessage `json:"doc"`
	SchemaVersion int             `json:"schemaVersion"`
	BaseVersion   int             `json:"baseVersion"`
}

func putPage(ps *pages.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		day, ok := dayParam(w, r)
		if !ok {
			return
		}
		var req savePageRequest
		err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxPageBody)).Decode(&req)
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			writePageError(w, pages.ErrTooLong)
			return
		}
		if err != nil {
			writeError(w, http.StatusBadRequest, "validation", "The request body isn't valid JSON.")
			return
		}
		p, err := ps.Save(r.Context(), userID(r), day, chi.URLParam(r, "notebook"), pages.Save(req))
		if err != nil {
			writePageError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toPageResponse(p))
	}
}

func dayParam(w http.ResponseWriter, r *http.Request) (time.Time, bool) {
	day, err := time.Parse(time.DateOnly, chi.URLParam(r, "day"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "No such Page.")
		return time.Time{}, false
	}
	return day, true
}

// moment formats a moment as RFC 3339 UTC, or null.
func moment(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
}

func writePageError(w http.ResponseWriter, err error) {
	var conflict *pages.ConflictError
	var bad *pages.InvalidDocError
	switch {
	case errors.As(err, &bad):
		writeError(w, http.StatusBadRequest, "validation", "This Page couldn't be saved: its document isn't valid.")
	case errors.As(err, &conflict):
		writeJSON(w, http.StatusConflict, struct {
			Error apiError     `json:"error"`
			Page  pageResponse `json:"page"`
		}{
			apiError{"version_conflict", "This Page changed on another device."},
			toPageResponse(conflict.Current),
		})
	case errors.Is(err, pages.ErrClientOutdated):
		writeError(w, http.StatusConflict, "client_outdated", "Zoop was updated. Reload to keep saving.")
	case errors.Is(err, pages.ErrTooLong):
		writeError(w, http.StatusRequestEntityTooLarge, "too_long", "This Page is too long to save. Try moving some of it to another Day.")
	case errors.Is(err, pages.ErrNoPage):
		writeError(w, http.StatusNotFound, "not_found", "No such Page.")
	case errors.Is(err, pages.ErrNotFound):
		writeError(w, http.StatusUnauthorized, "unauthenticated", "Please sign in.")
	default:
		slog.Error("page request failed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "Something went wrong.")
	}
}
