import type { Metadata } from "next";

export const metadata: Metadata = { title: "Clientes — Admin OVAV" };

const CUSTOMERS = [
  { id: "1", name: "Acme Corp", email: "admin@acme.com", plan: "Enterprise", since: "2026-01-15", mrr: "$2,450", users: 48 },
  { id: "2", name: "Maria Lopez", email: "maria@dev.io", plan: "Pro", since: "2026-03-22", mrr: "$19", users: 1 },
  { id: "3", name: "TechStart Inc", email: "ops@techstart.com", plan: "Pro", since: "2026-04-10", mrr: "$475", users: 25 },
  { id: "4", name: "DataFlow SA", email: "eng@dataflow.com", plan: "Enterprise", since: "2026-02-01", mrr: "$4,900", users: 100 },
  { id: "5", name: "Carlos Ruiz", email: "carlos@dev.mx", plan: "Core", since: "2026-05-18", mrr: "$0", users: 1 },
  { id: "6", name: "GlobalTech", email: "it@globaltech.com", plan: "Enterprise", since: "2025-11-30", mrr: "$9,800", users: 200 },
];

export default function AdminCustomersPage() {
  return (
    <div className="p-8">
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold">Clientes</h1>
          <p className="text-gray-500 text-sm mt-1">{CUSTOMERS.length} clientes activos</p>
        </div>
        <div className="flex gap-3">
          <input type="search" placeholder="Buscar cliente..." className="px-3 py-1.5 bg-gray-900 border border-gray-700 rounded-lg text-sm text-gray-300 placeholder-gray-500 focus:border-emerald-500 focus:outline-none w-64" />
          <select className="px-3 py-1.5 bg-gray-900 border border-gray-700 rounded-lg text-sm text-gray-300">
            <option>Todos los planes</option>
            <option>Core</option>
            <option>Pro</option>
            <option>Enterprise</option>
          </select>
          <button className="px-4 py-1.5 bg-emerald-500 hover:bg-emerald-400 text-gray-950 rounded-lg text-sm font-medium transition">Exportar CSV</button>
        </div>
      </div>

      <div className="border border-gray-800 rounded-xl overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-gray-800 bg-gray-900/50 text-left text-gray-500">
              <th className="py-3 px-4 font-medium">Cliente</th>
              <th className="py-3 px-4 font-medium">Email</th>
              <th className="py-3 px-4 font-medium">Plan</th>
              <th className="py-3 px-4 font-medium">Desde</th>
              <th className="py-3 px-4 font-medium">MRR</th>
              <th className="py-3 px-4 font-medium">Usuarios</th>
              <th className="py-3 px-4 font-medium"></th>
            </tr>
          </thead>
          <tbody>
            {CUSTOMERS.map((c) => (
              <tr key={c.id} className="border-b border-gray-800/50 hover:bg-gray-900/30">
                <td className="py-3 px-4 text-gray-200 font-medium">{c.name}</td>
                <td className="py-3 px-4 text-gray-400">{c.email}</td>
                <td className="py-3 px-4">
                  <span className={`text-xs px-1.5 py-0.5 rounded ${c.plan === "Enterprise" ? "bg-purple-500/20 text-purple-400" : c.plan === "Pro" ? "bg-emerald-500/20 text-emerald-400" : "bg-gray-700 text-gray-400"}`}>{c.plan}</span>
                </td>
                <td className="py-3 px-4 text-gray-500">{c.since}</td>
                <td className="py-3 px-4 text-gray-300 font-mono">{c.mrr}</td>
                <td className="py-3 px-4 text-gray-400">{c.users}</td>
                <td className="py-3 px-4">
                  <button className="text-xs text-emerald-400 hover:text-emerald-300">Ver →</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
