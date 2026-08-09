export function CTA() {
  return (
    <section id="cta" className="section-container bg-ovav-surface/50">
      <div className="max-w-3xl mx-auto text-center">
        <h2 className="section-title">
          Your workflow already has the instruments.<br />
          <span className="gradient-text">OVAV is the conductor.</span>
        </h2>
        <p className="section-subtitle">
          Launching on Product Hunt July 7, 2026. Join the waitlist for
          early access, exclusive beta features, and founder perks.
        </p>

        {/* Email signup */}
        <form
          className="flex flex-col sm:flex-row gap-3 max-w-md mx-auto mt-8"
          action="https://api.ovav.dev/v1/waitlist"
          method="POST"
        >
          <input
            type="email"
            name="email"
            placeholder="you@company.com"
            required
            className="flex-1 px-4 py-3 rounded-lg bg-ovav-bg border border-ovav-border text-white placeholder:text-ovav-muted focus:outline-none focus:border-ovav-accent transition-colors"
          />
          <button type="submit" className="btn-primary whitespace-nowrap">
            Start governing — Free
          </button>
        </form>
        <p className="text-xs text-ovav-muted mt-3">
          No spam. One email when we launch. Unsubscribe anytime.
        </p>

        {/* Alternative CTAs */}
        <div className="mt-12 grid sm:grid-cols-3 gap-4 max-w-2xl mx-auto">
          <a
            href="https://github.com/ovav/ovav"
            target="_blank"
            rel="noopener noreferrer"
            className="card hover:border-ovav-accent/50 transition-colors text-center"
          >
            <span className="text-2xl mb-2 block">⭐</span>
            <h4 className="font-semibold text-white text-sm">Star on GitHub</h4>
            <p className="text-xs text-ovav-muted mt-1">
              Open-core, Apache 2.0
            </p>
          </a>
          <a
            href="https://docs.ovav.dev"
            target="_blank"
            rel="noopener noreferrer"
            className="card hover:border-ovav-accent/50 transition-colors text-center"
          >
            <span className="text-2xl mb-2 block">📚</span>
            <h4 className="font-semibold text-white text-sm">Read the Docs</h4>
            <p className="text-xs text-ovav-muted mt-1">
              Quickstart in 5 minutes
            </p>
          </a>
          <a
            href="https://cpanel.ovav.dev"
            className="card hover:border-ovav-accent/50 transition-colors text-center"
          >
            <span className="text-2xl mb-2 block">🔧</span>
            <h4 className="font-semibold text-white text-sm">Access cPanel</h4>
            <p className="text-xs text-ovav-muted mt-1">
              For existing OVAV users
            </p>
          </a>
        </div>

        {/* Product Hunt badge */}
        <div className="mt-10">
          <p className="text-sm text-ovav-muted mb-3">Launching on</p>
          <a
            href="https://www.producthunt.com"
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-2 px-5 py-2.5 rounded-lg bg-[#DA552F] text-white font-semibold text-sm hover:bg-[#c44d2a] transition-colors"
          >
            <svg width="20" height="20" viewBox="0 0 40 40" fill="none">
              <path d="M40 20C40 31.0457 31.0457 40 20 40C8.9543 40 0 31.0457 0 20C0 8.9543 8.9543 0 20 0C31.0457 0 40 8.9543 40 20Z" fill="white"/>
              <path d="M22.5 20H18V15H22.5C23.8807 15 25 16.1193 25 17.5C25 18.8807 23.8807 20 22.5 20Z" fill="#DA552F"/>
              <path d="M18 25H13V15H18V25Z" fill="white"/>
            </svg>
            Product Hunt — July 7, 2026
          </a>
        </div>
      </div>
    </section>
  );
}
