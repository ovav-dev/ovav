/**
 * OVAV MEMORY v3 — Governance Tool Registry
 *
 * Exposes F0-F5 validators as pi.dev tools with the same
 * naming pattern as the existing MiMoCode plugin:
 *   - ovav_validate      → runs F0-F5 validators
 *   - ovav_daily        → state summary
 *   - ovav_next_work    → plan resolution
 *   - ovav_check_integrity → system integrity score
 */

import { validateF0, validateF5, type F5Report } from "./f5_integration.js";
import { validateArchitecture } from "./f1_architecture.js";
import { validateInfrastructure } from "./f2_infrastructure.js";
import { validateRoles } from "./f3_roles.js";
import { validateSecurity } from "./f4_security.js";

export interface ToolResult {
  source: string;
  output: unknown;
  error?: string;
}

// ── Tool Implementations ───────────────────────────────────────────────────────

export async function runOvavValidate(
  basePath: string,
  scope: "all" | "f0" | "f1" | "f2" | "f3" | "f4" | "f5" = "all"
): Promise<ToolResult> {
  try {
    switch (scope) {
      case "f0":
        return { source: "f0", output: await validateF0(basePath) };
      case "f1":
        return { source: "f1", output: await validateArchitecture(basePath) };
      case "f2":
        return { source: "f2", output: await validateInfrastructure(basePath) };
      case "f3":
        return { source: "f3", output: await validateRoles(basePath) };
      case "f4":
        return { source: "f4", output: await validateSecurity(basePath) };
      case "f5":
      case "all":
      default: {
        const report: F5Report = await validateF5(basePath);
        return {
          source: "f5",
          output: {
            message: "OVAV validators complete",
            score: report.score,
            status: report.status,
            errors: report.totalErrors,
            warnings: report.totalWarnings,
            detail: report,
          },
        };
      }
    }
  } catch (err) {
    return {
      source: "error",
      output: null,
      error: err instanceof Error ? err.message : "unknown error",
    };
  }
}

export async function runOvavDaily(basePath: string): Promise<ToolResult> {
  try {
    const f1 = await validateArchitecture(basePath);
    const f2 = await validateInfrastructure(basePath);

    return {
      source: "ovav_daily",
      output: {
        timestamp: new Date().toISOString(),
        f1_valid: f1.valid,
        f2_valid: f2.valid,
        f1_errors: f1.errors.length,
        f2_errors: f2.errors.length,
      },
    };
  } catch (err) {
    return {
      source: "error",
      output: null,
      error: err instanceof Error ? err.message : "unknown error",
    };
  }
}

export async function runOvavNextWork(basePath: string): Promise<ToolResult> {
  try {
    const f1 = await validateArchitecture(basePath);
    const f3 = await validateRoles(basePath);

    const recommendations: string[] = [];
    if (!f1.valid) recommendations.push(`Fix F1 (Architecture): ${f1.errors[0] ?? "unknown"}`);
    if (!f3.valid) recommendations.push(`Fix F3 (Roles): ${f3.errors[0] ?? "unknown"}`);
    if (f1.valid && f3.valid) recommendations.push("All critical gates pass — system healthy");

    return {
      source: "ovav_next_work",
      output: {
        recommendations,
        f1_valid: f1.valid,
        f3_valid: f3.valid,
      },
    };
  } catch (err) {
    return {
      source: "error",
      output: null,
      error: err instanceof Error ? err.message : "unknown error",
    };
  }
}

export async function runOvavCheckIntegrity(basePath: string): Promise<ToolResult> {
  try {
    const report = await validateF5(basePath);

    return {
      source: "ovav_check_integrity",
      output: {
        score: `${report.score}%`,
        status: report.status,
        f1_pass: report.f1.valid,
        f2_pass: report.f2.valid,
        f3_pass: report.f3.valid,
        f4_pass: report.f4.valid,
        total_errors: report.totalErrors,
        total_warnings: report.totalWarnings,
      },
    };
  } catch (err) {
    return {
      source: "error",
      output: null,
      error: err instanceof Error ? err.message : "unknown error",
    };
  }
}
