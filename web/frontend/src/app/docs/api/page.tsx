import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "API Reference — OVAV Docs",
  description: "API REST de OVAV para integración y automatización.",
};

const ENDPOINTS = [
  { method: "GET", path: "/health", desc: "Health check público", auth: "Ninguna" },
  { method: "POST", path: "/auth/register", desc: "Registrar usuario", auth: "Ninguna" },
  { method: "POST", path: "/auth/login", desc: "Solicitar magic link", auth: "Ninguna" },
  { method: "GET", path: "/auth/verify", desc: "Verificar token de magic link", auth: "Token" },
  { method: "GET", path: "/auth/oauth/{provider}", desc: "Iniciar OAuth flow", auth: "Ninguna" },
  { method: "GET", path: "/auth/session", desc: "Obtener sesión actual", auth: "JWT" },
  { method: "POST", path: "/checkout/session", desc: "Crear sesión de Stripe", auth: "Ninguna" },
  { method: "POST", path: "/licenses/validate", desc: "Validar license key (CLI)", auth: "API Key" },
  { method: "GET", path: "/licenses", desc: "Listar licencias del usuario", auth: "JWT" },
  { method: "GET", path: "/licenses/{id}", desc: "Detalle de licencia", auth: "JWT" },
  { method: "POST", path: "/licenses/{id}/revoke", desc: "Revocar licencia propia", auth: "JWT" },
];

export default function APIPage() {
  return (
    <main className="max-w-3xl mx-auto px-6 py-12">
      <a href="/docs" className="text-sm text-gray-400 hover:text-emerald-400 mb-6 inline-block transition">← Docs</a>
      <h1 className="text-3xl font-bold mb-3">API Reference</h1>
      <p className="text-lg text-gray-400 mb-10">API REST completa. Base URL: <code className="text-emerald-400">https://api.ovav.dev</code></p>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">Endpoints</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-left text-gray-500">
                <th className="py-2 pr-4 font-medium">Método</th>
                <th className="py-2 pr-4 font-medium">Path</th>
                <th className="py-2 pr-4 font-medium">Descripción</th>
                <th className="py-2 font-medium">Auth</th>
              </tr>
            </thead>
            <tbody>
              {ENDPOINTS.map((e) => (
                <tr key={e.method + e.path} className="border-b border-gray-800/50">
                  <td className="py-2 pr-4">
                    <span className={`text-xs px-1.5 py-0.5 rounded font-mono ${
                      e.method === "GET" ? "bg-blue-500/20 text-blue-400" :
                      e.method === "POST" ? "bg-emerald-500/20 text-emerald-400" :
                      "bg-gray-700 text-gray-400"
                    }`}>{e.method}</span>
                  </td>
                  <td className="py-2 pr-4 font-mono text-gray-300">{e.path}</td>
                  <td className="py-2 pr-4 text-gray-400">{e.desc}</td>
                  <td className="py-2 text-xs text-gray-500">{e.auth}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">Autenticación</h2>
        <div className="space-y-3">
          <div className="p-4 border border-gray-800 rounded-xl">
            <h3 className="font-medium text-sm mb-1">JWT (usuarios web)</h3>
            <pre className="bg-gray-900 p-3 rounded-lg overflow-x-auto text-xs"><code>Authorization: Bearer eyJhbGciOiJSUzI1NiIs...</code></pre>
          </div>
          <div className="p-4 border border-gray-800 rounded-xl">
            <h3 className="font-medium text-sm mb-1">API Key (CLI → servidor de licencias)</h3>
            <pre className="bg-gray-900 p-3 rounded-lg overflow-x-auto text-xs"><code>X-API-Key: ovav-pro-xxxxxxxxxxxx</code></pre>
          </div>
        </div>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">Rate Limits</h2>
        <div className="grid grid-cols-2 gap-3">
          {[
            { tier: "Core", limit: "60 req/min" },
            { tier: "Pro", limit: "300 req/min" },
            { tier: "Enterprise", limit: "1000 req/min" },
          ].map((r) => (
            <div key={r.tier} className="p-3 border border-gray-800 rounded-xl text-center">
              <div className="text-sm font-medium">{r.tier}</div>
              <div className="text-xs text-gray-500 mt-0.5">{r.limit}</div>
            </div>
          ))}
        </div>
      </section>
    </main>
  );
}
