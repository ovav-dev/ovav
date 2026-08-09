/**
 * OVAV Security Plugin — MiMo Code governance hooks.
 *
 * Enforces OVAV permission authority via MiMo Code plugin hooks:
 *   - tool.execute.before: Block dangerous bash commands (protected_denies)
 *   - actor.preStop: Audit subagent lifecycle events
 *   - session.pre: Workspace safety validation
 *   - permission.ask: Override default permissions with OVAV policy
 *
 * Reads from .ovav/policy/permission_authority.json (canonical source).
 *
 * Capabilities:
 *   1. Security Gate — blocks git push, sudo, force operations, package installs
 *   2. Actor Audit — logs subagent completion/failure for governance trail
 *   3. Session Guard — validates workspace integrity at session start
 *   4. Permission Override — enforces OVAV deny-by-default pattern
 */

import { existsSync, readFileSync, appendFileSync, mkdirSync } from "node:fs";
import { join, dirname } from "node:path";

// ── Policy Loader ──────────────────────────────────────────────────────────────

function loadPermissionPolicy(worktree) {
  const policyPath = join(worktree, ".ovav", "policy", "permission_authority.json");
  if (!existsSync(policyPath)) return null;
  try {
    const content = readFileSync(policyPath, "utf8");
    if (content.length > 524_288) return null; // 512KB guard
    return JSON.parse(content);
  } catch {
    return null;
  }
}

// ── Bash Command Blocklist (from permission_authority.json protected_denies) ───

const DANGEROUS_PATTERNS = [
  // Git destructive operations
  { pattern: /^git\s+push\s+--force/i, reason: "force_push_blocked" },
  { pattern: /^git\s+push\s+-f/i, reason: "force_push_blocked" },
  { pattern: /^git\s+push\s+(?!.*--force)/i, reason: "raw_push_blocked_use_governed_gate" },
  { pattern: /^git\s+branch\s+-D\s/i, reason: "force_branch_delete_blocked" },
  { pattern: /^git\s+reset\s+--hard/i, reason: "hard_reset_blocked" },
  { pattern: /^git\s+clean\s+-fd/i, reason: "git_clean_blocked" },

  // Privilege escalation
  { pattern: /^sudo\s/i, reason: "sudo_blocked" },
  { pattern: /^su\s/i, reason: "su_blocked" },

  // Package management
  { pattern: /^npm\s+install\s(?!.*--save-dev)/i, reason: "npm_install_blocked_use_governed_install" },
  { pattern: /^pip\s+install\s/i, reason: "pip_install_blocked" },
  { pattern: /^apt\s+install\s/i, reason: "apt_install_blocked" },
  { pattern: /^brew\s+install\s/i, reason: "brew_install_blocked" },

  // Auth exposure
  { pattern: /^gh\s+auth\s+token/i, reason: "auth_token_exposure_blocked" },
  { pattern: /^gh\s+auth\s+login/i, reason: "auth_login_blocked" },

  // OVAV internal
  { pattern: /^python3\s+tools\/install\//i, reason: "ovav_install_blocked_use_governed_pipeline" },
  { pattern: /^python3\s+tools\/protocols\//i, reason: "ovav_protocols_blocked" },

  // Destructive file operations
  { pattern: /^rm\s+-rf\s+\//i, reason: "recursive_root_delete_blocked" },
  { pattern: /^rm\s+-rf\s+~/i, reason: "recursive_home_delete_blocked" },
];

// ── Audit Logger ───────────────────────────────────────────────────────────────

function auditLog(worktree, entry) {
  const logDir = join(worktree, ".ovav", "runtime", "logs");
  const logPath = join(logDir, "security_hooks.jsonl");
  try {
    if (!existsSync(logDir)) mkdirSync(logDir, { recursive: true });
    const line = JSON.stringify({
      timestamp: new Date().toISOString(),
      ...entry,
    }) + "\n";
    appendFileSync(logPath, line, "utf8");
  } catch {
    // Audit log failure is non-fatal
  }
}

// ── Plugin ─────────────────────────────────────────────────────────────────────

export const OvavSecurity = async ({ client, directory, worktree }) => {
  const wt = worktree || directory;
  if (!wt) {
    throw new Error("ovav-security: worktree and directory are both undefined");
  }

  // Load policy once at init
  const policy = loadPermissionPolicy(wt);
  const protectedDenies = policy?.protected_denies?.bash || [];

  const toast = (message, variant = "warning", duration = 5000) => {
    try {
      if (client?.tui?.toast?.show) {
        client.tui.toast.show({ message, variant, duration });
      }
    } catch {}
  };

  return {
    // ── Hook: tool.execute.before ────────────────────────────────────────────
    // Security Gate: block dangerous bash commands before execution
    "tool.execute.before": async (input, output) => {
      const { tool, sessionID } = input;

      // Only gate bash/shell tools
      if (tool !== "bash" && tool !== "shell") return;

      const cmd = output.args?.command || output.args?.cmd || "";
      if (!cmd) return;

      // Check against DANGEROUS_PATTERNS
      for (const { pattern, reason } of DANGEROUS_PATTERNS) {
        if (pattern.test(cmd)) {
          auditLog(wt, {
            event: "command_blocked",
            tool,
            command: cmd.substring(0, 200),
            reason,
            sessionID,
          });
          toast(`🚫 OVAV bloqueó: ${reason}`, "error", 6000);
          output.cancel = true;
          output.cancelReason = `OVAV Security Gate: ${reason}. Use governed pathways.`;
          return;
        }
      }

      // Check against policy protected_denies (dynamic from JSON)
      for (const denyPattern of protectedDenies) {
        if (typeof denyPattern === "string" && cmd.match(new RegExp(denyPattern.replace(/\*/g, ".*"), "i"))) {
          auditLog(wt, {
            event: "command_blocked_policy",
            tool,
            command: cmd.substring(0, 200),
            reason: "policy_protected_deny",
            pattern: denyPattern,
            sessionID,
          });
          toast(`🚫 OVAV policy bloqueó: ${denyPattern}`, "error", 6000);
          output.cancel = true;
          output.cancelReason = `OVAV Policy denied: pattern "${denyPattern}"`;
          return;
        }
      }
    },

    // ── Hook: actor.preStop ──────────────────────────────────────────────────
    // Actor Audit: lightweight pre-delivery log (no outcome/canWrite here)
    "actor.preStop": async (input, output) => {
      const { actorID, agentType, mode, task, iteration, sessionID } = input;

      auditLog(wt, {
        event: "actor_preStop",
        actorID,
        agentType,
        mode,
        task: task?.substring(0, 300),
        iteration,
        sessionID,
      });
    },

    // ── Hook: actor.postStop ────────────────────────────────────────────────
    // Actor Audit: outcome + canWrite logic belongs here (ActorPostStopInput)
    "actor.postStop": async (input, output) => {
      const { actorID, agentType, mode, task, outcome, iteration, sessionID, canWrite } = input;

      auditLog(wt, {
        event: "actor_postStop",
        actorID,
        agentType,
        mode,
        task: task?.substring(0, 300),
        outcome,
        iteration,
        sessionID,
        canWrite,
      });

      // Block write-capable actors from silent failure — retry before delivery
      if (canWrite && outcome === "failure") {
        output.continue = true;
        output.reason = "OVAV: write-capable actor failed — retrying before delivery";
        toast(`⚠️ Actor ${agentType} falló con write access — retry`, "warning");
      }
    },

    // ── Hook: session.pre ────────────────────────────────────────────────────
    // Session Guard: validate workspace at session start
    "session.pre": async (input, output) => {
      const { sessionID, agentID } = input;

      // Check protected branch
      const branchCheck = existsSync(join(wt, ".ovav", "runtime", "protected_branch_waiver.yaml"));
      auditLog(wt, {
        event: "session_pre",
        sessionID,
        agentID,
        has_waiver: branchCheck,
      });

      // Validate critical governance files exist
      const governanceFiles = [
        ".ovav/policy/permission_authority.json",
        ".ovav/plan/caps.yaml",
        ".ovav/laws/area_boundary_enforcement.yaml",
      ];

      const missing = governanceFiles.filter(f => !existsSync(join(wt, f)));
      if (missing.length > 0) {
        auditLog(wt, {
          event: "governance_files_missing",
          sessionID,
          missing,
        });
        // Non-blocking warning — session can proceed but governance is degraded
        toast(`⚠️ Archivos de gobernanza faltantes: ${missing.length}`, "warning");
      }
    },

    // ── Hook: permission.ask ─────────────────────────────────────────────────
    // Permission Override: enforce OVAV deny-by-default
    "permission.ask": async (input, output) => {
      const { permission, patterns } = input;

      // Auto-deny known dangerous patterns
      if (permission === "bash" || permission === "shell") {
        for (const pattern of patterns || []) {
          for (const { pattern: denyPattern, reason } of DANGEROUS_PATTERNS) {
            if (denyPattern.test(pattern)) {
              auditLog(wt, {
                event: "permission_denied",
                permission,
                pattern,
                reason,
              });
              output.status = "deny";
              return;
            }
          }
        }
      }

      // Auto-deny external directory access outside allowlist
      if (permission === "external_directory") {
        const allowList = policy?.protected_denies?.external_directory || {};
        for (const pattern of patterns || []) {
          const allowRule = allowList[pattern];
          if (allowRule === "deny" || (!allowList[pattern] && allowList["*"] === "deny")) {
            auditLog(wt, {
              event: "external_directory_denied",
              pattern,
            });
            output.status = "deny";
            return;
          }
        }
      }
    },
  };
};
