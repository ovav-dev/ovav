export function Footer() {
  return (
    <footer className="border-t border-ovav-border bg-ovav-bg">
      <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
        <div className="grid sm:grid-cols-4 gap-8">
          {/* Brand */}
          <div className="sm:col-span-1">
            <a href="/" className="flex items-center gap-2 font-bold text-lg mb-3">
              <span className="text-ovav-accent">◆</span>
              <span className="text-white">OVAV</span>
            </a>
            <p className="text-xs text-ovav-muted leading-relaxed">
              AI Workstation Governor.
              <br />
              Open-core. Local-first.
              <br />
              Go + TypeScript.
            </p>
          </div>

          {/* Product */}
          <div>
            <h4 className="font-semibold text-white text-sm mb-3">Product</h4>
            <ul className="space-y-2 text-sm text-ovav-muted">
              <li><a href="#pricing" className="hover:text-ovav-text transition-colors">Pricing</a></li>
              <li><a href="#profiles" className="hover:text-ovav-text transition-colors">Profiles</a></li>
              <li><a href="#moat" className="hover:text-ovav-text transition-colors">Why OVAV</a></li>
              <li><a href="https://cpanel.ovav.dev" className="hover:text-ovav-text transition-colors">cPanel →</a></li>
            </ul>
          </div>

          {/* Resources */}
          <div>
            <h4 className="font-semibold text-white text-sm mb-3">Resources</h4>
            <ul className="space-y-2 text-sm text-ovav-muted">
              <li><a href="https://docs.ovav.dev" target="_blank" rel="noopener noreferrer" className="hover:text-ovav-text transition-colors">Documentation</a></li>
              <li><a href="https://github.com/ovav/ovav" target="_blank" rel="noopener noreferrer" className="hover:text-ovav-text transition-colors">GitHub</a></li>
              <li><a href="mailto:hello@ovav.dev" className="hover:text-ovav-text transition-colors">Contact</a></li>
            </ul>
          </div>

          {/* Company */}
          <div>
            <h4 className="font-semibold text-white text-sm mb-3">Company</h4>
            <ul className="space-y-2 text-sm text-ovav-muted">
              <li><span>Buenos Aires, Argentina</span></li>
              <li><a href="mailto:enterprise@ovav.dev" className="hover:text-ovav-text transition-colors">Enterprise Sales</a></li>
              <li><a href="mailto:hello@ovav.dev" className="hover:text-ovav-text transition-colors">hello@ovav.dev</a></li>
            </ul>
          </div>
        </div>

        <div className="mt-10 pt-6 border-t border-ovav-border flex flex-col sm:flex-row items-center justify-between gap-4 text-xs text-ovav-muted">
          <p>© 2026 OVAV. All rights reserved.</p>
          <div className="flex items-center gap-4">
            <a href="https://github.com/ovav/ovav" target="_blank" rel="noopener noreferrer" className="hover:text-ovav-text transition-colors">GitHub</a>
            <a href="https://twitter.com/ovav" target="_blank" rel="noopener noreferrer" className="hover:text-ovav-text transition-colors">Twitter</a>
            <span>Open-core · Apache 2.0</span>
          </div>
        </div>
      </div>
    </footer>
  );
}
