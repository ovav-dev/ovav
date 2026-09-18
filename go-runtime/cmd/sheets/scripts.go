// Package main — Apps Script API client for the OVAV Sheets bridge.
//
// Wraps the Apps Script REST API v1 with the minimum surface we need:
//   - list projects
//   - find the script bound to a spreadsheet (by parentId)
//   - download every .gs / .html file
//   - upload (push) changes
//   - run a server-side function
//   - list versions and create new ones
//
// All operations require either `script.projects.readonly` (read paths)
// or `script.projects` (mutations). OAuth scopes are negotiated at
// auth time via `--scopes script,script-ro,...`.
//
// The Apps Script API is documented at
// https://developers.google.com/apps-script/api/reference/rest
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const appsScriptAPIBase = "https://script.googleapis.com/v1"

// ScriptProject is a row from `projects.list`.
type ScriptProject struct {
	ScriptID   string `json:"scriptId"`
	Title      string `json:"title"`
	ParentID   string `json:"parentId"`
	CreateTime string `json:"createTime"`
	UpdateTime string `json:"updateTime"`
	Creator    struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"creator"`
}

// ScriptFile is one file inside a script project (.gs, .html, .json).
//
// Note: LastModifyUser is read-only metadata from the API and is
// never serialized when pushing back — putting the field back would
// trigger "Unknown name time at lastModifyUser" because the schema
// expects name/email/domain/photoUrl not time/user.
type ScriptFile struct {
	Name          string                 `json:"name"`
	Type          string                 `json:"type"` // "SERVER_JS", "HTML", "JSON"
	Source        string                 `json:"source,omitempty"`
	SourceRaw     string                 `json:"-"`
	Extension     string                 `json:"-"`
	LastModifyRaw map[string]interface{} `json:"-"`
}

type scriptContentResp struct {
	ScriptID string       `json:"scriptId"`
	Files    []ScriptFile `json:"files"`
}

// ScriptClient talks to the Apps Script API using the same bearer
// token infrastructure as Sheets.
type ScriptClient struct {
	c    *Creds
	http *http.Client
}

// NewScriptClient wraps an existing *Creds (already loaded from vault).
func NewScriptClient(c *Creds) *ScriptClient {
	return &ScriptClient{c: c, http: AuthorizedClient(c)}
}

// ListProjects returns the projects visible to the OAuth identity.
func (sc *ScriptClient) ListProjects() ([]ScriptProject, error) {
	if err := EnsureFresh(sc.c); err != nil {
		return nil, err
	}
	u := appsScriptAPIBase + "/projects?pageSize=50"
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	resp, err := sc.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("script: list: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("script: list HTTP %d: %s", resp.StatusCode, truncate(string(body), 240))
	}
	var out struct {
		Projects []ScriptProject `json:"projects"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("script: parse list: %w", err)
	}
	return out.Projects, nil
}

// FindByContainer walks the project list and returns the one whose
// ParentID equals the given spreadsheet (or Drive folder) ID.
func (sc *ScriptClient) FindByContainer(containerID string) (*ScriptProject, error) {
	projects, err := sc.ListProjects()
	if err != nil {
		return nil, err
	}
	for i := range projects {
		if projects[i].ParentID == containerID {
			return &projects[i], nil
		}
	}
	return nil, fmt.Errorf("script: no project bound to container %s (visible: %d)", containerID, len(projects))
}

// GetContent downloads every file in the project.
func (sc *ScriptClient) GetContent(scriptID string) (*scriptContentResp, error) {
	if err := EnsureFresh(sc.c); err != nil {
		return nil, err
	}
	u := fmt.Sprintf("%s/projects/%s/content", appsScriptAPIBase, scriptID)
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	resp, err := sc.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("script: get content: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("script: get content HTTP %d: %s", resp.StatusCode, truncate(string(body), 240))
	}
	var out scriptContentResp
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("script: parse content: %w", err)
	}
	return &out, nil
}

// PullToDir downloads the script and writes every file to disk under
// outDir. Returns the manifest (list of written paths + sha256 hashes).
func (sc *ScriptClient) PullToDir(scriptID, outDir string) (*ScriptManifest, error) {
	content, err := sc.GetContent(scriptID)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("script: mkdir %s: %w", outDir, err)
	}
	mf := &ScriptManifest{
		ScriptID:  scriptID,
		PulledAt:  time.Now().UTC(),
		Files:     make([]ScriptFileMeta, 0, len(content.Files)),
	}
	for _, f := range content.Files {
		ext := ".gs"
		switch f.Type {
		case "HTML":
			ext = ".html"
		case "JSON":
			ext = ".json"
		}
		path := filepath.Join(outDir, f.Name+ext)
		body := []byte(f.Source)
		if err := os.WriteFile(path, body, 0o644); err != nil {
			return nil, fmt.Errorf("script: write %s: %w", path, err)
		}
		sum := sha256.Sum256(body)
		mf.Files = append(mf.Files, ScriptFileMeta{
			Name: f.Name, Type: f.Type, Path: f.Name + ext,
			SizeBytes: len(body), SHA256: hex.EncodeToString(sum[:]),
		})
	}
	// write manifest
	mfBytes, _ := json.MarshalIndent(mf, "", "  ")
	if err := os.WriteFile(filepath.Join(outDir, "MANIFEST.json"), mfBytes, 0o644); err != nil {
		return nil, fmt.Errorf("script: write manifest: %w", err)
	}
	return mf, nil
}

// ScriptFileMeta is one entry in the on-disk MANIFEST.json.
type ScriptFileMeta struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Path      string `json:"path"`
	SizeBytes int    `json:"size_bytes"`
	SHA256    string `json:"sha256"`
}

// ScriptManifest is the result of a Pull (written to MANIFEST.json).
type ScriptManifest struct {
	ScriptID  string           `json:"script_id"`
	PulledAt  time.Time        `json:"pulled_at"`
	Files     []ScriptFileMeta `json:"files"`
}

// PushFromDir uploads every file under inDir to the script project.
// Always preserves existing files (Apps Script requires the manifest);
// inDir is treated as the authoritative source for any file present there.
// `confirm=true` skips the plan print and applies directly (NOT recommended).
func (sc *ScriptClient) PushFromDir(scriptID, inDir string, confirm bool) (*ScriptManifest, error) {
	existing, err := sc.GetContent(scriptID)
	if err != nil {
		return nil, err
	}
	existingMap := map[string]ScriptFile{}
	for _, f := range existing.Files {
		existingMap[f.Name] = f
	}

	entries, err := os.ReadDir(inDir)
	if err != nil {
		return nil, fmt.Errorf("script: read dir: %w", err)
	}
	// Build a name-keyed map of new files.
	newMap := map[string]ScriptFile{}
	for _, e := range entries {
		if e.IsDir() || e.Name() == "MANIFEST.json" {
			continue
		}
		base := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		ext := filepath.Ext(e.Name())
		fType := "SERVER_JS"
		switch ext {
		case ".html":
			fType = "HTML"
		case ".json":
			fType = "JSON"
		}
		body, err := os.ReadFile(filepath.Join(inDir, e.Name()))
		if err != nil {
			return nil, err
		}
		newMap[base] = ScriptFile{Name: base, Type: fType, Source: string(body)}
	}

	// Build plan: for every existing file, replace with newMap if
	// present; otherwise keep existing. Then append newMap entries
	// that don't already exist. This guarantees the manifest is never
	// accidentally deleted.
	plan := []ScriptFile{}
	planNames := map[string]bool{}
	for _, f := range existing.Files {
		if n, ok := newMap[f.Name]; ok {
			plan = append(plan, n)
		} else {
			// Keep existing untouched (preserves manifest, etc).
			plan = append(plan, ScriptFile{Name: f.Name, Type: f.Type, Source: f.Source})
		}
		planNames[f.Name] = true
	}
	for name, n := range newMap {
		if !planNames[name] {
			plan = append(plan, n)
		}
	}

	// Compute diff counts vs. existing.
	var adds, mods int
	for _, p := range plan {
		old, ok := existingMap[p.Name]
		if !ok {
			adds++
		} else if old.Source != p.Source {
			mods++
		}
	}

	fmt.Printf("Push plan for script %s\n", scriptID)
	fmt.Printf("  + %d new  ~ %d modified  (= preserved: %d)\n", adds, mods, len(existing.Files)-mods)
	for _, p := range plan {
		old, ok := existingMap[p.Name]
		status := "+"
		if ok {
			if old.Source != p.Source {
				status = "~"
			} else {
				status = "="
			}
		}
		fmt.Printf("  %s %s  (%s, %d bytes)\n", status, p.Name, p.Type, len(p.Source))
	}

	if !confirm {
		fmt.Println("\n  pass --confirm to apply")
		return &ScriptManifest{ScriptID: scriptID, PulledAt: time.Now().UTC()}, nil
	}

	body, _ := json.Marshal(map[string]any{"files": plan})
	u := fmt.Sprintf("%s/projects/%s/content", appsScriptAPIBase, scriptID)
	req, _ := http.NewRequest(http.MethodPut, u, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := sc.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("script: push: %w", err)
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("script: push HTTP %d: %s", resp.StatusCode, truncate(string(rb), 320))
	}
	return sc.PullToDir(scriptID, inDir)
}

// RunFunction executes a server-side function by name.
// `params` is a JSON object passed to the Apps Script function.
func (sc *ScriptClient) RunFunction(scriptID, function string, params map[string]any) (any, error) {
	if err := EnsureFresh(sc.c); err != nil {
		return nil, err
	}
	body, _ := json.Marshal(map[string]any{
		"function": function,
		"parameters": params,
		"devMode":   true,
	})
	u := fmt.Sprintf("%s/scripts/%s:run", appsScriptAPIBase, scriptID)
	req, _ := http.NewRequest(http.MethodPost, u, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := sc.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("script: run: %w", err)
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("script: run HTTP %d: %s", resp.StatusCode, truncate(string(rb), 320))
	}
	// Apps Script :run returns { done: bool, response: { result, error? } }
	// or on script-level error: { done: bool, error: {...} }.
	var out struct {
		Done     bool `json:"done"`
		Response *struct {
			Result any `json:"result"`
		} `json:"response"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Details []struct {
				Type      string `json:"@type"`
				ScriptStackTraceElements []map[string]any `json:"scriptStackTraceElements"`
				ErrorMessage string `json:"errorMessage"`
			} `json:"details"`
		} `json:"error,omitempty"`
	}
	if err := json.Unmarshal(rb, &out); err != nil {
		return nil, fmt.Errorf("script: parse run: %w", err)
	}
	if out.Error != nil {
		msg := out.Error.Message
		if len(out.Error.Details) > 0 && out.Error.Details[0].ErrorMessage != "" {
			msg += ": " + out.Error.Details[0].ErrorMessage
		}
		return nil, fmt.Errorf("script: %s() %s", function, msg)
	}
	if out.Response == nil {
		return nil, nil
	}
	return out.Response.Result, nil
}

// CreateVersion snapshots the current code as an immutable version.
// description should be a human-readable changelog.
func (sc *ScriptClient) CreateVersion(scriptID, description string) (int64, error) {
	if err := EnsureFresh(sc.c); err != nil {
		return 0, err
	}
	body, _ := json.Marshal(map[string]any{"description": description})
	u := fmt.Sprintf("%s/projects/%s/versions", appsScriptAPIBase, scriptID)
	req, _ := http.NewRequest(http.MethodPost, u, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := sc.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("script: version: %w", err)
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("script: version HTTP %d: %s", resp.StatusCode, truncate(string(rb), 320))
	}
	var out struct {
		VersionNumber int64 `json:"versionNumber"`
	}
	if err := json.Unmarshal(rb, &out); err != nil {
		return 0, fmt.Errorf("script: parse version: %w", err)
	}
	return out.VersionNumber, nil
}

// ListVersions returns every immutable version of the project.
func (sc *ScriptClient) ListVersions(scriptID string) ([]ScriptVersion, error) {
	if err := EnsureFresh(sc.c); err != nil {
		return nil, err
	}
	u := fmt.Sprintf("%s/projects/%s/versions", appsScriptAPIBase, scriptID)
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	resp, err := sc.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("script: list versions: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("script: list versions HTTP %d: %s", resp.StatusCode, truncate(string(body), 240))
	}
	var out struct {
		Versions []ScriptVersion `json:"versions"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("script: parse versions: %w", err)
	}
	// newest first
	sort.Slice(out.Versions, func(i, j int) bool { return out.Versions[i].VersionNumber > out.Versions[j].VersionNumber })
	return out.Versions, nil
}

// ScriptVersion is one immutable snapshot.
type ScriptVersion struct {
	ScriptID      string `json:"scriptId"`
	VersionNumber int64  `json:"versionNumber"`
	Description   string `json:"description"`
	CreateTime    string `json:"createTime"`
}

// ── helpers ──────────────────────────────────────────────────────────

// decodeSpreadIDs parses spreadsheet IDs from common inputs.
func decodeSpreadIDs(spec string) []string {
	parts := strings.Split(spec, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// url.Values is needed for query string building.
var _ = url.Values{}

// CaptureForScript snapshots an Apps Script payload to the same vault
// as Sheet snapshots. Reuses the Sheets Client snapshot pipeline.
func (sc *ScriptClient) CaptureForScript(repoRoot, scriptID, op string, payload []byte) (*SnapshotRecord, error) {
	cl := &Client{c: sc.c, spreadsheetID: "scripts", http: sc.http}
	return cl.Capture(repoRoot, op, "before "+op, "_scripts_"+sanitize(scriptID), payload)
}
