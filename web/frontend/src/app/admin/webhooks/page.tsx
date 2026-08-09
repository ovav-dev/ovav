import type { Metadata } from "next";

export const metadata: Metadata = { title: "Webhooks — Admin OVAV" };

const WEBHOOKS = [
  { id: "1", event: "checkout.session.completed", status: "200", time: "Hoy, 14:23:05", customer: "Acme Corp", retries: 0 },
  { id: "2", event: "customer.subscription.updated", status: "200", time: "Hoy, 11:05:42", customer: "Maria Lopez", retries: 0 },
  { id: "3", event: "invoice.paid", status: "200", time: "Ayer, 22:15:30", customer: "DataFlow SA", retries: 0 },
  { id: "4", event: "customer.subscription.deleted", status: "200", time: "Ayer, 18:40:12", customer: "ExCustomer", retries: 1 },
  { id: "5", event: "checkout.session.expired", status: "200", time: "8 Jun, 09:22:00", customer: "—", retries: 0 },
];

export default function AdminWebhooksPage() {
  return (
    <div className="p-8">
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold">Webhooks</h1>
          <p className="text-gray-500 text-sm mt-1">Eventos de Stripe recibidos. Últimas 24h.</p>
        </div>
        <div className="flex items-center gap-3">
          <span className="flex items-center gap-2 text-sm text-gray-400">
            <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />
            Conectado
          </span>
          <button className="px-4 py-1.5 border border-gray-700 hover:border-gray-500 rounded-lg text-sm transition">Reenviar todos</button>
        </div>
      </div>

      <div className="border border-gray-800 rounded-xl overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-gray-800 bg-gray-900/50 text-left text-gray-500">
              <th className="py-3 px-4 font-medium">Evento</th>
              <th className="py-3 px-4 font-medium">Status</th>
              <th className="py-3 px-4 font-medium">Timestamp</th>
              <th className="py-3 px-4 font-medium">Cliente</th>
              <th className="py-3 px-4 font-medium">Reintentos</th>
              <th className="py-3 px-4 font-medium"></th>
            </tr>
          </thead>
          <tbody>
            {WEBHOOKS.map((w) => (
              <tr key={w.id} className="border-b border-gray-800/50 hover:bg-gray-900/30">
                <td className="py-3 px-4 font-mono text-xs text-gray-300">{w.event}</td>
                <td className="py-3 px-4">
                  <span className="text-xs px-1.5 py-0.5 rounded bg-emerald-500/20 text-emerald-400">{w.status}</span>
                </td>
                <td className="py-3 px-4 text-gray-500">{w.time}</td>
                <td className="py-3 px-4 text-gray-400">{w.customer}</td>
                <td className="py-3 px-4 text-gray-500">{w.retries}</td>
                <td className="py-3 px-4">
                  <button className="text-xs text-emerald-400 hover:text-emerald-300">Payload</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
