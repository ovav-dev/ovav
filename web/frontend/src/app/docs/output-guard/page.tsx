import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Output Guard — OVAV Docs",
  description: "Cada respuesta verificada y firmada criptográficamente antes de entregarse.",
};

export default function OutputGuardPage() {
  return (
    <main className="max-w-3xl mx-auto px-6 py-12">
      <a href="/docs" className="text-sm text-gray-400 hover:text-emerald-400 mb-6 inline-block transition">← Docs</a>
      <h1 className="text-3xl font-bold mb-3">Output Guard</h1>
      <p className="text-lg text-gray-400 mb-10">Firma criptográfica de cada respuesta. Imposible de falsificar.</p>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">¿Qué hace el Output Guard?</h2>
        <p className="text-gray-400 mb-4">
          Cada vez que un agente de IA genera una respuesta, el Output Guard la verifica y firma
          criptográficamente usando HMAC-SHA256 con una clave secreta que solo OVAV conoce.
          Si alguien —un agente, un script, un atacante— intenta modificar ese output, la firma
          no coincide y OVAV bloquea la entrega.
        </p>
        <div className="p-4 border border-emerald-500/20 bg-emerald-500/5 rounded-xl">
          <p className="text-sm text-gray-400">
            <span className="text-emerald-400 font-semibold">En criollo:</span> Lo que ves en pantalla
            es exactamente lo que el agente generó. Nadie lo modificó en el camino. Tenés garantía
            criptográfica.
          </p>
        </div>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">Flujo de verificación</h2>
        <div className="space-y-3">
          {[
            { step: "1", label: "Agente genera respuesta", desc: "El modelo produce el output solicitado." },
            { step: "2", label: "OVAV intercepta", desc: "Antes de mostrarlo, el output pasa por el guard." },
            { step: "3", label: "Verificación de contenido", desc: "Se validan reglas de seguridad, patrones prohibidos, y coherencia." },
            { step: "4", label: "Firma HMAC-SHA256", desc: "Si pasa verificación, se firma con la clave secreta de OVAV." },
            { step: "5", label: "Entrega al usuario", desc: "Solo outputs con firma válida llegan a la pantalla." },
          ].map((s) => (
            <div key={s.step} className="flex gap-4 p-4 border border-gray-800 rounded-xl">
              <div className="w-8 h-8 rounded-full bg-emerald-500/20 border border-emerald-500/50 flex items-center justify-center text-sm font-bold text-emerald-400 shrink-0">{s.step}</div>
              <div>
                <h3 className="font-medium text-sm">{s.label}</h3>
                <p className="text-xs text-gray-500 mt-0.5">{s.desc}</p>
              </div>
            </div>
          ))}
        </div>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">Qué protege</h2>
        <ul className="space-y-2 text-gray-400 text-sm">
          <li className="flex gap-2"><span className="text-emerald-400">✓</span> Inyección de instrucciones maliciosas en el output</li>
          <li className="flex gap-2"><span className="text-emerald-400">✓</span> Modificación del contenido por agentes intermedios</li>
          <li className="flex gap-2"><span className="text-emerald-400">✓</span> Suplantación de identidad del agente</li>
          <li className="flex gap-2"><span className="text-emerald-400">✓</span> Exposición accidental de datos sensibles</li>
          <li className="flex gap-2"><span className="text-emerald-400">✓</span> Respuestas que violan las reglas de gobernanza</li>
        </ul>
      </section>

      <div className="p-6 border border-gray-800 rounded-xl bg-gray-900/50">
        <h2 className="font-semibold mb-2">Limitaciones</h2>
        <p className="text-sm text-gray-400">
          El Output Guard protege la integridad del mensaje entre el agente y vos. No protege contra
          un agente que deliberadamente genera contenido malicioso desde el origen — para eso está
          Boundary Law y el resto del sistema de gobernanza.
        </p>
      </div>
    </main>
  );
}
