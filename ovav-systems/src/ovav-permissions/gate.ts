/**
 * OVAV MEMORY v3 — Permission Gates
 *
 * Integrates with pi.dev's project_trust hook to enforce OVAV-native
 * permission model. Based on the MiMoCode security plugin pattern
 * but adapted for OVAV's allowlist approach.
 */

export interface PermissionDecision {
  allowed: boolean;
  reason: string;
  gate: "ovav";
}

// DANGEROUS_PATTERNS: commands that are always blocked by OVAV
const DANGEROUS_PATTERNS: Array<{ pattern: RegExp; reason: string }> = [
  // Git destructive
  { pattern: /^git\s+push\s+--force/i, reason: "force_push_blocked" },
  { pattern: /^git\s+push\s+-f/i, reason: "force_push_blocked" },
  { pattern: /^git\s+push\s+(?!.*--force)/i, reason: "raw_git_push_blocked_use_governed_gate" },
  { pattern: /^git\s+branch\s+-D\s/i, reason: "force_branch_delete_blocked" },
  { pattern: /^git\s+reset\s+--hard/i, reason: "hard_reset_blocked" },
  { pattern: /^git\s+clean\s+-fd/i, reason: "git_clean_blocked" },

  // Privilege escalation
  { pattern: /^sudo\s/i, reason: "sudo_blocked" },
  { pattern: /^su\s/i, reason: "su_blocked" },

  // Package management
  { pattern: /^pip\s+install/i, reason: "pip_install_blocked" },
  { pattern: /^apt\s+install/i, reason: "apt_install_blocked" },
  { pattern: /^npm\s+install\s(?!.*--save-dev)/i, reason: "npm_install_blocked" },
  { pattern: /^brew\s+install/i, reason: "brew_install_blocked" },

  // Auth exposure
  { pattern: /^gh\s+auth\s+token/i, reason: "auth_token_exposure_blocked" },
  { pattern: /^gh\s+auth\s+login/i, reason: "auth_login_blocked" },

  // Destructive
  { pattern: /^rm\s+-rf\s+\//i, reason: "recursive_root_delete_blocked" },
  { pattern: /^rm\s+-rf\s+~/i, reason: "recursive_home_delete_blocked" },
];

export class PermissionGate {
  private worktree: string;

  constructor(worktree: string) {
    this.worktree = worktree;
  }

  /**
   * Evaluate a bash/shell command against OVAV security policy.
   * Returns PermissionDecision with allowed=false if blocked.
   */
  evaluateCommand(command: string): PermissionDecision {
    const cmd = command.trim().substring(0, 300);

    for (const { pattern, reason } of DANGEROUS_PATTERNS) {
      if (pattern.test(cmd)) {
        return {
          allowed: false,
          reason: `OVAV Permission Gate: ${reason}. Command: "${cmd.substring(0, 80)}..."`,
          gate: "ovav",
        };
      }
    }

    return { allowed: true, reason: "allowed", gate: "ovav" };
  }

  /**
   * Evaluate external directory access against OVAV allowlist.
   */
  evaluateExternalDirectory(requestedPath: string): PermissionDecision {
    // Default deny for external directories unless explicitly allowed
    const normalized = requestedPath.replace(/\/$/, "");

    // Allowlist: safe directories that are always OK
    const safePaths = [
      "/tmp",
      "/dev/null",
      "/dev/stdout",
      "/dev/stderr",
    ];

    for (const safe of safePaths) {
      if (normalized === safe || normalized.startsWith(safe + "/")) {
        return { allowed: true, reason: "safe_path", gate: "ovav" };
      }
    }

    // Block home directory access unless under known safe subpaths
    const home = process.env.HOME ?? "";
    if (normalized.startsWith(home)) {
      const safeHomeSubpaths = ["/.cache", "/.local/share", "/.config"];
      for (const subpath of safeHomeSubpaths) {
        if (normalized.startsWith(home + subpath)) {
          return { allowed: true, reason: "safe_home_subpath", gate: "ovav" };
        }
      }
    }

    return {
      allowed: false,
      reason: `External directory access denied: ${requestedPath}. Use governed pathways.`,
      gate: "ovav",
    };
  }

  /**
   * Evaluate environment variable access.
   */
  evaluateEnvAccess(varName: string): PermissionDecision {
    // Block dangerous env vars
    const blocked = [
      /AWS_SECRET/i,
      /GITHUB_TOKEN/i,
      /PRIVATE_KEY/i,
      /SECRET/i,
      /PASSWORD/i,
      /API_KEY/i,
    ];

    for (const pattern of blocked) {
      if (pattern.test(varName)) {
        return {
          allowed: false,
          reason: `OVAV: env var access to ${varName} is blocked`,
          gate: "ovav",
        };
      }
    }

    return { allowed: true, reason: "allowed", gate: "ovav" };
  }
}
