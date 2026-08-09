export function Hero() {
  return (
    <section className="section-container pt-32 sm:pt-40 pb-20">
      <div className="flex flex-col items-center text-center gap-6">
        {/* Badge */}
        <div className="inline-flex items-center gap-2 px-4 py-1.5 rounded-full bg-ovav-surface border border-ovav-border text-sm text-ovav-muted">
          <span className="w-2 h-2 rounded-full bg-green-500 animate-pulse" />
          Open Source — Go + TypeScript
        </div>

        {/* Main headline */}
        <h1 className="text-4xl sm:text-6xl lg:text-7xl font-extrabold leading-tight max-w-4xl">
          <span className="gradient-text">Professional Development</span>
          <br />
          <span className="text-white">Governance</span>
        </h1>

        {/* Subheadline */}
        <p className="text-lg sm:text-xl text-ovav-muted max-w-2xl">
          Eight expert profiles orchestrate your entire development workflow.
          Evidence-backed. Tool-agnostic. Local-first.
        </p>

        {/* CTAs */}
        <div className="flex flex-col sm:flex-row gap-4 mt-4">
          <a href="#pricing" className="btn-primary text-lg px-8 py-4">
            Start governing
          </a>
          <a href="#how-it-works" className="btn-secondary text-lg px-8 py-4">
            See how it works
          </a>
        </div>

        {/* Terminal preview */}
        <div className="mt-12 w-full max-w-2xl">
          <div className="card font-mono text-sm">
            <div className="flex items-center gap-2 mb-4 pb-3 border-b border-ovav-border">
              <span className="w-3 h-3 rounded-full bg-red-500" />
              <span className="w-3 h-3 rounded-full bg-yellow-500" />
              <span className="w-3 h-3 rounded-full bg-green-500" />
              <span className="text-ovav-muted ml-2 text-xs">terminal — ovav@workstation</span>
            </div>
            <div className="space-y-1">
              <p><span className="text-ovav-accent2">$</span> <span className="text-ovav-accent">ovav</span> plan</p>
              <p className="text-ovav-muted">┌─ OVAV Plan — Tailor Composer ──────────────┐</p>
              <p className="text-ovav-muted">│ Profile: Platform Engineering             │</p>
              <p className="text-ovav-muted">│ Model: deepseek-v4-pro                     │</p>
              <p className="text-ovav-muted">│ Task: Build landing page for ovav.dev       │</p>
              <p className="text-ovav-muted">│ Tools: Next.js · Tailwind · Cloudflare      │</p>
              <p className="text-ovav-muted">└────────────────────────────────────────────┘</p>
              <p className="text-ovav-accent2">✓</p>
            </div>
          </div>
        </div>

        {/* Trust signals */}
        <div className="mt-8 flex flex-wrap items-center justify-center gap-6 text-sm text-ovav-muted">
          <span>🛡️ Local-first — your code never leaves your machine</span>
          <span className="hidden sm:inline">·</span>
          <span>⚡ Go runtime — 15MB binary, zero dependencies</span>
          <span className="hidden sm:inline">·</span>
          <span>🔓 Open-core — auditable, contributable, trustworthy</span>
        </div>
      </div>
    </section>
  );
}
