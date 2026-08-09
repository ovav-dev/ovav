/**
 * OVAV MEMORY v3 — F4: Security Gate Validator
 *
 * Security validators:
 * - No plaintext secrets in tracked files
 * - No exfiltration patterns
 * - Supply chain hygiene
 */
import * as fs from "node:fs/promises";
import * as path from "node:path";
// Patterns that indicate secrets or credentials
const SECRET_PATTERNS = [
    { pattern: /api[_-]?key\s*[=:]\s*['"]?[a-zA-Z0-9_-]{20,}/i, id: "api_key_literal" },
    { pattern: /secret\s*[=:]\s*['"]?[a-zA-Z0-9_-]{20,}/i, id: "secret_literal" },
    { pattern: /password\s*[=:]\s*['"]?[^\s'"]{8,}/i, id: "password_literal" },
    { pattern: /bearer\s+[a-zA-Z0-9_-]{20,}/i, id: "bearer_token" },
    { pattern: /ghp_[a-zA-Z0-9]{36,}/i, id: "github_pat" },
    { pattern: /xox[baprs]-[a-zA-Z0-9-]{10,}/i, id: "slack_token" },
    { pattern: /-----BEGIN\s+(RSA\s+)?PRIVATE\s+KEY-----/i, id: "private_key_header" },
];
// Patterns that indicate data exfiltration intent
const EXFIL_PATTERNS = [
    { pattern: /curl\s+.+\s+--data[\s-]?(?:binary|raw)?\s+['"]?\$/, id: "curl_exfil" },
    { pattern: /wget\s+.+\s+--(?:post|data)[\s-]/i, id: "wget_exfil" },
    { pattern: /nc\s+-e\s+/i, id: "netcat_reverse_shell" },
    { pattern: /exfil|exfiltrate|data\s+leak/i, id: "exfil_keyword" },
];
const DANGEROUS_EXTENSIONS = [".env", ".pem", ".key", ".p12", ".pfx", ".crt"];
export async function validateSecurity(basePath) {
    const errors = [];
    const warnings = [];
    // Check .gitignore excludes dangerous files
    const gitignorePath = path.join(basePath, ".gitignore");
    try {
        const gitignore = await fs.readFile(gitignorePath, "utf-8");
        for (const ext of DANGEROUS_EXTENSIONS) {
            if (!gitignore.includes(ext)) {
                warnings.push(`.gitignore does not exclude ${ext} files`);
            }
        }
    }
    catch {
        warnings.push(".gitignore not found");
    }
    // Check for secrets in .ovav/ directory (should never have them)
    const ovavDir = path.join(basePath, ".ovav");
    await scanForSecrets(ovavDir, errors, warnings);
    // Check go-runtime/cmd for dangerous patterns
    const cmdDir = path.join(basePath, "go-runtime/cmd");
    try {
        await scanDirForExfil(cmdDir, errors, warnings);
    }
    catch {
        // Directory may not exist yet — non-fatal
    }
    return {
        valid: errors.length === 0,
        errors,
        warnings,
    };
}
async function scanForSecrets(dir, errors, warnings) {
    let entries;
    try {
        entries = await fs.readdir(dir, { withFileTypes: true });
    }
    catch {
        return;
    }
    for (const entry of entries) {
        const fullPath = path.join(dir, entry.name);
        if (entry.isDirectory()) {
            await scanForSecrets(fullPath, errors, warnings);
        }
        else if (entry.isFile()) {
            const ext = path.extname(entry.name);
            if (DANGEROUS_EXTENSIONS.includes(ext)) {
                errors.push(`Dangerous file in .ovav/: ${path.relative(dir, fullPath)}`);
                continue;
            }
            try {
                const stat = await fs.stat(fullPath);
                if (stat.size > 524_288)
                    continue; // Skip >512KB
                const content = await fs.readFile(fullPath, "utf-8");
                for (const { pattern, id } of SECRET_PATTERNS) {
                    if (pattern.test(content)) {
                        errors.push(`Secret pattern detected (${id}): ${path.relative(dir, fullPath)}`);
                    }
                }
            }
            catch {
                // Skip unreadable files
            }
        }
    }
}
async function scanDirForExfil(dir, errors, warnings) {
    let entries;
    try {
        entries = await fs.readdir(dir, { withFileTypes: true });
    }
    catch {
        return;
    }
    for (const entry of entries) {
        const fullPath = path.join(dir, entry.name);
        if (entry.isDirectory()) {
            await scanDirForExfil(fullPath, errors, warnings);
        }
        else if (entry.isFile() && entry.name.endsWith(".go")) {
            try {
                const content = await fs.readFile(fullPath, "utf-8");
                for (const { pattern, id } of EXFIL_PATTERNS) {
                    if (pattern.test(content)) {
                        errors.push(`Exfil pattern detected (${id}): ${fullPath}`);
                    }
                }
            }
            catch {
                // Skip unreadable
            }
        }
    }
}
//# sourceMappingURL=f4_security.js.map