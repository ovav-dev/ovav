/**
 * OVAV MEMORY v3 — F4: Security Gate Validator
 *
 * Security validators:
 * - No plaintext secrets in tracked files
 * - No exfiltration patterns
 * - Supply chain hygiene
 */
export interface ValidationResult {
    valid: boolean;
    errors: string[];
    warnings: string[];
}
export declare function validateSecurity(basePath: string): Promise<ValidationResult>;
