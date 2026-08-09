"use client";

import { useEffect, useState } from "react";
import { useSearchParams, useParams } from "next/navigation";
import api from "@/lib/api";

export default function OAuthCallbackPage() {
  const searchParams = useSearchParams();
  const params = useParams<{ provider: string }>();
  const [status, setStatus] = useState<"processing" | "success" | "error">("processing");
  const [message, setMessage] = useState("Conectando con tu cuenta...");

  useEffect(() => {
    const code = searchParams.get("code");
    const provider = params.provider as "google" | "github";

    if (!code) {
      setStatus("error");
      setMessage("Código de autorización no encontrado.");
      return;
    }

    if (provider !== "google" && provider !== "github") {
      setStatus("error");
      setMessage("Proveedor no soportado.");
      return;
    }

    (async () => {
      try {
        const auth = await api.oauthCallback(provider, code);
        api.saveToken(auth.access_token);
        setStatus("success");
        setMessage(`Conectado con ${provider}. Redirigiendo...`);
        setTimeout(() => {
          window.location.href = "/dashboard";
        }, 1000);
      } catch (err) {
        setStatus("error");
        setMessage(err instanceof Error ? err.message : "Error en la autenticación.");
      }
    })();
  }, [searchParams, params]);

  return (
    <main className="max-w-md mx-auto px-6 py-32 text-center">
      {status === "processing" && (
        <>
          <div className="animate-spin text-4xl mb-6">⚙️</div>
          <h1 className="text-2xl font-bold mb-2">Autenticando</h1>
          <p className="text-gray-400">{message}</p>
        </>
      )}
      {status === "success" && (
        <>
          <div className="text-4xl mb-6">✅</div>
          <h1 className="text-2xl font-bold mb-2">¡Conectado!</h1>
          <p className="text-gray-400">{message}</p>
        </>
      )}
      {status === "error" && (
        <>
          <div className="text-4xl mb-6">❌</div>
          <h1 className="text-2xl font-bold mb-2">Error</h1>
          <p className="text-gray-400 mb-6">{message}</p>
          <a href="/login" className="text-emerald-400 hover:text-emerald-300">
            ← Volver a intentar
          </a>
        </>
      )}
    </main>
  );
}
