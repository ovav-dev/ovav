import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Seguridad — OVAV Docs",
  description: "Threat model, controles de seguridad, y política de vulnerabilidades.",
};

const CONTROLS = [
  { name: "Secrets Hygiene", desc: "Detección pre-commit de secretos, API keys, tokens, y credenciales.", status: "Activo" },
  { name: "Output Guard", desc: "Firma HMAC-SHA256 de cada respuesta. Imposible de falsificar.", status: "Activo" },
  { name: "Boundary Law", desc: "Agentes confinados a sus áreas. Sin escalación de privilegios.", status: "Activo" },
  { name: "Drift Detection", desc: "Monitoreo continuo de cambios no autorizados en archivos de sistema.", status: "Activo" },
  { name: "Session Capsule", desc: "Aislamiento total entre sesiones. Sin cross-talk.", status: "Activo" },
  { name: "File Permissions", desc: "Verificación de permisos 600/700 en archivos sensibles.", status: "Activo" },
  { name: "Branch Protection", desc: "Ramas protegidas inexpugnables sin waiver del CEO.", status: "Activo" },
  { name: "Surface Governor", desc: "Validación de superficie de trabajo antes de cada operación.", status: "Activo" },
  { name: "Integrity Mesh", desc: "Malla de integridad que detecta y repara corrupción de archivos.", status: "Activo" },
  { name: "Trigger Engine", desc: "43 auto-triggers que validan cada operación en tiempo real.", status: "Activo" },
  { name: "Permission Authority", desc: "Matriz canónica de permisos. Ningún agente puede auto-elevarse.", status: "Activo" },
  { name: "Vault Encryption", desc: "Secretos en reposo con AES-256-GCM. Claves nunca en texto plano.", status: "Activo" },
  { name: "Audit Trail", desc: "Registro inmutable de cada decisión de gobernanza.", status: "Enterprise" },
  { name: "SSO Enforcement", desc: "Single Sign-On obligatorio con proveedores verificados.", status: "Enterprise" },
];

export default function SecurityPage() {
  return (
    <main className="max-w-3xl mx-auto px-6 py-12">
      <a href="/docs" className="text-sm text-gray-400 hover:text-emerald-400 mb-6 inline-block transition">← Docs</a>
      <h1 className="text-3xl font-bold mb-3">Seguridad</h1>
      <p className="text-lg text-gray-400 mb-10">Modelo de amenazas, controles, y política de divulgación.</p>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">Controles de seguridad</h2>
        <div className="space-y-2">
          {CONTROLS.map((c) => (
            <div key={c.name} className="p-3 border border-gray-800 rounded-xl flex justify-between items-start gap-4">
              <div>
                <h3 className="font-medium text-sm">{c.name}</h3>
                <p className="text-xs text-gray-500 mt-0.5">{c.desc}</p>
              </div>
              <span className={`text-xs px-2 py-0.5 rounded shrink-0 ${
                c.status === "Activo" ? "bg-emerald-500/20 text-emerald-400" : "bg-purple-500/20 text-purple-400"
              }`}>{c.status}</span>
            </div>
          ))}
        </div>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">Threat Model</h2>
        <p className="text-gray-400 text-sm mb-4">
          OVAV modela 4 adversarios y 10 vectores de ataque. Nuestro threat model completo está
          disponible en el repositorio público.
        </p>
        <div className="grid grid-cols-2 gap-3">
          {[
            { adversary: "Dev malicioso", risk: "Mitigado" },
            { adversary: "Atacante remoto", risk: "Mitigado" },
            { adversary: "OpenCode comprometido", risk: "Sin defensa" },
            { adversary: "Acceso físico root", risk: "Sin defensa" },
          ].map((a) => (
            <div key={a.adversary} className="p-3 border border-gray-800 rounded-xl">
              <div className="text-sm font-medium">{a.adversary}</div>
              <div className={`text-xs mt-1 ${a.risk === "Mitigado" ? "text-emerald-400" : "text-yellow-400"}`}>{a.risk}</div>
            </div>
          ))}
        </div>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">Reportar una vulnerabilidad</h2>
        <p className="text-gray-400 text-sm">
          Si encontrás una vulnerabilidad de seguridad, por favor no la publiques.
          Enviala a{" "}
          <a href="mailto:security@ovav.dev" className="text-emerald-400 hover:text-emerald-300">security@ovav.dev</a>
          . Respondemos en ≤48 horas. Ofrecemos recompensas por hallazgos válidos.
        </p>
      </section>

      <div className="p-6 border border-gray-800 rounded-xl bg-gray-900/50">
        <h2 className="font-semibold mb-2">Limitaciones</h2>
        <p className="text-sm text-gray-400">
          OVAV protege contra agentes de IA no autorizados y modificaciones maliciosas
          dentro del scope definido. No protege contra compromiso del sistema operativo,
          acceso físico, o un proveedor de herramientas malicioso. Leé el threat model completo
          para entender exactamente qué cubre y qué no.
        </p>
      </div>
    </main>
  );
}
