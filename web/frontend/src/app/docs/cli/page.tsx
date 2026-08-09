import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "CLI Reference — OVAV Docs",
  description: "Referencia completa de la línea de comandos de OVAV.",
};

const COMMANDS = [
  { cmd: "ovav init", desc: "Inicializar OVAV en el directorio actual", flags: "--force, --template" },
  { cmd: "ovav status", desc: "Ver estado del sistema, sesiones activas, presupuesto", flags: "--json, --watch" },
  { cmd: "ovav validate", desc: "Ejecutar todos los validators", flags: "--category, --ci, --format" },
  { cmd: "ovav backup", desc: "Backup gobernado (crear, restaurar, listar)", flags: "--create, --restore, --list, --schedule" },
  { cmd: "ovav boundary", desc: "Gestionar Boundary Law", flags: "set, list, remove, area" },
  { cmd: "ovav license", desc: "Gestionar licencia", flags: "set, status, renew, revoke" },
  { cmd: "ovav connector", desc: "Gestionar conectores", flags: "register, enable, disable, list" },
  { cmd: "ovav secrets", desc: "Escanear y gestionar secretos", flags: "scan, ignore, audit" },
  { cmd: "ovav session", desc: "Gestionar sesiones de agente", flags: "list, kill, inspect" },
  { cmd: "ovav config", desc: "Configuración global", flags: "get, set, list, reset" },
  { cmd: "ovav update", desc: "Actualizar OVAV a la última versión", flags: "--check, --channel" },
  { cmd: "ovav uninstall", desc: "Desinstalar OVAV limpiamente", flags: "--purge, --keep-config" },
];

export default function CLIPage() {
  return (
    <main className="max-w-3xl mx-auto px-6 py-12">
      <a href="/docs" className="text-sm text-gray-400 hover:text-emerald-400 mb-6 inline-block transition">← Docs</a>
      <h1 className="text-3xl font-bold mb-3">CLI Reference</h1>
      <p className="text-lg text-gray-400 mb-10">Referencia completa de la línea de comandos.</p>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">Instalación</h2>
        <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm mb-3"><code>curl -sSL https://ovav.dev/install | bash</code></pre>
        <p className="text-gray-400 text-sm">Disponible en Linux, macOS, y WSL. Python 3.11+ requerido.</p>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">Comandos</h2>
        <div className="space-y-2">
          {COMMANDS.map((c) => (
            <div key={c.cmd} className="p-3 border border-gray-800 rounded-xl">
              <code className="text-emerald-400 text-sm font-medium">{c.cmd}</code>
              <p className="text-xs text-gray-400 mt-1">{c.desc}</p>
              <p className="text-xs text-gray-600 mt-0.5">Flags: {c.flags}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">Flags globales</h2>
        <div className="space-y-2">
          {[
            { flag: "--help, -h", desc: "Mostrar ayuda" },
            { flag: "--version, -v", desc: "Mostrar versión" },
            { flag: "--json", desc: "Output en formato JSON" },
            { flag: "--quiet, -q", desc: "Modo silencioso" },
            { flag: "--debug", desc: "Modo debug con output detallado" },
          ].map((f) => (
            <div key={f.flag} className="flex gap-4 p-2 border-b border-gray-800/50 text-sm">
              <code className="text-gray-300 shrink-0 w-40">{f.flag}</code>
              <span className="text-gray-500">{f.desc}</span>
            </div>
          ))}
        </div>
      </section>
    </main>
  );
}
