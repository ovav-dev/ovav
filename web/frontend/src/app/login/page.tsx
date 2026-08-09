"use client";

import { useState } from "react";
import api from "@/lib/api";

export default function LoginPage() {
  const [email, setEmail] = useState("");
  const [loading, setLoading] = useState(false);
  const [sent, setSent] = useState(false);
  const [error, setError] = useState("");

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!email) return;

    setLoading(true);
    setError("");

    try {
      const result = await api.login(email);

      // In development, we get the magic_link back directly
      if (result.magic_link) {
        const token = new URL(result.magic_link).searchParams.get("token");
        if (token) {
          const auth = await api.verifyToken(token);
          api.saveToken(auth.access_token);
          window.location.href = "/dashboard";
          return;
        }
      }

      setSent(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Error al enviar el enlace");
    } finally {
      setLoading(false);
    }
  };

  const handleOAuth = async (provider: "google" | "github") => {
    try {
      const { url } = await api.oauthLogin(provider);
      window.location.href = url;
    } catch (err) {
      setError(err instanceof Error ? err.message : "Error al iniciar OAuth");
    }
  };

  if (sent) {
    return (
      <main className="max-w-md mx-auto px-6 py-32 text-center">
        <div className="text-4xl mb-6">📧</div>
        <h1 className="text-2xl font-bold mb-2">Revisá tu email</h1>
        <p className="text-gray-400">
          Enviamos un enlace mágico a <strong>{email}</strong>.
          Hacé clic en el enlace para iniciar sesión.
        </p>
        <button
          onClick={() => setSent(false)}
          className="inline-block mt-8 text-emerald-400 hover:text-emerald-300 text-sm"
        >
          ← Usar otro email
        </button>
      </main>
    );
  }

  return (
    <main className="max-w-md mx-auto px-6 py-32">
      <h1 className="text-3xl font-bold mb-2">Iniciar sesión</h1>
      <p className="text-gray-400 mb-8">Sin contraseñas. Solo tu email.</p>

      {error && (
        <div className="mb-4 p-3 border border-red-500/30 bg-red-500/10 rounded-lg text-sm text-red-400">
          {error}
        </div>
      )}

      <form onSubmit={handleLogin}>
        <input
          type="email"
          placeholder="tu@email.com"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
          className="w-full px-4 py-3 bg-gray-900 border border-gray-700 rounded-lg text-gray-100 placeholder-gray-500 focus:border-emerald-500 focus:outline-none mb-4"
        />
        <button
          type="submit"
          disabled={!email || loading}
          className="w-full py-3 bg-emerald-500 hover:bg-emerald-400 disabled:bg-gray-700 disabled:text-gray-500 text-gray-950 font-semibold rounded-lg transition flex items-center justify-center gap-2"
        >
          {loading ? (
            <>
              <span className="animate-spin">⚙️</span>
              Enviando...
            </>
          ) : (
            "Enviar enlace mágico"
          )}
        </button>
      </form>

      <div className="mt-6 pt-6 border-t border-gray-800">
        <p className="text-sm text-gray-500 text-center mb-4">O continuar con</p>
        <div className="flex gap-3">
          <button
            onClick={() => handleOAuth("google")}
            className="flex-1 py-2.5 border border-gray-700 hover:border-gray-500 rounded-lg text-sm transition"
          >
            Google
          </button>
          <button
            onClick={() => handleOAuth("github")}
            className="flex-1 py-2.5 border border-gray-700 hover:border-gray-500 rounded-lg text-sm transition"
          >
            GitHub
          </button>
        </div>
      </div>
    </main>
  );
}
