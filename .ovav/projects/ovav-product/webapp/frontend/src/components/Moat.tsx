const MOAT_PILLARS = [
  {
    icon: "🎯",
    title: "Professional SDLC orchestration",
    description:
      "Plan → build → test → deploy. Governed end-to-end with professional standards, not just code suggestions at one phase.",
  },
  {
    icon: "🔒",
    title: "Local-first professional workstation",
    description:
      "Your code stays on your machine. Your standards are yours. Vault encryption, zero cloud dependency, complete control.",
  },
  {
    icon: "👥",
    title: "Eight professional profiles",
    description:
      "Platform Engineering. Product Development. Evidence & Research. Health Science. Commercial Strategy. Education. Career Development. Deep expertise, not generic autocomplete.",
  },
  {
    icon: "🧩",
    title: "Tool-agnostic architecture",
    description:
      "Bring your own models, editors, and platforms. OpenAI today, Claude tomorrow, Llama on Friday. OVAV governs them all. Zero vendor lock-in.",
  },
  {
    icon: "⚡",
    title: "Pure Go native runtime",
    description:
      "15MB binary. Zero Electron overhead. Zero cloud dependency. Runs on your machine, your CI, your servers. Built to last.",
  },
  {
    icon: "📊",
    title: "Evidence-backed professional standards",
    description:
      "Benchmarks, research briefs, auditable decision trails. Every professional decision leaves a record. No black boxes.",
  },
  {
    icon: "🔓",
    title: "Open-core professional standard",
    description:
      "Auditable. Contributable. Vendor-independent. Go + TypeScript. Built in the open, governed with integrity.",
  },
];

export function Moat() {
  return (
    <section id="moat" className="section-container">
      <h2 className="section-title">
        Why OVAV
      </h2>
      <p className="section-subtitle">
        OVAV is the governance layer your development workflow has been missing.
        Not another AI tool — the professional standard that orchestrates
        everything you already use.
      </p>

      {/* Moat pillars */}
      <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4 max-w-5xl mx-auto mb-16">
        {MOAT_PILLARS.map((pillar) => (
          <div key={pillar.title} className="card">
            <span className="text-2xl mb-3 block">{pillar.icon}</span>
            <h3 className="font-semibold text-white mb-2">{pillar.title}</h3>
            <p className="text-sm text-ovav-muted leading-relaxed">
              {pillar.description}
            </p>
          </div>
        ))}
      </div>

      {/* Layer Above comparison */}
      <h3 className="text-2xl font-bold text-center mb-2">
        The layer above your tools
      </h3>
      <p className="text-sm text-ovav-muted text-center mb-8 max-w-2xl mx-auto">
        OVAV doesn&apos;t compete with your editor, your AI assistant, or your
        deployment platform. It governs all of them. Like a conductor
        doesn&apos;t compete with the musicians — it makes them better together.
      </p>

      <div className="overflow-x-auto max-w-5xl mx-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-ovav-border text-left">
              <th className="py-3 px-4 text-ovav-muted font-medium">What you use</th>
              <th className="py-3 px-4 text-ovav-muted font-medium">What it does</th>
              <th className="py-3 px-4 text-white font-semibold bg-ovav-accent/10 rounded-t-lg">
                How OVAV governs it
              </th>
            </tr>
          </thead>
          <tbody className="text-ovav-muted">
            {[
              {
                tool: "VS Code / Neovim / Terminal",
                does: "Your editor",
                governs:
                  "OVAV orchestrates across all editors — 8 profiles, evidence trails, professional standards",
              },
              {
                tool: "Copilot / Cursor / Continue",
                does: "AI code suggestions",
                governs:
                  "OVAV governs which model, when, with what context — auditable, switchable, controllable",
              },
              {
                tool: "Vercel / Fly.io / Railway",
                does: "Deployment",
                governs:
                  "OVAV manages deploy targets, environment configs, and promotion pipelines with evidence",
              },
              {
                tool: "Linear / Jira / Notion",
                does: "Project management",
                governs:
                  "OVAV links professional decisions to tasks — every commit has a governed context",
              },
              {
                tool: "Sentry / Datadog",
                does: "Monitoring",
                governs:
                  "OVAV provides pre-deploy validation and post-deploy evidence — not just alerts, accountability",
              },
            ].map((row, i) => (
              <tr
                key={row.tool}
                className={`border-b border-ovav-border/50 ${
                  i % 2 === 0 ? "bg-ovav-surface/30" : ""
                }`}
              >
                <td className="py-2.5 px-4 text-ovav-text text-xs font-medium">
                  {row.tool}
                </td>
                <td className="py-2.5 px-4 text-xs">{row.does}</td>
                <td className="py-2.5 px-4 text-xs bg-ovav-accent/5">
                  <span className="text-ovav-accent2">{row.governs}</span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <p className="text-center text-sm text-ovav-muted mt-6 max-w-2xl mx-auto">
        OVAV makes every tool you already pay for more valuable.
        It&apos;s the professional standard that connects them.
      </p>
    </section>
  );
}
