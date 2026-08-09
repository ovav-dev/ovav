const PROFILES = [
  {
    name: "Platform Engineering",
    lead: "Thavren",
    color: "#8ec07c",
    location: "🇩🇪 Berlin",
    description:
      "Infrastructure, security, CLI, runtime. The foundation everything else runs on. Go-native, zero dependencies.",
    skills: ["Go runtime", "CLI design", "Docker", "CI/CD", "Security"],
  },
  {
    name: "Digital Product Engineering",
    lead: "Dante",
    color: "#fe8019",
    location: "🇮🇹 Milan",
    description:
      "Web apps, frontend, deploy pipelines. Craftsmanship through design. TypeScript, Next.js, Cloudflare.",
    skills: ["React", "Next.js", "TypeScript", "Cloudflare Pages", "UI/UX"],
  },
  {
    name: "Evidence & Decision Intelligence",
    lead: "Eidren",
    color: "#83a598",
    location: "🇬🇧 London",
    description:
      "Research, benchmarks, competitive analysis. Evidence-backed decisions, not opinions. Sources verified.",
    skills: ["Research", "Benchmarks", "Data analysis", "Source verification"],
  },
  {
    name: "Education & Career Development",
    lead: "Valeria",
    color: "#d3869b",
    location: "🇨🇦 Toronto",
    description:
      "Learning paths, curriculum engines, skill gap detection. Personalized development plans backed by data.",
    skills: ["Curriculum design", "Gap analysis", "Mentoring", "Career paths"],
  },
  {
    name: "Health & Performance Science",
    lead: "Renata",
    color: "#b8bb26",
    location: "🇧🇷 São Paulo",
    description:
      "Nutrition, fitness, cognitive performance. A healthy developer is a productive developer.",
    skills: ["Nutrition science", "Fitness planning", "Cognitive health", "Wellness"],
  },
  {
    name: "Commercial & Growth Strategy",
    lead: "Sofía",
    color: "#fabd2f",
    location: "🇪🇸 Barcelona",
    description:
      "Pricing, GTM, partnerships, revenue. Turning technical excellence into sustainable business.",
    skills: ["GTM strategy", "Pricing", "Market analysis", "Partnerships"],
  },
  {
    name: "DevOps & Infrastructure",
    lead: "Uriel",
    color: "#8ec07c",
    location: "🇮🇱 Haifa",
    description:
      "Deployment, monitoring, reliability. Keeping OVAV running 24/7 across Fly.io and Cloudflare.",
    skills: ["Fly.io", "Cloudflare", "Docker", "Monitoring", "Reliability"],
  },
  {
    name: "UI/UX Design",
    lead: "Laura",
    color: "#fe8019",
    location: "🇸🇪 Stockholm",
    description:
      "Interface design, usability, accessibility. Italian design meets Scandinavian minimalism.",
    skills: ["UI Design", "UX Research", "Accessibility", "Design systems"],
  },
];

export function Profiles() {
  return (
    <section id="profiles" className="section-container bg-ovav-surface/50">
      <h2 className="section-title">
        Eight Professional Profiles.<br />
        <span className="gradient-text">One Workstation Governor.</span>
      </h2>
      <p className="section-subtitle">
        OVAV doesn&apos;t give you one generic &quot;developer&quot; profile.
        It gives you eight specialized professional personas, each with its own
        domain expertise, workflows, and quality standards.
      </p>

      <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-4 max-w-6xl mx-auto">
        {PROFILES.map((profile) => (
          <div
            key={profile.name}
            className="card group hover:border-ovav-accent/50 transition-colors duration-200"
          >
            {/* Color bar */}
            <div
              className="w-full h-1 rounded-full mb-4"
              style={{ backgroundColor: profile.color }}
            />

            <div className="flex items-start justify-between mb-3">
              <h3 className="font-semibold text-white text-sm leading-tight">
                {profile.name}
              </h3>
              <span className="text-xs text-ovav-muted ml-2 flex-shrink-0">
                {profile.location}
              </span>
            </div>

            <p className="text-xs text-ovav-muted mb-3 leading-relaxed">
              {profile.description}
            </p>

            <div className="flex items-center gap-2 mb-3">
              <span
                className="text-xs font-medium px-2 py-0.5 rounded-full"
                style={{
                  backgroundColor: profile.color + "20",
                  color: profile.color,
                }}
              >
                {profile.lead}
              </span>
              <span className="text-xs text-ovav-muted">Lead</span>
            </div>

            <div className="flex flex-wrap gap-1.5">
              {profile.skills.map((skill) => (
                <span
                  key={skill}
                  className="text-[11px] px-2 py-0.5 rounded-md bg-ovav-bg text-ovav-muted border border-ovav-border"
                >
                  {skill}
                </span>
              ))}
            </div>
          </div>
        ))}
      </div>

      <p className="text-center text-sm text-ovav-muted mt-10">
        Each profile runs as an independent agent with its own tools, standards,
        and quality gates.{" "}
        <span className="text-ovav-text">Six leads. Eight areas. One governor.</span>
      </p>
    </section>
  );
}
