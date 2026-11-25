package server

import (
	"encoding/json"
	"net/http"
)

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
