import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Quickstart — OVAV Docs",
  description: "Empezá con OVAV en 5 minutos.",
};

export default function QuickstartPage() {
  return (
    <main className="max-w-3xl mx-auto px-6 py-12">
      <a href="/docs" className="text-sm text-gray-400 hover:text-emerald-400 mb-6 inline-block transition">← Docs</a>
      <h1 className="text-3xl font-bold mb-3">Quickstart</h1>
      <p className="text-lg text-gray-400 mb-10">OVAV funcionando en tu proyecto en 5 minutos.</p>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">1. Instalación</h2>
        <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm mb-3"><code>curl -sSL https://ovav.dev/install | bash</code></pre>
        <p className="text-gray-400 text-sm">Un solo comando. Funciona en Linux, macOS, y WSL en Windows.</p>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">2. Inicializar OVAV en tu proyecto</h2>
        <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm mb-3"><code>cd tu-proyecto{"\n"}ovav init</code></pre>
        <p className="text-gray-400 text-sm">Esto crea la estructura <code className="text-emerald-400">.ovav/</code> con todos los archivos de gobernanza.</p>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">3. Verificar que todo funciona</h2>
        <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm mb-3"><code>ovav status{"\n"}ovav validate</code></pre>
        <p className="text-gray-400 text-sm"><code className="text-emerald-400">status</code> te muestra el estado del sistema. <code className="text-emerald-400">validate</code> corre todos los validators.</p>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">4. Tu primer backup gobernado</h2>
        <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm mb-3"><code>ovav backup --create</code></pre>
        <p className="text-gray-400 text-sm">OVAV verifica integridad de archivos antes de empaquetar. Sin secretos, sin basura.</p>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">5. Configurar Boundary Law</h2>
        <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm mb-3"><code>ovav boundary set --branch main --action deny-push{"\n"}ovav boundary set --branch main --action deny-force-push</code></pre>
        <p className="text-gray-400 text-sm">Ahora <code className="text-emerald-400">main</code> está protegida. Ningún agente puede pushear sin pasar por el flujo correcto.</p>
      </section>

      <div className="mt-12 p-6 border border-emerald-500/20 bg-emerald-500/5 rounded-xl">
        <h2 className="font-semibold mb-2">¿Listo para más?</h2>
        <p className="text-sm text-gray-400">
          La <a href="/docs/hora-cero" className="text-emerald-400 hover:text-emerald-300">Guía Hora Cero</a> te lleva
          de cero a productivo en 60 minutos con ejercicios prácticos.
        </p>
      </div>
    </main>
  );
}
