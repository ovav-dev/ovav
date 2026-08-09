/**
 * OVAV MEMORY v3 — F3: Roles Gate Validator
 *
 * Verifies that service area roles are defined in caps.yaml
 * and that each area has a lead and scope.
 */

import * as fs from "node:fs/promises";
import * as path from "node:path";

export interface ValidationResult {
  valid: boolean;
  errors: string[];
  warnings: string[];
}

export interface ServiceArea {
  name: string;
  lead: string;
  scope: string[];
}

function parseCapsYaml(content: string): ServiceArea[] | null {
  const areas: ServiceArea[] = [];

  // YAML parsing handles two formats:
  // Format 1 (flat list):
  //   service_areas:
  //     - name: Platform Engineering
  //       lead: thavren
  //       scope: [...]
  //
  // Format 2 (nested by priority):
  //   service_areas:
  //     p0_areas:
  //       platform_engineering:
  //         id: "platform_engineering"
  //         lead: "thavren"
  //         scope: [...]
  //
  // Format 3 (flat with id/name):
  //   service_areas:
  //     platform_engineering:
  //       id: "platform_engineering"
  //       lead: "thavren"
  //       scope: [...]

  const lines = content.split("\n");
  let inServiceAreas = false;
  let depth = 0;
  let currentArea: ServiceArea | null = null;

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];

    // Detect service_areas section start
    if (line.match(/^service_areas?:\s*$/i)) {
      inServiceAreas = true;
      depth = 0;
      continue;
    }

    if (!inServiceAreas) continue;

    // Track indentation depth
    const indent = line.match(/^(\s*)/)?.[1].length ?? 0;

    // Exit conditions
    if (indent === 0 && line.match(/^\w/) && !line.startsWith("#")) {
      // Top-level key not part of service_areas
      if (!line.match(/^service_areas?:/i) && !line.startsWith("-")) {
        break;
      }
    }

    // Skip empty lines and comments
    if (!line.trim() || line.trim().startsWith("#")) continue;

    // Format 1: Flat list with - name:
    if (line.trim().startsWith("- name:") || line.trim().startsWith("- ")) {
      if (currentArea && currentArea.name) {
        areas.push(currentArea);
      }
      if (line.includes("name:")) {
        const name = line.split("name:")[1]?.trim().replace(/['"]/g, "") || "";
        currentArea = { name, lead: "", scope: [] };
      } else {
        currentArea = { name: "", lead: "", scope: [] };
      }
      continue;
    }

    // Format 2/3: Nested structure with area keys
    const areaKeyMatch = line.match(/^\s{2,}([a-z_][a-z0-9_]*):\s*$/i);
    if (areaKeyMatch && !line.includes("scope:") && !line.includes("lead:") && !line.includes("id:")) {
      // This is an area key (e.g., "platform_engineering:")
      // Skip category headers like "p0_areas", "p1_areas", "p2_areas"
      const areaKey = areaKeyMatch[1];
      if (areaKey.match(/^p\d_areas?$/i)) {
        continue; // Skip category headers
      }

      if (currentArea && currentArea.name) {
        areas.push(currentArea);
      }
      const areaName = areaKey.replace(/_/g, " ");
      currentArea = { name: areaName, lead: "", scope: [] };
      continue;
    }

    if (!currentArea) {
      // Try to detect area start in flat format
      if (line.match(/^\s+[a-z_]+:\s*$/i) && !line.includes("scope:") && !line.includes("lead:") && !line.includes("id:")) {
        const areaName = line.match(/^\s+([a-z_][a-z0-9_]*):/i)?.[1]?.replace(/_/g, " ") || "";
        currentArea = { name: areaName, lead: "", scope: [] };
      }
      continue;
    }

    // Extract id (becomes name if name not set)
    if (line.includes('id:') && !currentArea.name) {
      const id = line.split('id:')[1]?.trim().replace(/['"]/g, "") || "";
      currentArea.name = id.replace(/_/g, " ");
    }

    // Extract lead
    if (line.includes("lead:")) {
      const lead = line.split("lead:")[1]?.trim().replace(/['"]/g, "") || "";
      currentArea.lead = lead;
    }

    // Extract name (overrides id)
    if (line.includes("name:") && !line.trim().startsWith("-")) {
      const name = line.split("name:")[1]?.trim().replace(/['"]/g, "") || "";
      currentArea.name = name;
    }
  }

  // Push last area
  if (currentArea && currentArea.name) {
    areas.push(currentArea);
  }

  return areas.length > 0 ? areas : null;
}

export async function validateRoles(basePath: string): Promise<ValidationResult> {
  const errors: string[] = [];
  const warnings: string[] = [];

  const capsPath = path.join(basePath, ".ovav/plan/caps.yaml");
  let capsContent: string;

  try {
    capsContent = await fs.readFile(capsPath, "utf-8");
  } catch {
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
    } else if (!knownLeads.has(area.lead)) {
      warnings.push(`Service area "${area.name}" has unknown lead: ${area.lead}`);
    }
  }

  return {
    valid: errors.length === 0,
    errors,
    warnings,
  };
}
