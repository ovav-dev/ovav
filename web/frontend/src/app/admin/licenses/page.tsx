import type { Metadata } from "next";

export const metadata: Metadata = { title: "Licencias — Admin OVAV" };

const LICENSES = [
  { key: "ovav-ent-a1b2c3d4...", owner: "Acme Corp", tier: "Enterprise", status: "active", instances: "48/100", expires: "2027-01-15" },
  { key: "ovav-pro-x9y8z7w6...", owner: "Maria Lopez", tier: "Pro", status: "active", instances: "1/3", expires: "2026-09-22" },
  { key: "ovav-pro-q1w2e3r4...", owner: "TechStart Inc", tier: "Pro", status: "trial", instances: "3/3", expires: "2026-06-24" },
  { key: "ovav-ent-t5y6u7i8...", owner: "DataFlow SA", tier: "Enterprise", status: "active", instances: "100/100", expires: "2027-02-01" },
  { key: "ovav-core-m9n8b7v6...", owner: "Carlos Ruiz", tier: "Core", status: "active", instances: "1/1", expires: "—" },
  { key: "ovav-ent-z1x2c3v4...", owner: "GlobalTech", tier: "Enterprise", status: "grace", instances: "85/200", expires: "2026-06-03" },
];

export default function AdminLicensesPage() {
  return (
    <div className="p-8">
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold">Licencias</h1>
          <p className="text-gray-500 text-sm mt-1">{LICENSES.length} licencias registradas</p>
        </div>
        <div className="flex gap-3">
          <input type="search" placeholder="Buscar por key o cliente..." className="px-3 py-1.5 bg-gray-900 border border-gray-700 rounded-lg text-sm text-gray-300 placeholder-gray-500 focus:border-emerald-500 focus:outline-none w-64 font-mono" />
          <button className="px-4 py-1.5 border border-red-500/30 text-red-400 hover:bg-red-500/10 rounded-lg text-sm font-medium transition">Revocar múltiples</button>
        </div>
      </div>

      <div className="border border-gray-800 rounded-xl overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-gray-800 bg-gray-900/50 text-left text-gray-500">
              <th className="py-3 px-4 font-medium">License Key</th>
              <th className="py-3 px-4 font-medium">Titular</th>
              <th className="py-3 px-4 font-medium">Tier</th>
              <th className="py-3 px-4 font-medium">Estado</th>
              <th className="py-3 px-4 font-medium">Instancias</th>
              <th className="py-3 px-4 font-medium">Expira</th>
              <th className="py-3 px-4 font-medium"></th>
            </tr>
          </thead>
          <tbody>
            {LICENSES.map((l, i) => (
              <tr key={i} className="border-b border-gray-800/50 hover:bg-gray-900/30">
                <td className="py-3 px-4 font-mono text-xs text-gray-300">{l.key}</td>
                <td className="py-3 px-4 text-gray-200">{l.owner}</td>
                <td className="py-3 px-4">
                  <span className={`text-xs px-1.5 py-0.5 rounded ${l.tier === "Enterprise" ? "bg-purple-500/20 text-purple-400" : l.tier === "Pro" ? "bg-emerald-500/20 text-emerald-400" : "bg-gray-700 text-gray-400"}`}>{l.tier}</span>
                </td>
                <td className="py-3 px-4">
                  <span className={`text-xs px-1.5 py-0.5 rounded ${l.status === "active" ? "bg-emerald-500/20 text-emerald-400" : l.status === "trial" ? "bg-blue-500/20 text-blue-400" : l.status === "grace" ? "bg-yellow-500/20 text-yellow-400" : "bg-red-500/20 text-red-400"}`}>{l.status}</span>
                </td>
                <td className="py-3 px-4 text-gray-400 font-mono">{l.instances}</td>
                <td className="py-3 px-4 text-gray-500">{l.expires}</td>
                <td className="py-3 px-4">
                  <button className="text-xs text-red-400 hover:text-red-300">Revocar</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
