import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Backup Gobernado — OVAV Docs",
  description: "Backups verificados. Sin secretos. Sin basura.",
};

export default function BackupPage() {
  return (
    <main className="max-w-3xl mx-auto px-6 py-12">
      <a href="/docs" className="text-sm text-gray-400 hover:text-emerald-400 mb-6 inline-block transition">← Docs</a>
      <h1 className="text-3xl font-bold mb-3">Backup Gobernado</h1>
      <p className="text-lg text-gray-400 mb-10">Backups que saben lo que hacen. Sin secretos, sin basura, sin sorpresas.</p>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">¿Por qué necesitás backup gobernado?</h2>
        <p className="text-gray-400 mb-4">
          Un backup tradicional copia todo — incluidos secretos, node_modules, archivos temporales,
          y basura que no debería persistir. El backup gobernado de OVAV aplica las mismas reglas
          de gobernanza antes de empaquetar: filtra secretos, ignora basura, y verifica integridad.
        </p>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">Comandos</h2>
        <div className="space-y-4">
          <div className="p-4 border border-gray-800 rounded-xl">
            <h3 className="font-medium text-emerald-400 text-sm mb-1">Crear backup</h3>
            <pre className="bg-gray-900 p-3 rounded-lg overflow-x-auto text-sm"><code>ovav backup --create</code></pre>
            <p className="text-xs text-gray-500 mt-2">Verifica, filtra, empaqueta, y firma el backup.</p>
          </div>
          <div className="p-4 border border-gray-800 rounded-xl">
            <h3 className="font-medium text-emerald-400 text-sm mb-1">Restaurar</h3>
            <pre className="bg-gray-900 p-3 rounded-lg overflow-x-auto text-sm"><code>ovav backup --restore backup-2026-06-09.tar.gz</code></pre>
            <p className="text-xs text-gray-500 mt-2">Verifica firma, integridad, y restaura en entorno aislado.</p>
          </div>
          <div className="p-4 border border-gray-800 rounded-xl">
            <h3 className="font-medium text-emerald-400 text-sm mb-1">Listar backups</h3>
            <pre className="bg-gray-900 p-3 rounded-lg overflow-x-auto text-sm"><code>ovav backup --list</code></pre>
            <p className="text-xs text-gray-500 mt-2">Muestra fecha, tamaño, y estado de integridad de cada backup.</p>
          </div>
        </div>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">Qué excluye automáticamente</h2>
        <ul className="space-y-2 text-gray-400 text-sm">
          <li className="flex gap-2"><span className="text-emerald-400">✓</span> <code className="text-gray-500">node_modules/</code></li>
          <li className="flex gap-2"><span className="text-emerald-400">✓</span> <code className="text-gray-500">.next/</code>, <code className="text-gray-500">dist/</code>, <code className="text-gray-500">build/</code></li>
          <li className="flex gap-2"><span className="text-emerald-400">✓</span> Archivos con secretos detectados</li>
          <li className="flex gap-2"><span className="text-emerald-400">✓</span> <code className="text-gray-500">.env</code> y <code className="text-gray-500">.env.local</code></li>
          <li className="flex gap-2"><span className="text-emerald-400">✓</span> Logs, caches, y temporales</li>
        </ul>
      </section>

      <div className="p-6 border border-emerald-500/20 bg-emerald-500/5 rounded-xl">
        <h2 className="font-semibold mb-2">Automatización</h2>
        <p className="text-sm text-gray-400">
          Configurá backups automáticos pre-commit o pre-push:
          <code className="text-emerald-400 block mt-2">ovav backup --schedule daily --retention 30</code>
        </p>
      </div>
    </main>
  );
}
