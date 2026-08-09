import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Boundary Law — OVAV Docs",
  description: "Cada agente opera en su carril. Nadie pisa ramas protegidas.",
};

export default function BoundaryLawPage() {
  return (
    <main className="max-w-3xl mx-auto px-6 py-12">
      <a href="/docs" className="text-sm text-gray-400 hover:text-emerald-400 mb-6 inline-block transition">← Docs</a>
      <h1 className="text-3xl font-bold mb-3">Boundary Law</h1>
      <p className="text-lg text-gray-400 mb-10">El principio fundacional de OVAV: cada agente tiene su carril. Nadie se sale.</p>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">¿Qué es Boundary Law?</h2>
        <p className="text-gray-400 mb-4">
          Boundary Law es el mecanismo de OVAV que define áreas de trabajo inexpugnables para los agentes de IA.
          Cada agente —Thavren, Dante, Eidren, cualquier otro— opera dentro de límites predefinidos que no puede
          cruzar sin autorización explícita.
        </p>
        <div className="p-4 border border-gray-800 rounded-xl bg-gray-900/50">
          <p className="text-sm text-gray-400">
            <span className="text-emerald-400 font-semibold">Regla:</span> Ningún agente puede modificar ramas
            protegidas, acceder a secretos, o ejecutar trabajo fuera de su área asignada. Si lo intenta,
            OVAV lo bloquea automáticamente. No es una sugerencia — es una ley.
          </p>
        </div>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">Cómo funciona</h2>
        <div className="space-y-4">
          <div className="p-4 border border-gray-800 rounded-xl">
            <h3 className="font-medium text-emerald-400 mb-1">1. Definición de áreas</h3>
            <p className="text-sm text-gray-400">Cada agente tiene un scope documentado: qué puede tocar, qué no, y a quién pedirle ayuda.</p>
          </div>
          <div className="p-4 border border-gray-800 rounded-xl">
            <h3 className="font-medium text-emerald-400 mb-1">2. Gates automatizados</h3>
            <p className="text-sm text-gray-400">OVAV valida cada operación antes de ejecutarla. Si está fuera de scope, la bloquea sin preguntar.</p>
          </div>
          <div className="p-4 border border-gray-800 rounded-xl">
            <h3 className="font-medium text-emerald-400 mb-1">3. Handoff Protocol</h3>
            <p className="text-sm text-gray-400">Cuando un agente no puede hacer algo, deriva formalmente al área correcta con contexto completo.</p>
          </div>
          <div className="p-4 border border-gray-800 rounded-xl">
            <h3 className="font-medium text-emerald-400 mb-1">4. Auditoría completa</h3>
            <p className="text-sm text-gray-400">Cada bloqueo, cada handoff, cada decisión queda registrada. Trazabilidad total.</p>
          </div>
        </div>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">Configuración</h2>
        <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm mb-3"><code>{`# Proteger ramas
ovav boundary set --branch main --action deny-push
ovav boundary set --branch main --action deny-force-push
ovav boundary set --branch develop --action require-review

# Definir áreas de agente
ovav boundary area --name frontend --allow "src/**/*.tsx"
ovav boundary area --name backend --allow "api/**/*.py"

# Ver configuración activa
ovav boundary list`}</code></pre>
      </section>

      <div className="p-6 border border-gray-800 rounded-xl bg-gray-900/50">
        <h2 className="font-semibold mb-2">¿Por qué es importante?</h2>
        <p className="text-sm text-gray-400">
          Sin Boundary Law, un agente malicioso o mal configurado podría modificar producción,
          exponer secretos, o corromper el código base. Con Boundary Law, cada operación está
          gobernada. No es paranoia — es ingeniería de seguridad.
        </p>
      </div>
    </main>
  );
}
