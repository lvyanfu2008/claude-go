package mcp

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

// OAuthState holds the OAuth state for an MCP server.
type OAuthState struct {
	ServerName   string
	ClientID     string
	TokenURL     string
	AuthURL      string
	CodeVerifier string
	RedirectURI  string
	State        string

	mu     sync.Mutex
	Token  *oauth2.Token
	DoneCh chan struct{}
	Err    error
}

// OAuthManager manages OAuth flows for MCP servers.
// Mirrors TS services/mcp/auth.ts OAuth flow (simplified, no XAA).
type OAuthManager struct {
	mu     sync.Mutex
	states map[string]*OAuthState // state string → auth state
}

// NewOAuthManager creates a new OAuthManager.
func NewOAuthManager() *OAuthManager {
	return &OAuthManager{
		states: make(map[string]*OAuthState),
	}
}

// PerformOAuthFlow runs the full OAuth 2.0 authorization code + PKCE flow.
// It starts a local HTTP server for the redirect, opens the browser for user
// authorization, and waits for the callback.
func (m *OAuthManager) PerformOAuthFlow(
	ctx context.Context,
	serverName string,
	authServerMetadataURL string,
	clientID string,
	callbackPort int,
) (*oauth2.Token, error) {
	// Step 1: Discover OAuth metadata.
	oauthMeta, err := m.discoverOAuthMetadata(ctx, authServerMetadataURL)
	if err != nil {
		return nil, fmt.Errorf("discover OAuth metadata: %w", err)
	}

	// Step 2: Generate PKCE code verifier and challenge.
	codeVerifier, err := generateCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("generate PKCE verifier: %w", err)
	}
	codeChallenge := generateCodeChallenge(codeVerifier)

	// Step 3: Start local HTTP server for redirect.
	listener, redirectURI, err := m.startRedirectServer(callbackPort)
	if err != nil {
		return nil, fmt.Errorf("start redirect server: %w", err)
	}
	defer listener.Close()

	// Step 4: Build authorization URL.
	state := randomString(32)
	authState := &OAuthState{
		ServerName:   serverName,
		ClientID:     clientID,
		TokenURL:     oauthMeta.TokenEndpoint,
		AuthURL:      oauthMeta.AuthorizationEndpoint,
		CodeVerifier: codeVerifier,
		RedirectURI:  redirectURI,
		State:        state,
		DoneCh:       make(chan struct{}),
	}

	m.mu.Lock()
	m.states[state] = authState
	m.mu.Unlock()

	authURL := fmt.Sprintf("%s?response_type=code&client_id=%s&redirect_uri=%s&code_challenge=%s&code_challenge_method=S256&state=%s",
		oauthMeta.AuthorizationEndpoint,
		url.QueryEscape(clientID),
		url.QueryEscape(redirectURI),
		codeChallenge,
		state,
	)

	// Step 5: Open browser (platform-specific).
	if err := openBrowser(authURL); err != nil {
		return nil, fmt.Errorf("open browser: %w", err)
	}

	// Step 6: Wait for callback or timeout.
	select {
	case <-authState.DoneCh:
		if authState.Err != nil {
			return nil, authState.Err
		}
		return authState.Token, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("OAuth flow timed out")
	}
}

func (m *OAuthManager) startRedirectServer(preferredPort int) (net.Listener, string, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", preferredPort))
	if err != nil {
		// Try any available port.
		listener, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return nil, "", err
		}
	}

	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		m.handleCallback(w, r)
	})

	go http.Serve(listener, mux)

	return listener, redirectURI, nil
}

func (m *OAuthManager) handleCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	stateStr := q.Get("state")
	code := q.Get("code")
	errStr := q.Get("error")

	m.mu.Lock()
	authState, ok := m.states[stateStr]
	m.mu.Unlock()

	if !ok {
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}

	if errStr != "" {
		authState.Err = fmt.Errorf("OAuth error: %s (%s)", errStr, q.Get("error_description"))
		close(authState.DoneCh)
		fmt.Fprintf(w, "<html><body><h1>Authorization Failed</h1><p>%s</p></body></html>", errStr)
		return
	}

	if code == "" {
		authState.Err = fmt.Errorf("no authorization code received")
		close(authState.DoneCh)
		http.Error(w, "No authorization code", http.StatusBadRequest)
		return
	}

	// Exchange code for token.
	token, err := m.exchangeCode(r.Context(), authState, code)
	if err != nil {
		authState.Err = err
		close(authState.DoneCh)
		http.Error(w, "Failed to exchange code", http.StatusInternalServerError)
		return
	}

	authState.Token = token
	close(authState.DoneCh)

	fmt.Fprintf(w, "<html><body><h1>Authorization Successful</h1><p>You can close this window.</p></body></html>")
}

func (m *OAuthManager) exchangeCode(ctx context.Context, state *OAuthState, code string) (*oauth2.Token, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {state.RedirectURI},
		"client_id":     {state.ClientID},
		"code_verifier": {state.CodeVerifier},
	}

	resp, err := client.Post(state.TokenURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var token oauth2.Token
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}

	return &token, nil
}

// OAuthServerMetadata mirrors the MCP OAuth discovery metadata.
type OAuthServerMetadata struct {
	Issuer                                    string `json:"issuer"`
	AuthorizationEndpoint                     string `json:"authorization_endpoint"`
	TokenEndpoint                             string `json:"token_endpoint"`
	RegistrationEndpoint                      string `json:"registration_endpoint,omitempty"`
	RevocationEndpoint                        string `json:"revocation_endpoint,omitempty"`
	CodeChallengeMethodsSupported             []string `json:"code_challenge_methods_supported,omitempty"`
}

func (m *OAuthManager) discoverOAuthMetadata(ctx context.Context, metadataURL string) (*OAuthServerMetadata, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(metadataURL)
	if err != nil {
		return nil, fmt.Errorf("fetch OAuth metadata from %s: %w", metadataURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OAuth metadata endpoint returned %d from %s", resp.StatusCode, metadataURL)
	}

	var meta OAuthServerMetadata
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, fmt.Errorf("parse OAuth metadata: %w", err)
	}

	return &meta, nil
}

// TokenStore persists OAuth tokens for reuse.
type TokenStore struct {
	mu     sync.RWMutex
	tokens map[string]*oauth2.Token // server name → token
}

// NewTokenStore creates a new in-memory token store.
func NewTokenStore() *TokenStore {
	return &TokenStore{
		tokens: make(map[string]*oauth2.Token),
	}
}

// Get retrieves a stored token.
func (s *TokenStore) Get(serverName string) *oauth2.Token {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.tokens[serverName]
}

// Set stores a token.
func (s *TokenStore) Set(serverName string, token *oauth2.Token) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[serverName] = token
}

// Clear removes a token.
func (s *TokenStore) Clear(serverName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tokens, serverName)
}

// PKCE helpers

func generateCodeVerifier() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// generateCodeChallenge creates an S256 code challenge from a verifier.
func generateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func randomString(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)[:n]
}
