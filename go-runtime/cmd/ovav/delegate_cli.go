// delegate_cli.go — Native OVAV delegation command
// C5-T3 Bug B root fix: OVAV gobernando sus propios agentes sin pasar por actor tool.
//
// Usage:
//
//	ovav delegate lead-eidren "Investigar A2A mesh runtime"
//	ovav delegate team-clara "Coverage sprint en validators/"
//	ovav delegate --agent lead-eidren --task "Research task" --json
//
// Este command:
//  1. Resuelve el agent_id al perfil OVAV (via subagent catalog)
//  2. Carga el contexto del workspace (git state, files)
//  3. Genera un payload JSON estructurado con el system prompt del agente
//  4. Escribe el payload a /tmp/ovav-delegate-<pid>.json para pickup por sesion
//
// Para ejecución completa con output del modelo, usar ovav-delegate workflow
// desde una sesión MiMoCode con workflow engine activo.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ovav/ovav/internal/cli"
	"github.com/ovav/ovav/internal/subagent"
)

func cmdDelegate(args []string) int {
	if len(args) == 0 {
		printDelegateHelp()
		return 0
	}

	// Parse flags
	var agentID, taskText string
	var jsonOut bool
	var i int
	for i = 0; i < len(args); i++ {
		switch args[i] {
		case "--json", "-json":
			jsonOut = true
		case "--agent", "-a":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "ERROR: --agent requires value\n")
				return 2
			}
			agentID = args[i+1]
			i++
		case "--task", "-t":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "ERROR: --task requires value\n")
				return 2
			}
			taskText = args[i+1]
			i++
		case "--help", "-h":
			printDelegateHelp()
			return 0
		default:
			// Positional: agentID [task...]
			if agentID == "" {
				agentID = args[i]
			} else if taskText == "" {
				taskText = strings.Join(args[i:], " ")
				i = len(args)
			}
		}
	}

	if agentID == "" {
		fmt.Fprintf(os.Stderr, "ERROR: agent_id required\n")
		return 2
	}
	if taskText == "" {
		fmt.Fprintf(os.Stderr, "ERROR: task text required\n")
		return 2
	}

	repoRoot, err := cli.FindRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: not in OVAV repo: %v\n", err)
		return 3
	}

	// Load subagent catalog
	catalog, err := subagent.LoadCatalog(repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR loading catalog: %v\n", err)
		return 3
	}

	// Resolve agent (handles aliases like "eidren" → "lead-eidren")
	resolution := catalog.Resolve(agentID)
	if resolution.Error != "" {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", resolution.Error)
		fmt.Fprintf(os.Stderr, "Run 'ovav resolve-subagent --list' to see all agents\n")
		return 3
	}

	// Pick the best match: exact > alias
	var agent subagent.Agent
	if len(resolution.ExactMatches) > 0 {
		agent = resolution.ExactMatches[0]
	} else if len(resolution.AliasMatches) > 0 {
		agent = resolution.AliasMatches[0]
	}

	if agent.ID == "" {
		fmt.Fprintf(os.Stderr, "ERROR: could not resolve agent for '%s'\n", agentID)
		return 3
	}

	// Load agent profile file (use FilePath from catalog entry)
	profile, err := loadAgentProfile(repoRoot, agent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: could not load profile for %s: %v\n", agent.ID, err)
	}

	// Build workspace context
	ctx := buildWorkspaceContext(repoRoot)

	// Build delegation payload
	payload := DelegationPayload{
		AgentID:      agent.ID,
		AgentName:    agent.Name,
		AgentArea:    agent.Area,
		AgentKind:    agent.Kind,
		Task:         taskText,
		Profile:      profile,
		WorkspaceCtx: ctx,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		SessionID:    os.Getenv("OVAV_SESSION_ID"),
	}

	// Write payload to temp file for pickup by session
	payloadFile := filepath.Join(os.TempDir(), fmt.Sprintf("ovav-delegate-%d.json", os.Getpid()))
	data, _ := json.MarshalIndent(payload, "", "  ")
	if err := os.WriteFile(payloadFile, data, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: writing payload: %v\n", err)
		return 3
	}

	if jsonOut {
		out := map[string]interface{}{
			"status":       "ok",
			"payload_file": payloadFile,
			"agent_id":     agent.ID,
			"agent_name":   agent.Name,
			"agent_area":   agent.Area,
			"task_preview": truncateStr(taskText, 80),
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(out)
		return 0
	}

	// Human-readable output
	fmt.Printf("🚀 Delegando a %s (%s / %s)\n", agent.Name, agent.ID, agent.Area)
	fmt.Printf("📋 Task: %s\n", truncateStr(taskText, 120))
	fmt.Printf("📁 Contexto: %d files, branch %s, %s\n",
		len(ctx.ModifiedFiles), ctx.Branch, ctx.Head)
	fmt.Printf("💾 Payload: %s\n", payloadFile)
	fmt.Printf("\n✅ Listo para ejecución via MiMoCode session.\n")
	fmt.Printf("   El task será ejecutado con el perfil completo de %s.\n", agent.Name)
	fmt.Printf("\nNOTA: Para ejecución completa con output de modelo, usa el\n")
	fmt.Printf("   workflow ovav-delegate.js desde una sesión MiMoCode.\n")

	return 0
}

// DelegationPayload is the structured context passed to the agent
type DelegationPayload struct {
	AgentID      string            `json:"agent_id"`
	AgentName    string            `json:"agent_name"`
	AgentArea    string            `json:"agent_area"`
	AgentKind    string            `json:"agent_kind"` // lead, team, area
	Task         string            `json:"task"`
	Profile      *AgentProfile     `json:"profile,omitempty"`
	WorkspaceCtx *WorkspaceContext `json:"workspace_context"`
	Timestamp    string            `json:"timestamp"`
	SessionID    string            `json:"session_id,omitempty"`
}

// AgentProfile is the loaded agent profile content
type AgentProfile struct {
	SystemPrompt string   `json:"system_prompt"`
	Skills       []string `json:"skills"`
	Area         string   `json:"area"`
	Lead         string   `json:"lead,omitempty"`
}

// WorkspaceContext is the current workspace state
type WorkspaceContext struct {
	RepoRoot       string   `json:"repo_root"`
	Branch         string   `json:"branch"`
	Head           string   `json:"head"`
	HeadAge        string   `json:"head_age"`
	ModifiedFiles  []string `json:"modified_files"`
	StagedFiles    []string `json:"staged_files"`
	Worktree       string   `json:"worktree,omitempty"`
	UntrackedFiles []string `json:"untracked_files"`
}

func buildWorkspaceContext(repoRoot string) *WorkspaceContext {
	ctx := &WorkspaceContext{
		RepoRoot: repoRoot,
	}

	// Git branch
	if out, err := runGit(repoRoot, "branch", "--show-current"); err == nil {
		ctx.Branch = strings.TrimSpace(string(out))
	}

	// Git HEAD
	if out, err := runGit(repoRoot, "log", "-1", "--format=%h %s"); err == nil {
		ctx.Head = strings.TrimSpace(string(out))
	}

	// Modified files
	if out, err := runGit(repoRoot, "diff", "--name-only"); err == nil {
		for _, f := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if f != "" {
				ctx.ModifiedFiles = append(ctx.ModifiedFiles, f)
			}
		}
	}

	// Staged files
	if out, err := runGit(repoRoot, "diff", "--cached", "--name-only"); err == nil {
		for _, f := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if f != "" {
				ctx.StagedFiles = append(ctx.StagedFiles, f)
			}
		}
	}

	// Untracked files
	if out, err := runGit(repoRoot, "ls-files", "--others", "--exclude-standard"); err == nil {
		for _, f := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if f != "" {
				ctx.UntrackedFiles = append(ctx.UntrackedFiles, f)
			}
		}
	}

	return ctx
}

func loadAgentProfile(repoRoot string, agent subagent.Agent) (*AgentProfile, error) {
	// FilePath from catalog entry is the canonical path to the agent profile
	if agent.FilePath != "" {
		// agent.FilePath is relative to repo root
		fullPath := filepath.Join(repoRoot, agent.FilePath)
		data, err := os.ReadFile(fullPath)
		if err == nil {
			content := string(data)
			profile := &AgentProfile{
				SystemPrompt: content,
				Skills:       agent.Keywords, // use keywords as skills
				Area:         agent.Area,
				Lead:         ptrStr(agent.ID), // self-reference
			}
			return profile, nil
		}
	}

	// Fallback: try common paths
	paths := []string{
		filepath.Join(repoRoot, agent.FilePath), // already absolute or relative
		filepath.Join(repoRoot, "runtimes", "opencode", "agents", agent.ID+".md"),
	}

	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err == nil {
			content := string(data)
			profile := &AgentProfile{
				SystemPrompt: content,
				Skills:       agent.Keywords,
				Area:         agent.Area,
				Lead:         ptrStr(agent.ID),
			}
			return profile, nil
		}
	}

	return nil, fmt.Errorf("profile not found for %s", agent.ID)
}

func ptrStr(s string) string {
	return s
}

func runGit(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	return cmd.Output()
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func printDelegateHelp() {
	fmt.Print(`ovav delegate — Native OVAV Delegation

Usage:
  ovav delegate <agent_id> <task_text>
  ovav delegate --agent <agent_id> --task <task_text>
  ovav delegate <agent_id> --json

Examples:
  ovav delegate lead-eidren "Investigar A2A mesh runtime"
  ovav delegate team-clara "Coverage sprint en validators/"
  ovav delegate --agent lead-thavren --task "Audit dead code" --json

Agents (leads):
  lead-eidren, lead-thavren, lead-elena, lead-dante, lead-sofia,
  lead-uriel, lead-renata, lead-camila, lead-kenji, lead-valeria

Agents (teams):
  team-clara, team-andres, team-marco, team-helena, team-irene,
  team-diana, team-pablo, team-oscar, team-nora, team-nadia, team-mia

Flags:
  --json       Output JSON with payload file path
  --agent, -a  Agent ID (lead-* or team-*)
  --task, -t   Task description
  --help, -h   Show this help

Note:
  This command prepares a structured delegation payload with the full
  agent profile and workspace context. For complete execution with
  model output, use the ovav-delegate workflow from a MiMoCode
  session with workflow engine active.
`)
}
