import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Validators — OVAV Docs",
  description: "Suite de validación completa para integridad del sistema.",
};

const CATEGORIES = [
  { name: "Seguridad", count: 18, items: ["Secrets hygiene", "File permissions", "Branch protection", "Output signature", "Drift detection", "Surface integrity"] },
  { name: "Integridad", count: 22, items: ["YAML schema validation", "Artifact consistency", "Reference integrity", "Duplicate detection", "Encoding checks"] },
  { name: "Gobernanza", count: 16, items: ["Boundary law compliance", "Permission authority", "Handoff protocol", "Area registry", "Scope validation"] },
  { name: "Runtime", count: 16, items: ["Session capsule health", "Token budget enforcement", "Trigger engine", "Health check loop", "Auto-sync status"] },
];

export default function ValidatorsPage() {
  return (
    <main className="max-w-3xl mx-auto px-6 py-12">
      <a href="/docs" className="text-sm text-gray-400 hover:text-emerald-400 mb-6 inline-block transition">← Docs</a>
      <h1 className="text-3xl font-bold mb-3">Validators</h1>
      <p className="text-lg text-gray-400 mb-10">72 validators. Cobertura total. Cero falsos positivos.</p>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">Visión general</h2>
        <p className="text-gray-400 mb-4">
          Los validators son el corazón mecánico de OVAV. No dependen de ningún modelo de IA — son código
          determinístico que verifica cada aspecto del sistema. Si algo falla, se bloquea. Sin excepciones.
        </p>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-6">
          {CATEGORIES.map((c) => (
            <div key={c.name} className="p-4 border border-gray-800 rounded-xl text-center">
              <div className="text-2xl font-bold text-emerald-400">{c.count}</div>
              <div className="text-xs text-gray-500 mt-1">{c.name}</div>
            </div>
          ))}
        </div>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">Categorías</h2>
        <div className="space-y-4">
          {CATEGORIES.map((c) => (
            <div key={c.name} className="p-4 border border-gray-800 rounded-xl">
              <h3 className="font-medium text-sm mb-2">
                {c.name} <span className="text-gray-600">({c.count} validators)</span>
              </h3>
              <div className="flex flex-wrap gap-1.5">
                {c.items.map((item) => (
                  <span key={item} className="text-xs bg-gray-800 text-gray-400 px-2 py-0.5 rounded">{item}</span>
                ))}
              </div>
            </div>
          ))}
        </div>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">Uso</h2>
        <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm mb-3"><code>{`# Ejecutar todos los validators
ovav validate

# Validators específicos
ovav validate --category security
ovav validate --category governance

# En CI/CD
ovav validate --ci --format json`}</code></pre>
      </section>

      <div className="p-6 border border-gray-800 rounded-xl bg-gray-900/50">
        <h2 className="font-semibold mb-2">Principio</h2>
        <p className="text-sm text-gray-400">
          Los validators son determinísticos. No usan IA. El mismo input siempre produce el mismo output.
          Si un validator falla, es porque hay un problema real — no un falso positivo del modelo.
        </p>
      </div>
    </main>
  );
}
