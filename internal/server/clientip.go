package server

import (
	"context"
	"net"
	"net/http"
	"strings"
)

type clientIPKey struct{}

// withClientIP works out each request's client IP once, for the per-IP rate limits.
//
// trustedProxies is how many proxies stand in front of the app, each appending
// the address it saw to X-Forwarded-For (Railway: 1). The client is the entry
// that many from the right; anything left of it is the visitor's own say-so.
// With 0 the header is ignored, so a client that reaches the app directly can't
// pick its own IP and dodge the limits.
func withClientIP(trustedProxies int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := resolveClientIP(r, trustedProxies)
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), clientIPKey{}, ip)))
		})
	}
}

// clientIP is the request's client IP, as withClientIP resolved it.
func clientIP(r *http.Request) string {
	ip, _ := r.Context().Value(clientIPKey{}).(string)
	return ip
}

func resolveClientIP(r *http.Request, trustedProxies int) string {
	if trustedProxies > 0 {
		var hops []string
		for _, v := range r.Header.Values("X-Forwarded-For") {
			for h := range strings.SplitSeq(v, ",") {
				hops = append(hops, strings.TrimSpace(h))
			}
		}
		// Fewer hops than proxies means the request didn't come through them all:
		// fall back to the peer rather than trust what's there.
		if len(hops) >= trustedProxies {
			if ip := hops[len(hops)-trustedProxies]; ip != "" {
				return ip
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
