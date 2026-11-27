package server

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/boinkkitty/zoop/internal/auth"
	"github.com/boinkkitty/zoop/internal/days"
	"github.com/boinkkitty/zoop/internal/ratelimit"
)

const sessionCookie = "zoop_session"

type credentials struct {
	Email    auth.Email    `json:"email"`
	Password auth.Password `json:"password"`
	Timezone string        `json:"timezone"`
}

func signup(a *auth.Service, rep *reporter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var c credentials
		if !decode(w, r, &c) {
			return
		}
		sess, err := a.Signup(r.Context(), c.Email, c.Password, c.Timezone, clientIP(r))
		if err != nil {
			writeAuthError(w, r, rep, err)
			return
		}
		setSessionCookie(w, r, sess)
		w.WriteHeader(http.StatusCreated)
	}
}

func login(a *auth.Service, rep *reporter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var c credentials
		if !decode(w, r, &c) {
			return
		}
		sess, err := a.Login(r.Context(), c.Email, c.Password, clientIP(r))
		if err != nil {
			writeAuthError(w, r, rep, err)
			return
		}
		setSessionCookie(w, r, sess)
		w.WriteHeader(http.StatusOK)
	}
}

// logout signs out this device: the session is deleted and the cookie cleared.
func logout(a *auth.Service, rep *reporter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie(sessionCookie); err == nil {
			if err := a.Logout(r.Context(), auth.Token(c.Value)); err != nil {
				writeAuthError(w, r, rep, err)
				return
			}
		}
		http.SetCookie(w, &http.Cookie{
			Name: sessionCookie, Path: "/", MaxAge: -1,
			HttpOnly: true, Secure: isHTTPS(r), SameSite: http.SameSiteLaxMode,
		})
		w.WriteHeader(http.StatusNoContent)
	}
}

type userKey struct{}

// requireUser lets the request through only with a live session, renewing its cookie when the expiry moves.
func requireUser(a *auth.Service, rep *reporter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var token auth.Token
			if c, err := r.Cookie(sessionCookie); err == nil {
				token = auth.Token(c.Value)
			}
			sess, err := a.Authenticate(r.Context(), token)
			if err != nil {
				writeAuthError(w, r, rep, err)
				return
			}
			if rl := requestLogFrom(r.Context()); rl != nil {
				rl.userID = sess.UserID
			}
			if sess.Renewed {
				sess.Token = token
				setSessionCookie(w, r, sess)
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey{}, sess.UserID)))
		})
	}
}

// limitPerUser slows down a runaway client or script, across all of the user's devices. It runs after requireUser.
func limitPerUser(limits *ratelimit.RateLimiter, rep *reporter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := limits.Take(r.Context(), ratelimit.APIRequestsPerUser, userID(r).String()); err != nil {
				writeAuthError(w, r, rep, err)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func userID(r *http.Request) uuid.UUID { return r.Context().Value(userKey{}).(uuid.UUID) }

type meResponse struct {
	Email             string   `json:"email"`
	EmailVerified     bool     `json:"emailVerified"`
	Plan              string   `json:"plan"`
	Timezone          string   `json:"timezone"`
	PendingTimezone   *string  `json:"pendingTimezone"`
	TimezoneSwitchDay *string  `json:"timezoneSwitchDay"`
	FirstDay          string   `json:"firstDay"`
	Today             string   `json:"today"`
	GraceUntil        *string  `json:"graceUntil"`
	SignInMethods     []string `json:"signInMethods"`
}

func getMe(a *auth.Service, rep *reporter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		acct, err := a.Account(r.Context(), userID(r))
		if err != nil {
			writeAuthError(w, r, rep, err)
			return
		}
		writeJSON(w, http.StatusOK, toMe(acct))
	}
}

// patchMe changes the timezone, from tomorrow.
func patchMe(a *auth.Service, rep *reporter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Timezone string `json:"timezone"`
		}
		if !decode(w, r, &body) {
			return
		}
		acct, err := a.SetTimezone(r.Context(), userID(r), body.Timezone)
		if err != nil {
			writeAuthError(w, r, rep, err)
			return
		}
		writeJSON(w, http.StatusOK, toMe(acct))
	}
}

func toMe(acct auth.Account) meResponse {
	res := meResponse{
		Email:           acct.Email,
		EmailVerified:   acct.EmailVerified,
		Plan:            acct.Plan,
		Timezone:        acct.Timezone,
		PendingTimezone: acct.PendingTimezone,
		FirstDay:        days.Format(acct.FirstDay),
		Today:           days.Format(acct.Today),
		SignInMethods:   acct.SignInMethods,
	}
	if acct.SwitchDay != nil {
		s := days.Format(*acct.SwitchDay)
		res.TimezoneSwitchDay = &s
	}
	if acct.GraceUntil != nil {
		s := acct.GraceUntil.UTC().Format(time.RFC3339)
		res.GraceUntil = &s
	}
	return res
}

func setSessionCookie(w http.ResponseWriter, r *http.Request, sess auth.Session) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    string(sess.Token),
		Path:     "/",
		Expires:  sess.ExpiresAt,
		HttpOnly: true,
		Secure:   isHTTPS(r),
		SameSite: http.SameSiteLaxMode,
	})
}

// clientIP is the visitor's address as Railway's proxy reports it, or the peer's without a proxy.
// It trusts one hop: the proxy appends the address it saw, and anything before that is the client's say-so.
func clientIP(r *http.Request) string {
	if fwd := r.Header.Values("X-Forwarded-For"); len(fwd) > 0 {
		hops := strings.Split(fwd[len(fwd)-1], ",")
		if last := strings.TrimSpace(hops[len(hops)-1]); last != "" {
			return last
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// isHTTPS is true behind Railway's TLS-terminating proxy, and false for local http.
func isHTTPS(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

func writeAuthError(w http.ResponseWriter, r *http.Request, rep *reporter, err error) {
	var v *auth.ValidationError
	var limited *ratelimit.ExceededError
	switch {
	case errors.As(err, &limited):
		writeRateLimited(w, limited)
	case errors.As(err, &v):
		writeError(w, http.StatusBadRequest, "validation", v.Message)
	case errors.Is(err, auth.ErrEmailTaken):
		writeError(w, http.StatusConflict, "email_taken", "That email already has an account. Sign in instead.")
	case errors.Is(err, auth.ErrBadLogin):
		writeError(w, http.StatusUnauthorized, "unauthenticated", "Wrong email or password.")
	case errors.Is(err, auth.ErrNoSession):
		writeError(w, http.StatusUnauthorized, "unauthenticated", "Please sign in.")
	default:
		rep.internal(w, r, err)
	}
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "validation", "The request body isn't valid JSON.")
		return false
	}
	return true
}
