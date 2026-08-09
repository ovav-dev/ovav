import type { Metadata } from "next";

export const metadata: Metadata = { title: "Alertas — Admin OVAV" };

const ALERTS = [
  { id: "1", type: "churn_risk", severity: "high", message: "GlobalTech — 3 días para expirar, sin método de pago", time: "Hoy, 10:00", ack: false },
  { id: "2", type: "license_abuse", severity: "medium", message: "TechStart Inc — 3/3 instancias, intento de 4ta bloqueado", time: "Hoy, 08:45", ack: true },
  { id: "3", type: "payment_failed", severity: "high", message: "DataFlow SA — Pago rechazado por fondos insuficientes", time: "Ayer, 23:15", ack: false },
  { id: "4", type: "trial_ending", severity: "low", message: "12 trials terminan en las próximas 48h", time: "Ayer, 12:00", ack: false },
];

export default function AdminAlertsPage() {
  return (
    <div className="p-8">
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold">Alertas</h1>
          <p className="text-gray-500 text-sm mt-1">{ALERTS.filter(a => !a.ack).length} sin resolver</p>
        </div>
        <button className="px-4 py-1.5 border border-gray-700 hover:border-gray-500 rounded-lg text-sm transition">Marcar todas como leídas</button>
      </div>

      <div className="space-y-3">
        {ALERTS.map((a) => (
          <div key={a.id} className={`p-4 border rounded-xl flex items-start gap-4 ${a.ack ? "border-gray-800 opacity-60" : a.severity === "high" ? "border-red-500/30 bg-red-500/5" : a.severity === "medium" ? "border-yellow-500/30 bg-yellow-500/5" : "border-gray-800"}`}>
            <span className={`text-lg shrink-0 ${a.severity === "high" ? "" : ""}`}>
              {a.severity === "high" ? "🔴" : a.severity === "medium" ? "🟡" : "🟢"}
            </span>
            <div className="flex-1">
              <div className="flex items-center gap-2 mb-1">
                <span className="text-xs px-1.5 py-0.5 rounded bg-gray-800 text-gray-400 uppercase">{a.type}</span>
                {!a.ack && <span className="text-xs text-red-400">Nuevo</span>}
              </div>
              <p className="text-sm text-gray-200">{a.message}</p>
              <p className="text-xs text-gray-600 mt-1">{a.time}</p>
            </div>
            <div className="flex gap-2">
              {!a.ack && <button className="px-3 py-1 bg-emerald-500/20 text-emerald-400 rounded text-xs hover:bg-emerald-500/30">Resolver</button>}
              <button className="px-3 py-1 border border-gray-700 rounded text-xs text-gray-500 hover:text-gray-300">↗</button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
