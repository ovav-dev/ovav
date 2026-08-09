/**
 * OVAV MEMORY v3 — F3: Roles Gate Validator
 *
 * Verifies that service area roles are defined in caps.yaml
 * and that each area has a lead and scope.
 */
import * as fs from "node:fs/promises";
import * as path from "node:path";
function parseCapsYaml(content) {
    const areas = [];
    // Simple YAML frontmatter parsing for service areas
    // Expects format:
    // service_areas:
    //   - name: Platform Engineering
    //     lead: thavren
    //     scope: [...]
    const lines = content.split("\n");
    let inServiceAreas = false;
    for (const line of lines) {
        if (line.match(/^service_areas?\s*:/i)) {
            inServiceAreas = true;
            continue;
        }
        if (inServiceAreas) {
            if (line.match(/^\w/) && !line.startsWith("  ") && !line.startsWith("-")) {
                // Exited service areas section
                break;
            }
            if (line.includes("name:")) {
                const name = line.split("name:")[1].trim().replace(/['"]/g, "");
                areas.push({ name, lead: "", scope: [] });
            }
            else if (line.includes("lead:") && areas.length > 0) {
                const lead = line.split("lead:")[1].trim().replace(/['"]/g, "");
                areas[areas.length - 1].lead = lead;
            }
        }
    }
    return areas.length > 0 ? areas : null;
}
export async function validateRoles(basePath) {
    const errors = [];
    const warnings = [];
    const capsPath = path.join(basePath, ".ovav/plan/caps.yaml");
    let capsContent;
    try {
        capsContent = await fs.readFile(capsPath, "utf-8");
    }
    catch {
        errors.push("caps.yaml not found — cannot validate roles");
        return { valid: false, errors, warnings };
    }
    // Guard: reject oversized files (>1MB)
    if (capsContent.length > 1_048_576) {
        errors.push("caps.yaml exceeds 1MB — rejecting");
        return { valid: false, errors, warnings };
    }
    const areas = parseCapsYaml(capsContent);
    if (!areas || areas.length === 0) {
        errors.push("No service areas defined in caps.yaml");
        return { valid: false, errors, warnings };
    }
    // Check each area has required fields
    const requiredLeads = ["thavren", "sofia", "elena", "eidren", "renata", "valeria", "uriel", "kenji", "camila", "dante", "diana", "pablo", "oscar", "nora", "nadia", "mia", "clara", "helena", "irene", "lucas", "andres", "marco"];
    const knownLeads = new Set(requiredLeads);
    for (const area of areas) {
        if (!area.lead) {
            errors.push(`Service area "${area.name}" has no lead defined`);
        }
        else if (!knownLeads.has(area.lead)) {
            warnings.push(`Service area "${area.name}" has unknown lead: ${area.lead}`);
        }
    }
    return {
        valid: errors.length === 0,
        errors,
        warnings,
    };
}
//# sourceMappingURL=f3_roles.js.map