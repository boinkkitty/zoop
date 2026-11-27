package server

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/boinkkitty/zoop/internal/auth"
)

// googleFlowCookie carries a Google sign-in from start to callback. It's sent only to the
// callback path, lives 10 minutes, and is cleared when the callback reads it, so a state is used once.
const (
	googleFlowCookie = "zoop_google_flow"
	googlePath       = "/api/v1/auth/google/"
	googleFlowMaxAge = 10 * 60
)

// googleStart sends the browser to Google. ?timezone= is the browser's zone, used if this creates the account.
func googleStart(a *auth.Service, rep *reporter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		to, flow, err := a.StartGoogle(r.Context(), r.URL.Query().Get("timezone"))
		if err != nil {
			redirectToSignIn(w, r, rep, err)
			return
		}
		raw, err := json.Marshal(flow)
		if err != nil {
			redirectToSignIn(w, r, rep, err)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name: googleFlowCookie, Value: base64.RawURLEncoding.EncodeToString(raw),
			Path: googlePath, MaxAge: googleFlowMaxAge,
			// Lax still sends it on Google's top-level redirect back to the callback.
			HttpOnly: true, Secure: isHTTPS(r), SameSite: http.SameSiteLaxMode,
		})
		http.Redirect(w, r, to, http.StatusFound)
	}
}

// googleCallback finishes the sign-in Google sent the browser back from, then opens the app.
func googleCallback(a *auth.Service, rep *reporter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flow := readGoogleFlow(r)
		clearGoogleFlow(w, r)
		q := r.URL.Query()
		if q.Get("error") != "" { // the user cancelled, or Google refused
			redirectToSignIn(w, r, rep, auth.ErrGoogleFailed)
			return
		}
		sess, err := a.FinishGoogle(r.Context(), flow, q.Get("state"), q.Get("code"))
		if err != nil {
			redirectToSignIn(w, r, rep, err)
			return
		}
		setSessionCookie(w, r, sess)
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func readGoogleFlow(r *http.Request) auth.GoogleFlow {
	var flow auth.GoogleFlow
	c, err := r.Cookie(googleFlowCookie)
	if err != nil {
		return flow
	}
	raw, err := base64.RawURLEncoding.DecodeString(c.Value)
	if err != nil {
		return flow
	}
	_ = json.Unmarshal(raw, &flow) // a mangled cookie leaves the flow empty, which FinishGoogle refuses
	return flow
}

func clearGoogleFlow(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name: googleFlowCookie, Path: googlePath, MaxAge: -1,
		HttpOnly: true, Secure: isHTTPS(r), SameSite: http.SameSiteLaxMode,
	})
}

// redirectToSignIn sends the browser to the sign-in screen with a code it can explain.
func redirectToSignIn(w http.ResponseWriter, r *http.Request, rep *reporter, err error) {
	code := "google_failed"
	var v *auth.ValidationError
	switch {
	case errors.Is(err, auth.ErrGoogleUnavailable):
		code = "google_unavailable"
	case errors.Is(err, auth.ErrGoogleUnverified):
		code = "google_unverified"
	case errors.Is(err, auth.ErrDeactivated):
		code = "account_deactivated"
	case errors.Is(err, auth.ErrGoogleFailed), errors.As(err, &v):
		slog.Info("google sign-in refused", "err", err)
	default:
		rep.report(r, err)
	}
	http.Redirect(w, r, "/signin?error="+url.QueryEscape(code), http.StatusFound)
}
