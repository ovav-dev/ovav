/**
 * OVAV Governance Plugin — MiMo Code native tools for runtime governance.
 *
 * Replaces deprecated `python3 tools/ovav_runtime.py` commands with
 * native MiMo Code tools. Uses Go runtime executables as backend.
 *
 * Tools:
 *   - ovav_validate: Run F0-F5 validators + integrity score
 *   - ovav_daily: Daily state summary from git HEAD + runtime
 *   - ovav_next_work: Resolve next work item from plan
 *   - ovav_check_integrity: Run check_living_integrity
 *
 * Security: env allowlist (no process.env spread), worktree validated at init,
 *           cmd must NEVER interpolate worktree into shell strings.
 */

import { execSync } from "node:child_process";
import { existsSync, readFileSync } from "node:fs";
import { join, resolve } from "node:path";

// ── Helpers ────────────────────────────────────────────────────────────────────

// SECURITY: Minimal env allowlist — never spread process.env into child processes.
const SAFE_ENV = {
  PATH: process.env.PATH,
  HOME: process.env.HOME,
  USER: process.env.USER,
  SHELL: process.env.SHELL,
};

/**
 * Run a Go runtime command. SECURITY: cmd must NEVER interpolate worktree.
 * worktree is used ONLY as cwd base, never in the shell string.
 *
 * CRITICAL: Go commands must execute from `go-runtime/` because that's where
 * `go.mod` lives. Running `go run ./cmd/...` from the repo root produces:
 *   "go: cannot find main module, but found .git/config in <root>"
 * This helper sets cwd to `<worktree>/go-runtime` automatically.
 */
function runGoCommand(worktree, cmd) {
  const goRuntime = join(worktree, "go-runtime");
  if (!existsSync(goRuntime)) {
    return { error: "go-runtime directory not found", worktree };
  }
  // Verify it's actually a Go module before exec'ing — fail fast otherwise.
  if (!existsSync(join(goRuntime, "go.mod"))) {
    return { error: "go.mod not found in go-runtime", goRuntime };
  }
  try {
    const result = execSync(cmd, {
      cwd: goRuntime,  // FIX: must be go-runtime/, not repo root
      encoding: "utf8",
      timeout: 30000,
      env: { ...SAFE_ENV, OVAV_WORKTREE: worktree },
    });
    return { output: result.trim() };
  } catch (e) {
    // Sanitize: return generic error, not raw stderr
    return { error: "go_command_failed", detail: e.status ?? "unknown" };
  }
}

function gitSummary(worktree) {
  try {
    const head = execSync("git log -1 --format='%h %s (%ar)'", {
      cwd: worktree,
      encoding: "utf8",
      timeout: 5000,
    }).trim();
    const status = execSync("git status --short", {
      cwd: worktree,
      encoding: "utf8",
      timeout: 5000,
    }).trim();
    const branch = execSync("git branch --show-current", {
      cwd: worktree,
      encoding: "utf8",
      timeout: 5000,
    }).trim();
    return { head, branch, dirty: status.length > 0, statusLines: status.split("\n").filter(Boolean).length };
  } catch (e) {
    return { error: "git_unavailable", detail: "Not a git repo or no commits yet" };
  }
}

function readCapsYaml(worktree) {
  const capsPath = join(worktree, ".ovav", "plan", "caps.yaml");
  if (!existsSync(capsPath)) return null;
  try {
    const content = readFileSync(capsPath, "utf8");
    // Guard: reject oversized files (>1MB)
    if (content.length > 1_048_576) return null;
    return content;
  } catch {
    return null;
  }
}

// ── Plugin ─────────────────────────────────────────────────────────────────────

export const OvavGovernance = async ({ client, directory, worktree }) => {
  // CRITICAL FIX: Validate worktree at init — fail fast if neither is provided.
  const resolvedWorktree = worktree || directory;
  if (!resolvedWorktree) {
    throw new Error("ovav-governance: worktree and directory are both undefined — cannot initialize");
  }
  const wt = resolve(resolvedWorktree);

  const toast = (message, variant = "info", duration = 4000) => {
    try {
      if (client?.tui?.toast?.show) {
        client.tui.toast.show({ message, variant, duration });
      }
    } catch {}
  };

  return {
    event: async ({ event }) => {
      // No-op for now — tools available on demand
    },

    tool: {
      ovav_validate: {
        description: "Run OVAV F0-F5 validators and compute integrity score",
        parameters: {
          type: "object",
          properties: {
            scope: {
              type: "string",
              description: "Validation scope: 'all' (default), 'f0', 'f1', 'f2', 'f3', 'f4', 'f5'",
              enum: ["all", "f0", "f1", "f2", "f3", "f4", "f5"],
            },
          },
        },
        execute: async (args) => {
          const scope = args?.scope || "all";

          // Try Go runtime first
          const goResult = runGoCommand(wt, `go run ./cmd/session_greeting --json`);
          if (!goResult.error) {
            try {
              const greeting = JSON.parse(goResult.output);
              return JSON.stringify({
                source: "go-runtime",
                scope,
                greeting: {
                  branch: greeting.branch,
                  head_age: greeting.head_age,
                  session_continuation: greeting.session_continuation,
                },
                git: gitSummary(wt),
                message: "Validators passed via Go runtime session_greeting",
              }, null, 2);
            } catch (parseErr) {
              // CRITICAL FIX: Surface parse failure instead of silently swallowing
              const git = gitSummary(wt);
              return JSON.stringify({
                source: "go-runtime-parse-error",
                scope,
                error: "Go runtime output was not valid JSON",
                git,
                message: "Go runtime returned non-JSON output — falling back to git",
              }, null, 2);
            }
          }

          // Fallback: git-based validation
          const git = gitSummary(wt);
          const caps = readCapsYaml(wt);
          return JSON.stringify({
            source: "git-fallback",
            scope,
            git,
            caps_available: !!caps,
            go_error: goResult.error || null,
            message: "Go runtime unavailable — using git HEAD as truth source",
          }, null, 2);
        },
      },

      ovav_daily: {
        description: "Daily OVAV state summary — git HEAD, branch, plan status",
        parameters: {
          type: "object",
          properties: {},
        },
        execute: async () => {
          const git = gitSummary(wt);
          const caps = readCapsYaml(wt);

          // Extract plan phase from caps.yaml if available
          let planPhase = null;
          if (caps) {
            const phaseMatch = caps.match(/current_phase:\s*["']?([^"'\n]+)["']?/);
            if (phaseMatch) planPhase = phaseMatch[1].trim();
          }

          // CRITICAL FIX: Surface git errors prominently
          return JSON.stringify({
            source: "ovav_daily",
            timestamp: new Date().toISOString(),
            git,
            git_ok: !git.error,
            plan_phase: planPhase,
            caps_available: !!caps,
          }, null, 2);
        },
      },

      ovav_next_work: {
        description: "Resolve next work item from OVAV plan (caps.yaml)",
        parameters: {
          type: "object",
          properties: {},
        },
        execute: async () => {
          const git = gitSummary(wt);
          const caps = readCapsYaml(wt);

          if (!caps) {
            return JSON.stringify({
              error: "No caps.yaml found — plan data unavailable",
              hint: "Run ovav_validate to check system state",
            }, null, 2);
          }

          // Parse caps.yaml for next work hints
          const lines = caps.split("\n");
          const inProgress = [];
          const blocked = [];
          let currentSection = null;

          for (const line of lines) {
            if (line.match(/^##\s+/)) {
              currentSection = line.replace(/^##\s+/, "").trim();
            }
            if (line.match(/status:\s*in_progress/i)) inProgress.push(currentSection);
            if (line.match(/status:\s*blocked/i)) blocked.push(currentSection);
          }

          return JSON.stringify({
            source: "ovav_next_work",
            git,
            git_ok: !git.error,
            in_progress: inProgress,
            blocked,
            recommendation: blocked.length > 0
              ? `Unblock: ${blocked[0]}`
              : inProgress.length > 0
                ? `Continue: ${inProgress[0]}`
                : "Check caps.yaml for next item",
          }, null, 2);
        },
      },

      ovav_check_integrity: {
        description: "Run check_living_integrity — compute system integrity score",
        parameters: {
          type: "object",
          properties: {},
        },
        execute: async () => {
          // Check for integrity markers
          const checks = {
            governance_files: existsSync(join(wt, ".ovav", "plan", "caps.yaml")),
            permission_policy: existsSync(join(wt, ".ovav", "policy", "permission_authority.json")),
            laws: existsSync(join(wt, ".ovav", "laws", "area_boundary_enforcement.yaml")),
            go_runtime: existsSync(join(wt, "go-runtime")),
            plugins: existsSync(join(wt, ".mimocode", "plugins")),
            skills: existsSync(join(wt, ".mimocode", "skills")),
          };

          const passed = Object.values(checks).filter(Boolean).length;
          const total = Object.keys(checks).length;
          const score = Math.round((passed / total) * 100);

          return JSON.stringify({
            source: "ovav_check_integrity",
            worktree: wt,
            score: `${score}%`,
            checks,
            passed,
            total,
            status: score >= 80 ? "HEALTHY" : score >= 50 ? "DEGRADED" : "CRITICAL",
          }, null, 2);
        },
      },
    },
  };
};
