import type { Metadata } from "next";

export const metadata: Metadata = { title: "Ingresos — Admin OVAV" };

export default function AdminRevenuePage() {
  return (
    <div className="p-8">
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold">Ingresos</h1>
          <p className="text-gray-500 text-sm mt-1">Métricas financieras en tiempo real.</p>
        </div>
        <select className="px-3 py-1.5 bg-gray-900 border border-gray-700 rounded-lg text-sm text-gray-300">
          <option>Últimos 12 meses</option>
          <option>Últimos 6 meses</option>
          <option>Últimos 3 meses</option>
        </select>
      </div>

      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
        {[
          { label: "MRR", value: "$24,830", change: "+8.3%" },
          { label: "ARR", value: "$297,960", change: "+12.1%" },
          { label: "LTV", value: "$1,247", change: "+5.7%" },
          { label: "CAC", value: "$89", change: "-15.2%" },
        ].map((m) => (
          <div key={m.label} className="p-5 border border-gray-800 rounded-xl">
            <p className="text-xs text-gray-500 mb-1">{m.label}</p>
            <div className="flex items-baseline gap-2">
              <span className="text-2xl font-bold font-mono">{m.value}</span>
              <span className="text-xs text-emerald-400">{m.change}</span>
            </div>
          </div>
        ))}
      </div>

      <div className="grid lg:grid-cols-2 gap-6 mb-8">
        <div className="p-6 border border-gray-800 rounded-xl">
          <h2 className="font-semibold mb-4">MRR Breakdown</h2>
          <div className="space-y-3">
            {[
              { label: "Nuevo MRR", value: "$3,240", color: "bg-emerald-500" },
              { label: "Expansión MRR", value: "$1,890", color: "bg-blue-500" },
              { label: "Contracción MRR", value: "-$620", color: "bg-yellow-500" },
              { label: "Churn MRR", value: "-$510", color: "bg-red-500" },
            ].map((b) => (
              <div key={b.label} className="flex items-center gap-3">
                <div className={`w-3 h-3 rounded ${b.color}`} />
                <span className="text-sm text-gray-400 flex-1">{b.label}</span>
                <span className="text-sm font-mono text-gray-200">{b.value}</span>
              </div>
            ))}
          </div>
        </div>

        <div className="p-6 border border-gray-800 rounded-xl">
          <h2 className="font-semibold mb-4">Churn por cohorte</h2>
          <div className="space-y-3">
            {[
              { cohort: "Ene 2026", size: 89, churn: "1.8%" },
              { cohort: "Feb 2026", size: 112, churn: "2.1%" },
              { cohort: "Mar 2026", size: 145, churn: "2.4%" },
              { cohort: "Abr 2026", size: 167, churn: "1.9%" },
              { cohort: "May 2026", size: 203, churn: "2.2%" },
            ].map((c) => (
              <div key={c.cohort} className="flex justify-between items-center text-sm">
                <span className="text-gray-300">{c.cohort}</span>
                <div className="flex items-center gap-4">
                  <span className="text-gray-500">{c.size} clientes</span>
                  <span className="text-red-400 font-mono">{c.churn}</span>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      <div className="flex gap-3">
        <button className="px-4 py-2 border border-gray-700 hover:border-gray-500 rounded-lg text-sm transition">Exportar CSV</button>
        <button className="px-4 py-2 border border-gray-700 hover:border-gray-500 rounded-lg text-sm transition">Exportar JSON</button>
      </div>
    </div>
  );
}
