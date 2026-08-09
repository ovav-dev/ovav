/**
 * OVAV Delegate Workflow
 *
 * C5-T3 (capa 5 Agent Governance): Bug B runtime fix.
 *
 * PROBLEMA: MiMoCode's `actor` tool SOLO acepta `explore` y `general`
 * como subagent_type. LEAD names (eidren, sofia) y team IDs (team-clara,
 * team-andres) son silently downgradeados a `general` → UI muestra "GENERAL TASK".
 *
 * SOLUCION: Usar `workflow + agent()` que SÍ puede invocar cualquier
 * agente registrado via `agent({ subagent_type: "lead-eidren", ... })`.
 *
 * Este workflow recibe un task y lo delega al agente correcto usando
 * el subagent_type canónico.
 *
 * Usage:
 *   workflow("ovav-delegate", {
 *     agent_id: "lead-eidren",   // o "team-clara", "team-marco", etc.
 *     task: "Descripción del task...",
 *     context: "state" | "full" | "none",
 *     model?: "opencode-go/deepseek-v4-pro"
 *   })
 */

export const meta = {
  name: "ovav-delegate",
  description: "OVAV delegation workflow — routes tasks to LEAD/team agents via workflow+agent()",
  version: "1.0",
};

// OVAV lead + team directory (sincronizado con ovav-squad-delegation skill)
const LEAD_DIRECTORY = {
  thavren:  { type: "lead", area: "platform_engineering", squad: ["team-andres","team-lucas","team-helena","team-irene","team-diana","team-pablo","team-oscar","team-nora","team-nadia","team-mia","team-clara","team-marco"] },
  elena:    { type: "lead", area: "ux_design", squad: ["team-felipe","team-beatriz","team-rosa","team-sara","team-teo"] },
  dante:    { type: "lead", area: "digital_product", squad: ["team-sergio","team-elena-frontend","team-uriel-devops"] },
  eidren:   { type: "lead", area: "research_intelligence", squad: ["team-carmen","team-celia","team-fatima","team-ines","team-kaori","team-mei"] },
  sofia:    { type: "lead", area: "commercial_growth", squad: ["team-gabriela","team-gael","team-leon","team-marina","team-oliver","team-ramiro","team-victor"] },
  uriel:    { type: "lead", area: "devops_infrastructure", squad: ["team-camila","team-diego"] },
  renata:   { type: "lead", area: "health_performance", squad: ["team-antonio","team-bruno","team-karina","team-luna","team-paula","team-ryu","team-sandra"] },
  camila:   { type: "lead", area: "legal_compliance", squad: [] },
  kenji:    { type: "lead", area: "adversarial_intelligence", squad: ["team-akiko","team-hiroshi","team-ruben","team-silvia","team-tomas"] },
  valeria:  { type: "lead", area: "education_career", squad: ["team-carmen","team-hugo","team-julian"] },
};

// Team agents
const TEAM_DIRECTORY = {
  // Thavren squad
  "team-andres":  { lead: "thavren", area: "platform_engineering" },
  "team-lucas":   { lead: "thavren", area: "platform_engineering" },
  "team-helena":  { lead: "thavren", area: "platform_engineering" },
  "team-irene":   { lead: "thavren", area: "platform_engineering" },
  "team-diana":   { lead: "thavren", area: "platform_engineering" },
  "team-pablo":   { lead: "thavren", area: "platform_engineering" },
  "team-oscar":   { lead: "thavren", area: "platform_engineering" },
  "team-nora":    { lead: "thavren", area: "platform_engineering" },
  "team-nadia":   { lead: "thavren", area: "platform_engineering" },
  "team-mia":     { lead: "thavren", area: "platform_engineering" },
  "team-clara":   { lead: "thavren", area: "platform_engineering" },
  "team-marco":   { lead: "thavren", area: "platform_engineering" },
  // Elena squad
  "team-felipe":  { lead: "elena", area: "ux_design" },
  "team-beatriz": { lead: "elena", area: "ux_design" },
  "team-rosa":    { lead: "elena", area: "ux_design" },
  "team-sara":    { lead: "elena", area: "ux_design" },
  "team-teo":     { lead: "elena", area: "ux_design" },
  // Dante squad
  "team-sergio":  { lead: "dante", area: "digital_product" },
  "team-elena-frontend": { lead: "dante", area: "digital_product" },
  "team-uriel-devops": { lead: "dante", area: "digital_product" },
  // Eidren squad
  "team-carmen":  { lead: "eidren", area: "research_intelligence" },
  "team-celia":   { lead: "eidren", area: "research_intelligence" },
  "team-fatima":  { lead: "eidren", area: "research_intelligence" },
  "team-ines":    { lead: "eidren", area: "research_intelligence" },
  "team-kaori":   { lead: "eidren", area: "research_intelligence" },
  "team-mei":     { lead: "eidren", area: "research_intelligence" },
  // Sofía squad
  "team-gabriela":{ lead: "sofia", area: "commercial_growth" },
  "team-gael":    { lead: "sofia", area: "commercial_growth" },
  "team-leon":    { lead: "sofia", area: "commercial_growth" },
  "team-marina":  { lead: "sofia", area: "commercial_growth" },
  "team-oliver":  { lead: "sofia", area: "commercial_growth" },
  "team-ramiro":  { lead: "sofia", area: "commercial_growth" },
  "team-victor":  { lead: "sofia", area: "commercial_growth" },
  // Uriel squad
  "team-camila":  { lead: "uriel", area: "devops_infrastructure" },
  "team-diego":   { lead: "uriel", area: "devops_infrastructure" },
  // Renata squad
  "team-antonio": { lead: "renata", area: "health_performance" },
  "team-bruno":   { lead: "renata", area: "health_performance" },
  "team-karina":  { lead: "renata", area: "health_performance" },
  "team-luna":    { lead: "renata", area: "health_performance" },
  "team-paula":   { lead: "renata", area: "health_performance" },
  "team-ryu":     { lead: "renata", area: "health_performance" },
  "team-sandra":  { lead: "renata", area: "health_performance" },
  // Kenji squad
  "team-akiko":   { lead: "kenji", area: "adversarial_intelligence" },
  "team-hiroshi": { lead: "kenji", area: "adversarial_intelligence" },
  "team-ruben":   { lead: "kenji", area: "adversarial_intelligence" },
  "team-silvia":  { lead: "kenji", area: "adversarial_intelligence" },
  "team-tomas":   { lead: "kenji", area: "adversarial_intelligence" },
  // Valeria squad
  "team-hugo":    { lead: "valeria", area: "education_career" },
  "team-julian":  { lead: "valeria", area: "education_career" },
};

function resolveAgent(agentId) {
  const lower = agentId.toLowerCase();

  // Lead: "thavren", "lead-thavren" → lead-thavren
  if (LEAD_DIRECTORY[lower]) {
    return { type: "lead", id: `lead-${lower}`, area: LEAD_DIRECTORY[lower].area };
  }
  if (lower.startsWith("lead-") && LEAD_DIRECTORY[lower.slice(5)]) {
    return { type: "lead", id: lower, area: LEAD_DIRECTORY[lower.slice(5)].area };
  }

  // Team: "team-clara" → team-clara
  if (TEAM_DIRECTORY[lower]) {
    return { type: "team", id: lower, area: TEAM_DIRECTORY[lower].area, lead: TEAM_DIRECTORY[lower].lead };
  }

  // Fallback: treat as generic subagent
  return { type: "unknown", id: lower, area: null };
}

export default async function ovavDelegate(args, { agent, parallel, log }) {
  const { agent_id, task, context = "state", model } = args;

  if (!agent_id || !task) {
    return {
      error: "Missing required fields: agent_id and task",
      usage: "workflow('ovav-delegate', { agent_id: 'lead-eidren', task: '...', context: 'state' })"
    };
  }

  log(`🚀 OVAV Delegate: ${agent_id} → "${task.substring(0, 60)}..."`);

  const resolved = resolveAgent(agent_id);
  log(`📍 Resolved: type=${resolved.type}, id=${resolved.id}, area=${resolved.area}`);

  if (resolved.type === "unknown") {
    log(`⚠️  Unknown agent: ${agent_id} — falling back to general agent`);
  }

  // Build agent options
  const agentOptions = {
    prompt: `<OVAV DELEGATION>
AREA: ${resolved.area || "unknown"}
DELEGATED_TASK: ${task}

Ejecuta este task siguiendo los protocolos de OVAV para el área ${resolved.area || "desconocida"}.
Reporta el resultado al agente que te delegó.

<Task>
${task}
</Task>
`,
    context_mode: context,
  };

  if (model) {
    agentOptions.model = model;
  }

  // Invoke the resolved agent via workflow agent()
  // NOTE: agent() inside workflow can accept subagent_type beyond explore/general
  // This is the Bug B fix — workflow+agent() bypasses actor.run limitation
  if (resolved.type === "lead" || resolved.type === "team") {
    agentOptions.subagent_type = resolved.id;
  }

  log(`📤 Invoking agent with subagent_type=${resolved.id}...`);
  const result = await agent(agentOptions);

  return {
    delegated_to: resolved.id,
    delegated_type: resolved.type,
    delegated_area: resolved.area,
    task: task,
    result: result,
  };
}
