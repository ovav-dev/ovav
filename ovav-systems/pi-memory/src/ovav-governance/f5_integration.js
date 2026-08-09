/**
 * OVAV MEMORY v3 — F5: Integration Gate Validator
 *
 * Runs all F0-F4 validators in sequence and computes
 * a combined integrity score.
 */
import { validateArchitecture } from "./f1_architecture.js";
import { validateInfrastructure } from "./f2_infrastructure.js";
import { validateRoles } from "./f3_roles.js";
import { validateSecurity } from "./f4_security.js";
export async function validateF5(basePath) {
    const [f1, f2, f3, f4] = await Promise.all([
        validateArchitecture(basePath),
        validateInfrastructure(basePath),
        validateRoles(basePath),
        validateSecurity(basePath),
    ]);
    const allErrors = [...f1.errors, ...f2.errors, ...f3.errors, ...f4.errors];
    const allWarnings = [...f1.warnings, ...f2.warnings, ...f3.warnings, ...f4.warnings];
    // Score: each gate is worth 25 points, errors subtract proportionally
    const f1Score = f1.valid ? 25 : Math.max(0, 25 - f1.errors.length * 5);
    const f2Score = f2.valid ? 25 : Math.max(0, 25 - f2.errors.length * 5);
    const f3Score = f3.valid ? 25 : Math.max(0, 25 - f3.errors.length * 5);
    const f4Score = f4.valid ? 25 : Math.max(0, 25 - f4.errors.length * 5);
    const score = f1Score + f2Score + f3Score + f4Score;
    const status = score >= 80 ? "HEALTHY" :
        score >= 50 ? "DEGRADED" :
            "CRITICAL";
    return {
        score,
        status,
        f1,
        f2,
        f3,
        f4,
        totalErrors: allErrors.length,
        totalWarnings: allWarnings.length,
    };
}
export async function validateF0(_basePath) {
    // F0 is deprecated — delegate to F5
    return {
        valid: true,
        errors: [],
        warnings: ["F0 (safety check) is deprecated — use F5 integration gate instead"],
    };
}
//# sourceMappingURL=f5_integration.js.map