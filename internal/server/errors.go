package server

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"

	"github.com/boinkkitty/zoop/internal/ratelimit"
)

// writeRateLimited is 429 rate_limited, with Retry-After in whole seconds until the window ends.
func writeRateLimited(w http.ResponseWriter, e *ratelimit.ExceededError) {
	w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(e.RetryAfter.Seconds()))))
	writeError(w, http.StatusTooManyRequests, "rate_limited", "Too many attempts. Try again in a little while.")
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// writeError writes the API's error shape: {"error": {"code", "message"}}.
func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Error apiError `json:"error"`
	}{apiError{code, message}})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
