const TIERS = [
  {
    name: "OVAV Free",
    price: "$0",
    period: "forever",
    tagline: "For individual developers exploring professional governance.",
    highlight: "$0 — forever",
    cta: "Get Started Free",
    href: "#cta",
    featured: false,
    includes: [
      "Full CLI (ovav plan, build, test, deploy)",
      "2 professional profiles",
      "Community models (DeepSeek, Llama, etc.)",
      "Public documentation",
      "Community support (GitHub Discussions)",
      "Vault encryption (AES-256-GCM)",
    ],
  },
  {
    name: "OVAV Pro",
    price: "$19",
    period: "/mo",
    tagline: "For professional developers who govern their entire workflow.",
    highlight: "$19/mo",
    subHighlight: "$190/year (2 months free)",
    cta: "Get Pro Access",
    href: "#cta",
    featured: true,
    includes: [
      "Everything in Free",
      "All 8 professional profiles",
      "Unlimited models (OpenAI, Anthropic, Google, Azure, OSS)",
      "Evidence & Decision Intelligence (Eidren benchmarks)",
      "Tailor Composer — personalized dev plans",
      "Priority support (&lt;24h business hours)",
      "Early access to new features (beta channel)",
    ],
  },
  {
    name: "OVAV Enterprise",
    price: "$49",
    period: "/user/mo",
    tagline: "For teams that demand professional standards at scale.",
    highlight: "$49/user/mo",
    subHighlight: "SSO, audit logs, self-hosting, SLA",
    cta: "Contact Sales",
    href: "mailto:enterprise@ovav.dev",
    featured: false,
    includes: [
      "Everything in Pro",
      "SSO (SAML/OIDC) — Google, GitHub, Azure AD",
      "Full audit logs",
      "Self-hosting (Docker, Kubernetes, on-prem)",
      "Custom company profiles",
      "SLA: 99.5% uptime",
      "Dedicated support + account manager",
      "Onboarding sessions with OVAV engineers",
    ],
  },
];

export function Pricing() {
  return (
    <section id="pricing" className="section-container">
      <h2 className="section-title">Professional governance, professionally priced</h2>
      <p className="section-subtitle">
        OVAV sits in the same pricing corridor as the professional tools you
        already trust. Not priced like an AI add-on — priced like the
        governance layer your workflow deserves.
      </p>

      <div className="grid md:grid-cols-3 gap-6 max-w-5xl mx-auto">
        {TIERS.map((tier) => (
          <div
            key={tier.name}
            className={`card flex flex-col ${
              tier.featured ? "pricing-highlight relative" : ""
            }`}
          >
            {tier.featured && (
              <div className="absolute -top-3 left-1/2 -translate-x-1/2 px-4 py-1 rounded-full bg-ovav-accent text-white text-xs font-semibold">
                Most Popular
              </div>
            )}

            <div className="mb-6">
              <h3 className="text-lg font-semibold text-white">{tier.name}</h3>
              <div className="mt-3 flex items-baseline gap-1">
                <span className="text-4xl font-extrabold text-white">
                  {tier.price}
                </span>
                <span className="text-ovav-muted text-sm">{tier.period}</span>
              </div>
              {tier.subHighlight && (
                <p className="mt-1 text-xs text-ovav-accent font-medium">
                  {tier.subHighlight}
                </p>
              )}
              <p className="mt-2 text-sm text-ovav-muted">{tier.tagline}</p>
            </div>

            <ul className="space-y-3 mb-8 flex-1">
              {tier.includes.map((item) => (
                <li key={item} className="flex items-start gap-2 text-sm">
                  <span className="text-ovav-accent2 mt-0.5 flex-shrink-0">✓</span>
                  <span className="text-ovav-muted">{item}</span>
                </li>
              ))}
            </ul>

            <a
              href={tier.href}
              className={
                tier.featured
                  ? "btn-primary w-full !justify-center"
                  : "btn-secondary w-full !justify-center"
              }
            >
              {tier.cta}
            </a>
          </div>
        ))}
      </div>

      {/* OVAV in context — Layer Above pricing comparison */}
      <div className="max-w-3xl mx-auto mt-16">
        <h3 className="text-xl font-bold text-white text-center mb-6">
          OVAV in context
        </h3>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <tbody className="text-ovav-muted">
              {[
                { tool: "1Password", price: "$7.99/mo", category: "Credential security" },
                { tool: "Linear", price: "$10–16/mo", category: "Project management" },
                { tool: "OVAV Pro", price: "$19/mo", category: "Professional governance", highlight: true },
                { tool: "Vercel", price: "$20/mo", category: "Deployment platform" },
                { tool: "Sentry", price: "$26/mo", category: "Error monitoring" },
                { tool: "GitLab Premium", price: "$29/mo", category: "DevOps platform" },
              ].map((row) => (
                <tr
                  key={row.tool}
                  className={`border-b border-ovav-border/50 ${
                    row.highlight ? "bg-ovav-accent/10" : ""
                  }`}
                >
                  <td className="py-2.5 px-4 text-ovav-text text-xs font-medium">
                    {row.tool}
                  </td>
                  <td className="py-2.5 px-4 text-xs text-right">
                    {row.price}
                  </td>
                  <td className="py-2.5 px-4 text-xs text-ovav-muted">
                    {row.category}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <p className="text-center text-xs text-ovav-muted mt-4 max-w-xl mx-auto">
          OVAV is the governance layer that orchestrates all the tools
          above and below it. It&apos;s not a replacement — it&apos;s the professional
          standard that connects them.
        </p>
      </div>

      <p className="text-center text-sm text-ovav-muted mt-8">
        Save 17% with annual billing on Pro ($190/year).{" "}
        Enterprise minimum 10 seats.{" "}
        <a href="mailto:enterprise@ovav.dev" className="text-ovav-accent hover:underline">
          500+ seats? Contact us.
        </a>
      </p>
    </section>
  );
}
