import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Session Capsule — OVAV Docs",
  description: "Cada sesión de agente está aislada. Sin fugas de contexto.",
};

export default function SessionCapsulePage() {
  return (
    <main className="max-w-3xl mx-auto px-6 py-12">
      <a href="/docs" className="text-sm text-gray-400 hover:text-emerald-400 mb-6 inline-block transition">← Docs</a>
      <h1 className="text-3xl font-bold mb-3">Session Capsule</h1>
      <p className="text-lg text-gray-400 mb-10">Aislamiento total entre sesiones. Sin contaminación de contexto.</p>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">El problema</h2>
        <p className="text-gray-400 mb-4">
          Los agentes de IA operan en sesiones. Sin aislamiento, una sesión puede filtrar información
          a otra: datos de un proyecto a otro, secretos entre contextos, o instrucciones maliciosas
          que persisten entre sesiones.
        </p>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">La solución de OVAV</h2>
        <p className="text-gray-400 mb-4">
          La Session Capsule crea un contenedor aislado para cada sesión de agente. Al iniciar una sesión:
        </p>
        <div className="space-y-3">
          {[
            { label: "Presupuesto de tokens", desc: "Límite máximo de tokens por sesión. Sin excepciones." },
            { label: "Firewall de contexto", desc: "Bloquea inyección de instrucciones desde fuentes externas." },
            { label: "Aislamiento de memoria", desc: "Cada sesión tiene su propio espacio. Sin cross-talk." },
            { label: "Conocimiento heredado", desc: "Solo lo que el ledger autoriza explícitamente se transfiere." },
            { label: "Caducidad automática", desc: "Al cerrar sesión, el contexto se destruye. Nada persiste." },
          ].map((item, i) => (
            <div key={i} className="p-4 border border-gray-800 rounded-xl">
              <h3 className="font-medium text-emerald-400 text-sm mb-1">{item.label}</h3>
              <p className="text-xs text-gray-500">{item.desc}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">Qué protege</h2>
        <ul className="space-y-2 text-gray-400 text-sm">
          <li className="flex gap-2"><span className="text-emerald-400">✓</span> Fugas de datos entre proyectos diferentes</li>
          <li className="flex gap-2"><span className="text-emerald-400">✓</span> Persistencia de instrucciones maliciosas entre sesiones</li>
          <li className="flex gap-2"><span className="text-emerald-400">✓</span> Exhaustión de presupuesto por sesiones descontroladas</li>
          <li className="flex gap-2"><span className="text-emerald-400">✓</span> Contaminación de un agente por el contexto de otro</li>
        </ul>
      </section>

      <div className="p-6 border border-gray-800 rounded-xl bg-gray-900/50">
        <h2 className="font-semibold mb-2">Transparencia</h2>
        <p className="text-sm text-gray-400">
          Cada sesión activa es visible en <code className="text-emerald-400">ovav status</code> con
          su presupuesto, uso actual, y tiempo restante. Nada está oculto.
        </p>
      </div>
    </main>
  );
}
