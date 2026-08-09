import type { Metadata } from "next";

export const metadata: Metadata = { title: "Admin Dashboard — OVAV" };

export default function AdminDashboard() {
  const kpis = [
    { label: "Clientes totales", value: "1,247", change: "+12%", up: true },
    { label: "MRR", value: "$24,830", change: "+8.3%", up: true },
    { label: "Licencias activas", value: "3,842", change: "+5.1%", up: true },
    { label: "Churn rate", value: "2.1%", change: "-0.4%", up: true },
  ];

  return (
    <div className="p-8">
      <div className="flex justify-between items-center mb-8">
        <div>
          <h1 className="text-2xl font-bold">Dashboard</h1>
          <p className="text-gray-500 text-sm mt-1">Resumen del negocio en tiempo real.</p>
        </div>
        <select className="px-3 py-1.5 bg-gray-900 border border-gray-700 rounded-lg text-sm text-gray-300">
          <option>Últimos 30 días</option>
          <option>Últimos 90 días</option>
          <option>Este año</option>
        </select>
      </div>

      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
        {kpis.map((kpi) => (
          <div key={kpi.label} className="p-5 border border-gray-800 rounded-xl">
            <p className="text-xs text-gray-500 mb-1">{kpi.label}</p>
            <div className="flex items-baseline gap-2">
              <span className="text-2xl font-bold">{kpi.value}</span>
              <span className={`text-xs ${kpi.up ? "text-emerald-400" : "text-red-400"}`}>{kpi.change}</span>
            </div>
          </div>
        ))}
      </div>

      <div className="grid lg:grid-cols-2 gap-6">
        <div className="p-6 border border-gray-800 rounded-xl">
          <h2 className="font-semibold mb-4">Nuevos clientes</h2>
          <div className="space-y-3">
            {[
              { name: "Acme Corp", plan: "Enterprise", date: "Hoy, 14:23", amount: "$2,450" },
              { name: "Maria Lopez", plan: "Pro", date: "Hoy, 11:05", amount: "$19" },
              { name: "TechStart Inc", plan: "Pro", date: "Ayer, 16:40", amount: "$19" },
              { name: "Carlos Ruiz", plan: "Core", date: "Ayer, 09:15", amount: "$0" },
              { name: "DataFlow SA", plan: "Enterprise", date: "8 Jun", amount: "$4,900" },
            ].map((c, i) => (
              <div key={i} className="flex justify-between items-center text-sm">
                <div>
                  <span className="text-gray-200">{c.name}</span>
                  <span className="ml-2 text-xs text-gray-500">{c.date}</span>
                </div>
                <div className="flex items-center gap-3">
                  <span className={`text-xs px-1.5 py-0.5 rounded ${c.plan === "Enterprise" ? "bg-purple-500/20 text-purple-400" : c.plan === "Pro" ? "bg-emerald-500/20 text-emerald-400" : "bg-gray-700 text-gray-400"}`}>{c.plan}</span>
                  <span className="text-gray-400 w-16 text-right">{c.amount}</span>
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="p-6 border border-gray-800 rounded-xl">
          <h2 className="font-semibold mb-4">Licencias por expirar</h2>
          <div className="space-y-3">
            {[
              { name: "GlobalTech", tier: "Enterprise", days: 3, users: 48 },
              { name: "StartupX", tier: "Pro", days: 5, users: 12 },
              { name: "DevStudio", tier: "Pro", days: 7, users: 3 },
              { name: "EduPlatform", tier: "Enterprise", days: 10, users: 85 },
            ].map((l, i) => (
              <div key={i} className="flex justify-between items-center text-sm">
                <span className="text-gray-200">{l.name}</span>
                <div className="flex items-center gap-3">
                  <span className="text-xs text-gray-500">{l.users} usuarios</span>
                  <span className={`text-xs px-1.5 py-0.5 rounded ${l.days <= 3 ? "bg-red-500/20 text-red-400" : l.days <= 7 ? "bg-yellow-500/20 text-yellow-400" : "bg-gray-700 text-gray-400"}`}>{l.days}d</span>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
