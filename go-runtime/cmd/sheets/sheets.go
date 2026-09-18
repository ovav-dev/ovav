// Package sheets implements the OVAV Google Sheets bridge.
//
// Architecture:
//   - cmd/sheets: CLI (auth, read, write, demo)
//   - cmd/sheets: zero third-party deps; stdlib net/http only
//   - cmd/sheets: credentials encrypted via internal/vault (AES-256-GCM)
//   - cmd/sheets: writes go through Sheets API v4 with allowlisted spreadsheetId
//
// Out of scope for v0.1: MCP wire transport. v0.1 is a deterministic CLI
// that proves the flow end-to-end. MCP can wrap it later.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	googleAuthURL  = "https://accounts.google.com/o/oauth2/auth"
	googleTokenURL = "https://oauth2.googleapis.com/token"
)

// Scope aliases — short names that map to OAuth2 URLs.
// Use --scopes <alias>,<alias> to request multiple on auth.
var scopeAliases = map[string]string{
	"sheets":       "https://www.googleapis.com/auth/spreadsheets",
	"spreadsheets": "https://www.googleapis.com/auth/spreadsheets",
	"script":       "https://www.googleapis.com/auth/script.projects",
	"script-ro":    "https://www.googleapis.com/auth/script.projects.readonly",
	"drive":        "https://www.googleapis.com/auth/drive",
	"drive-file":   "https://www.googleapis.com/auth/drive.file",
	"drive-ro":     "https://www.googleapis.com/auth/drive.readonly",
}

// resolveScopes turns comma-separated aliases into a space-joined
// OAuth scope string. Unknown aliases pass through verbatim (Google
// URLs) so power users can still request custom scopes.
func resolveScopes(spec string) string {
	parts := strings.Split(spec, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if v, ok := scopeAliases[p]; ok {
			out = append(out, v)
		} else {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return scopeAliases["sheets"]
	}
	return strings.Join(out, " ")
}

// AuthCodeURL builds the URL the user must open in a browser to grant
// the offline refresh_token scope. State is a CSRF nonce echoed back.
// `scopes` is a space-joined OAuth scope list — pass multiple to
// request them all in one consent screen.
func AuthCodeURL(clientID, redirectURI, state, scopes string) string {
	v := url.Values{}
	v.Set("client_id", clientID)
	v.Set("redirect_uri", redirectURI)
	v.Set("response_type", "code")
	v.Set("scope", scopes)
	v.Set("access_type", "offline")
	v.Set("prompt", "consent")
	v.Set("include_granted_scopes", "true")
	v.Set("state", state)
	return googleAuthURL + "?" + v.Encode()
}

// ExchangeCode trades an authorization code for access+refresh tokens.
func ExchangeCode(clientID, clientSecret, redirectURI, code string) (*Creds, error) {
	form := url.Values{}
	form.Set("code", code)
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("redirect_uri", redirectURI)
	form.Set("grant_type", "authorization_code")

	resp, err := http.PostForm(googleTokenURL, form)
	if err != nil {
		return nil, fmt.Errorf("sheets: token POST: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sheets: token exchange HTTP %d: %s", resp.StatusCode, truncate(string(body), 240))
	}

	var tr struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
		TokenType    string `json:"token_type"`
	}
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("sheets: parse token: %w", err)
	}
	if tr.RefreshToken == "" {
		return nil, fmt.Errorf("sheets: no refresh_token returned (re-consent required): %s", truncate(string(body), 240))
	}
	return &Creds{
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		Scope:        tr.Scope,
		TokenExpiry:  time.Now().Unix() + int64(tr.ExpiresIn),
	}, nil
}

// Refresh swaps a refresh_token for a fresh access_token. Mutates c.
func Refresh(c *Creds) error {
	form := url.Values{}
	form.Set("refresh_token", c.RefreshToken)
	form.Set("client_id", c.ClientID)
	form.Set("client_secret", c.ClientSecret)
	form.Set("grant_type", "refresh_token")
	resp, err := http.PostForm(googleTokenURL, form)
	if err != nil {
		return fmt.Errorf("sheets: refresh POST: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("sheets: refresh HTTP %d: %s", resp.StatusCode, truncate(string(body), 240))
	}
	var tr struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		Scope       string `json:"scope"`
	}
	if err := json.Unmarshal(body, &tr); err != nil {
		return fmt.Errorf("sheets: parse refresh: %w", err)
	}
	c.AccessToken = tr.AccessToken
	c.Scope = tr.Scope
	c.TokenExpiry = time.Now().Unix() + int64(tr.ExpiresIn)
	return nil
}

// EnsureFresh refreshes the access token if it is within 60s of expiry.
func EnsureFresh(c *Creds) error {
	if c.AccessToken != "" && time.Now().Unix() < c.TokenExpiry-60 {
		return nil
	}
	if c.RefreshToken == "" {
		return fmt.Errorf("sheets: no refresh_token — re-auth required")
	}
	return Refresh(c)
}

// AuthorizedClient returns an *http.Client that injects the Bearer token
// and auto-refreshes once on 401.
func AuthorizedClient(c *Creds) *http.Client {
	tr := &tokenTransport{base: http.DefaultTransport, creds: c}
	return &http.Client{Timeout: 30 * time.Second, Transport: tr}
}

type tokenTransport struct {
	base   http.RoundTripper
	creds  *Creds
	retried bool
}

func (t *tokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.creds.AccessToken != "" {
		req = req.Clone(req.Context())
		req.Header.Set("Authorization", "Bearer "+t.creds.AccessToken)
	}
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusUnauthorized || t.retried {
		return resp, nil
	}
	_ = resp.Body.Close()
	if err := Refresh(t.creds); err != nil {
		return nil, fmt.Errorf("sheets: 401 retry refresh: %w", err)
	}
	t.retried = true
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+t.creds.AccessToken)
	return t.base.RoundTrip(req)
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
