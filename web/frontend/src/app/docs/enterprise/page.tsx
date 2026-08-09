import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Deploy Enterprise — OVAV Docs",
  description: "Desplegá OVAV en tu organización con SSO, auditoría y compliance.",
};

const FEATURES = [
  { name: "SSO", desc: "Google Workspace, Okta, Azure AD, SAML", tier: "Enterprise" },
  { name: "Audit Log", desc: "Registro completo de cada acción de cada agente", tier: "Enterprise" },
  { name: "Team Management", desc: "Roles, permisos granulares, squads", tier: "Enterprise" },
  { name: "Custom Rules", desc: "Reglas de gobernanza específicas de tu org", tier: "Enterprise" },
  { name: "Priority Support", desc: "Slack compartido + SLAs de respuesta", tier: "Enterprise" },
  { name: "Compliance Reports", desc: "SOC 2, GDPR, EU AI Act readiness", tier: "Enterprise" },
];

export default function EnterprisePage() {
  return (
    <main className="max-w-3xl mx-auto px-6 py-12">
      <a href="/docs" className="text-sm text-gray-400 hover:text-emerald-400 mb-6 inline-block transition">← Docs</a>
      <h1 className="text-3xl font-bold mb-3">Deploy Enterprise</h1>
      <p className="text-lg text-gray-400 mb-10">OVAV a escala organizacional. SSO, auditoría, compliance.</p>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">Features Enterprise</h2>
        <div className="space-y-3">
          {FEATURES.map((f) => (
            <div key={f.name} className="p-4 border border-gray-800 rounded-xl flex justify-between items-center">
              <div>
                <h3 className="font-medium text-sm">{f.name}</h3>
                <p className="text-xs text-gray-500 mt-0.5">{f.desc}</p>
              </div>
              <span className="text-xs bg-purple-500/20 text-purple-400 px-2 py-0.5 rounded">{f.tier}</span>
            </div>
          ))}
        </div>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">Arquitectura</h2>
        <div className="p-4 border border-gray-800 rounded-xl bg-gray-900/50">
          <pre className="text-xs text-gray-400 overflow-x-auto"><code>{`┌─────────────────────────────────────────┐
│              OVAV Enterprise              │
├─────────────────────────────────────────┤
│  SSO Gateway (Okta / Azure AD / SAML)    │
│  Admin Dashboard (Web)                   │
│  License Server (API)                    │
│  Audit Pipeline (PostgreSQL)             │
│  Agent Runtime (por developer)           │
└─────────────────────────────────────────┘`}</code></pre>
        </div>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">Despliegue</h2>
        <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm mb-3"><code>{`# On-premise (Docker)
docker run -d \\
  -e OVAV_SSO_PROVIDER=okta \\
  -e OVAV_SSO_CLIENT_ID=... \\
  ovav/enterprise:latest

# Cloud (AWS / GCP / Azure)
terraform apply -var-file=production.tfvars

# Hybrid (license server cloud + runtime local)
ovav license set ovav-ent-xxxx-xxxx-xxxx
ovav status`}</code></pre>
      </section>

      <div className="p-6 border border-emerald-500/20 bg-emerald-500/5 rounded-xl">
        <h2 className="font-semibold mb-2">¿Interesado?</h2>
        <p className="text-sm text-gray-400">
          Escribinos a{" "}
          <a href="mailto:enterprise@ovav.dev" className="text-emerald-400 hover:text-emerald-300">enterprise@ovav.dev</a>
          {" "}para una demo personalizada con tu stack.
        </p>
      </div>
    </main>
  );
}
