// Package googletest is a stub OpenID Connect provider standing in for Google in tests.
// It serves discovery, JWKS, an authorize endpoint that signs straight in as whoever the
// test chose, and a token endpoint that enforces PKCE S256 and returns a signed ID token.
package googletest

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"
)

const (
	ClientID     = "zoop-test-client"
	ClientSecret = "zoop-test-secret"
	keyID        = "stub-key"
)

// Identity is the Google account the stub signs in as.
type Identity struct {
	Subject       string
	Email         string
	EmailVerified bool
}

// AuthRequest is what the app sent to the authorize endpoint.
type AuthRequest struct {
	ClientID, RedirectURI, ResponseType, Scope, State, Nonce, CodeChallenge, CodeChallengeMethod string
}

// Provider is a running stub. Set who signs in with As before the browser reaches authorize.
type Provider struct {
	srv *httptest.Server
	key *rsa.PrivateKey
	now func() time.Time
	mu  sync.Mutex
	as  Identity
	// nonceOverride, when set, replaces the nonce in issued ID tokens.
	nonceOverride *string
	codes         map[string]grant
	last          AuthRequest
}

type grant struct {
	req AuthRequest
	who Identity
}

// Start runs a stub provider until the test ends. now is the time ID tokens are issued at.
func Start(t testing.TB, now func() time.Time) *Provider {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	p := &Provider{key: key, now: now, codes: map[string]grant{}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /.well-known/openid-configuration", p.discovery)
	mux.HandleFunc("GET /keys", p.jwks)
	mux.HandleFunc("GET /authorize", p.authorize)
	mux.HandleFunc("POST /token", p.token)
	p.srv = httptest.NewServer(mux)
	t.Cleanup(p.srv.Close)
	return p
}

// Issuer is the provider's issuer URL, to configure the app with.
func (p *Provider) Issuer() string { return p.srv.URL }

// As chooses the Google account the next sign-ins come back as.
func (p *Provider) As(who Identity) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.as = who
}

// IssueWrongNonce makes later ID tokens carry a nonce the app never sent.
func (p *Provider) IssueWrongNonce() {
	p.mu.Lock()
	defer p.mu.Unlock()
	n := "not-the-nonce-you-sent"
	p.nonceOverride = &n
}

// LastAuthRequest is what the app most recently sent to authorize.
func (p *Provider) LastAuthRequest() AuthRequest {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.last
}

func (p *Provider) discovery(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                p.srv.URL,
		"authorization_endpoint":                p.srv.URL + "/authorize",
		"token_endpoint":                        p.srv.URL + "/token",
		"jwks_uri":                              p.srv.URL + "/keys",
		"response_types_supported":              []string{"code"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"code_challenge_methods_supported":      []string{"S256"},
	})
}

func (p *Provider) jwks(w http.ResponseWriter, _ *http.Request) {
	pub := p.key.PublicKey
	writeJSON(w, http.StatusOK, map[string]any{"keys": []map[string]string{{
		"kty": "RSA", "alg": "RS256", "use": "sig", "kid": keyID,
		"n": b64(pub.N.Bytes()),
		"e": b64(big.NewInt(int64(pub.E)).Bytes()),
	}}})
}

// authorize skips any consent screen: it redirects straight back with a code.
func (p *Provider) authorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	req := AuthRequest{
		ClientID: q.Get("client_id"), RedirectURI: q.Get("redirect_uri"),
		ResponseType: q.Get("response_type"), Scope: q.Get("scope"),
		State: q.Get("state"), Nonce: q.Get("nonce"),
		CodeChallenge: q.Get("code_challenge"), CodeChallengeMethod: q.Get("code_challenge_method"),
	}
	code := randomHex()
	p.mu.Lock()
	p.last = req
	p.codes[code] = grant{req: req, who: p.as}
	p.mu.Unlock()

	back, err := url.Parse(req.RedirectURI)
	if err != nil || req.ClientID != ClientID {
		http.Error(w, "bad client", http.StatusBadRequest)
		return
	}
	v := back.Query()
	v.Set("code", code)
	v.Set("state", req.State)
	back.RawQuery = v.Encode()
	http.Redirect(w, r, back.String(), http.StatusFound)
}

// token redeems a code once, only for the client that got it and only with the PKCE verifier.
func (p *Provider) token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		tokenError(w, "invalid_request")
		return
	}
	id, secret, ok := r.BasicAuth()
	if !ok {
		id, secret = r.PostForm.Get("client_id"), r.PostForm.Get("client_secret")
	}
	if id != ClientID || secret != ClientSecret {
		tokenError(w, "invalid_client")
		return
	}
	p.mu.Lock()
	g, found := p.codes[r.PostForm.Get("code")]
	delete(p.codes, r.PostForm.Get("code"))
	nonce := g.req.Nonce
	if p.nonceOverride != nil {
		nonce = *p.nonceOverride
	}
	p.mu.Unlock()

	sum := sha256.Sum256([]byte(r.PostForm.Get("code_verifier")))
	switch {
	case !found, r.PostForm.Get("grant_type") != "authorization_code",
		r.PostForm.Get("redirect_uri") != g.req.RedirectURI:
		tokenError(w, "invalid_grant")
		return
	case g.req.CodeChallengeMethod != "S256" || b64(sum[:]) != g.req.CodeChallenge:
		tokenError(w, "invalid_grant")
		return
	}

	now := p.now()
	idToken := p.sign(map[string]any{
		"iss": p.srv.URL, "aud": ClientID, "sub": g.who.Subject,
		"email": g.who.Email, "email_verified": g.who.EmailVerified,
		"nonce": nonce, "iat": now.Unix(), "exp": now.Add(time.Hour).Unix(),
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": randomHex(), "token_type": "Bearer", "expires_in": 3600, "id_token": idToken,
	})
}

func (p *Provider) sign(claims map[string]any) string {
	header, _ := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT", "kid": keyID})
	body, _ := json.Marshal(claims)
	signing := b64(header) + "." + b64(body)
	sum := sha256.Sum256([]byte(signing))
	sig, err := rsa.SignPKCS1v15(rand.Reader, p.key, crypto.SHA256, sum[:])
	if err != nil {
		panic(err)
	}
	return signing + "." + b64(sig)
}

func tokenError(w http.ResponseWriter, code string) {
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": code})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func randomHex() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
