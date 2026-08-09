"use client";

import { useState } from "react";
import api from "@/lib/api";

export default function AdminLoginPage() {
  const [email, setEmail] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError("");

    try {
      const response = await api.login(email);
      if (response.magic_link) {
        const token = new URL(response.magic_link).searchParams.get("token");
        if (token) {
          const auth = await api.verifyToken(token);
          api.saveToken(auth.access_token);
          window.location.href = "/admin";
          return;
        }
      }
      setError("Revisá tu email para el enlace mágico.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Error de autenticación");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-gray-950 flex items-center justify-center p-6">
      <div className="max-w-sm w-full">
        <div className="text-center mb-8">
          <h1 className="text-2xl font-bold">OVAV Admin</h1>
          <p className="text-gray-500 text-sm mt-1">Acceso restringido al equipo OVAV</p>
        </div>
        {error && (
          <div className="mb-4 p-3 border border-red-500/30 bg-red-500/10 rounded-lg text-sm text-red-400">{error}</div>
        )}
        <form onSubmit={handleLogin}>
          <input type="email" placeholder="admin@ovav.dev" value={email} onChange={(e) => setEmail(e.target.value)} required className="w-full px-4 py-3 bg-gray-900 border border-gray-700 rounded-lg text-gray-100 placeholder-gray-500 focus:border-emerald-500 focus:outline-none mb-4" />
          <button type="submit" disabled={!email || loading} className="w-full py-3 bg-emerald-500 hover:bg-emerald-400 disabled:bg-gray-700 disabled:text-gray-500 text-gray-950 font-semibold rounded-lg transition flex items-center justify-center gap-2">
            {loading ? <><span className="animate-spin">⚙️</span> Enviando...</> : "Ingresar al panel"}
          </button>
        </form>
        <a href="/" className="block text-center text-sm text-gray-600 hover:text-gray-400 mt-6 transition">← Volver al sitio</a>
      </div>
    </div>
  );
}
