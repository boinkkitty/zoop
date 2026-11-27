package server

import (
	"net/http"
	"testing"
)

func TestTheClientIPTrustsOnlyTheHopsOfConfiguredProxies(t *testing.T) {
	tests := []struct {
		name    string
		proxies int
		xff     []string
		want    string
	}{
		{"no proxy ignores the header", 0, []string{"203.0.113.9"}, "192.0.2.1"},
		{"no proxy and no header is the peer", 0, nil, "192.0.2.1"},
		{"one proxy takes the hop it appended", 1, []string{"203.0.113.9, 198.51.100.4"}, "198.51.100.4"},
		{"one proxy ignores what the client sent first", 1, []string{"6.6.6.6", "198.51.100.4"}, "198.51.100.4"},
		{"two proxies take the second hop from the right", 2, []string{"6.6.6.6, 203.0.113.9, 198.51.100.4"}, "203.0.113.9"},
		{"fewer hops than proxies falls back to the peer", 2, []string{"203.0.113.9"}, "192.0.2.1"},
		{"one proxy but no header is the peer", 1, nil, "192.0.2.1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := http.NewRequest(http.MethodGet, "/", nil)
			if err != nil {
				t.Fatal(err)
			}
			r.RemoteAddr = "192.0.2.1:54321"
			for _, v := range tt.xff {
				r.Header.Add("X-Forwarded-For", v)
			}
			if got := resolveClientIP(r, tt.proxies); got != tt.want {
				t.Errorf("resolveClientIP = %q, want %q", got, tt.want)
			}
		})
	}
}
