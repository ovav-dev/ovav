import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "ConnectorBus — OVAV Docs",
  description: "Extendé OVAV con tus propias herramientas y conectores.",
};

export default function ConnectorBusPage() {
  return (
    <main className="max-w-3xl mx-auto px-6 py-12">
      <a href="/docs" className="text-sm text-gray-400 hover:text-emerald-400 mb-6 inline-block transition">← Docs</a>
      <h1 className="text-3xl font-bold mb-3">ConnectorBus</h1>
      <p className="text-lg text-gray-400 mb-10">Sistema de extensibilidad. Conectá cualquier herramienta a OVAV.</p>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">¿Qué es?</h2>
        <p className="text-gray-400 mb-4">
          ConnectorBus es la capa de integración que permite conectar herramientas externas al ecosistema OVAV.
          APIs, CLIs, bases de datos, webhooks — cualquier cosa que tenga una interfaz puede integrarse.
        </p>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">Conectores disponibles</h2>
        <div className="grid gap-3">
          {[
            { name: "GitHub", desc: "PRs, issues, actions — gobernados por OVAV", status: "estable" },
            { name: "Slack", desc: "Notificaciones de validación y alertas de seguridad", status: "estable" },
            { name: "Linear/Jira", desc: "Issues sincronizados con gates de OVAV", status: "beta" },
            { name: "Datadog", desc: "Métricas de gobernanza en tu dashboard", status: "beta" },
            { name: "Custom HTTP", desc: "Webhook genérico para cualquier servicio", status: "estable" },
          ].map((c) => (
            <div key={c.name} className="p-4 border border-gray-800 rounded-xl flex justify-between items-center">
              <div>
                <h3 className="font-medium text-sm">{c.name}</h3>
                <p className="text-xs text-gray-500 mt-0.5">{c.desc}</p>
              </div>
              <span className={`text-xs px-2 py-0.5 rounded ${c.status === "estable" ? "bg-emerald-500/20 text-emerald-400" : "bg-yellow-500/20 text-yellow-400"}`}>{c.status}</span>
            </div>
          ))}
        </div>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">Crear tu propio conector</h2>
        <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm mb-3"><code>{`# 1. Crear el archivo del conector
mkdir -p .ovav/connectors/slack
touch .ovav/connectors/slack/connector.py

# 2. Implementar la interfaz
from ovav.connector import BaseConnector

class SlackConnector(BaseConnector):
    async def on_validate_pass(self, results):
        await self.send_slack(f"Validación OK: {results}")

# 3. Registrar
ovav connector register slack ./connectors/slack/connector.py

# 4. Activar
ovav connector enable slack`}</code></pre>
      </section>

      <div className="p-6 border border-emerald-500/20 bg-emerald-500/5 rounded-xl">
        <h2 className="font-semibold mb-2">Seguridad</h2>
        <p className="text-sm text-gray-400">
          Todo conector pasa por el mismo sistema de gobernanza. Boundary Law aplica a conectores
          igual que a agentes. Un conector no puede hacer nada que un agente no pueda.
        </p>
      </div>
    </main>
  );
}
