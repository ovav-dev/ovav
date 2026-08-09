import type { Metadata } from "next";

export const metadata: Metadata = { title: "Configuración — Admin OVAV" };

export default function AdminSettingsPage() {
  return (
    <div className="p-8 max-w-2xl">
      <h1 className="text-2xl font-bold mb-6">Configuración</h1>

      <section className="mb-8">
        <h2 className="font-semibold mb-4">General</h2>
        <div className="space-y-4">
          <div>
            <label className="text-sm text-gray-400 block mb-1">Nombre del workspace</label>
            <input type="text" defaultValue="OVAV Admin" className="w-full px-3 py-2 bg-gray-900 border border-gray-700 rounded-lg text-gray-300 text-sm focus:border-emerald-500 focus:outline-none" />
          </div>
          <div>
            <label className="text-sm text-gray-400 block mb-1">URL pública</label>
            <input type="text" defaultValue="https://ovav.dev" className="w-full px-3 py-2 bg-gray-900 border border-gray-700 rounded-lg text-gray-300 text-sm focus:border-emerald-500 focus:outline-none" />
          </div>
        </div>
      </section>

      <section className="mb-8">
        <h2 className="font-semibold mb-4">Stripe</h2>
        <div className="space-y-4">
          <div>
            <label className="text-sm text-gray-400 block mb-1">Secret Key</label>
            <input type="password" defaultValue="sk_live_••••••••••••••••" className="w-full px-3 py-2 bg-gray-900 border border-gray-700 rounded-lg text-gray-300 text-sm focus:border-emerald-500 focus:outline-none font-mono" />
          </div>
          <div>
            <label className="text-sm text-gray-400 block mb-1">Webhook Secret</label>
            <input type="password" defaultValue="whsec_••••••••••••••••" className="w-full px-3 py-2 bg-gray-900 border border-gray-700 rounded-lg text-gray-300 text-sm focus:border-emerald-500 focus:outline-none font-mono" />
          </div>
        </div>
      </section>

      <section className="mb-8">
        <h2 className="font-semibold mb-4">Notificaciones</h2>
        <div className="space-y-3">
          {[
            { label: "Nuevo cliente", desc: "Cuando alguien se registra", enabled: true },
            { label: "Churn alert", desc: "Cuando una licencia está por expirar", enabled: true },
            { label: "Pago fallido", desc: "Cuando un cobro es rechazado", enabled: true },
            { label: "Reporte semanal", desc: "Resumen de métricas cada lunes", enabled: false },
          ].map((n) => (
            <div key={n.label} className="flex items-center justify-between p-3 border border-gray-800 rounded-xl">
              <div>
                <p className="text-sm text-gray-200">{n.label}</p>
                <p className="text-xs text-gray-500">{n.desc}</p>
              </div>
              <button className={`w-10 h-6 rounded-full transition ${n.enabled ? "bg-emerald-500" : "bg-gray-700"}`}>
                <div className={`w-4 h-4 rounded-full bg-white transition mx-0.5 ${n.enabled ? "ml-auto" : ""}`} />
              </button>
            </div>
          ))}
        </div>
      </section>

      <button className="px-6 py-2.5 bg-emerald-500 hover:bg-emerald-400 text-gray-950 rounded-lg font-medium transition">Guardar cambios</button>
    </div>
  );
}
