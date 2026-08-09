"use client";

import { useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import api from "@/lib/api";

export default function VerifyPage() {
  const searchParams = useSearchParams();
  const [status, setStatus] = useState<"verifying" | "success" | "error">("verifying");
  const [message, setMessage] = useState("Verificando tu enlace mágico...");

  useEffect(() => {
    const token = searchParams.get("token");
    if (!token) {
      setStatus("error");
      setMessage("Token no encontrado en la URL.");
      return;
    }

    (async () => {
      try {
        const auth = await api.verifyToken(token);
        api.saveToken(auth.access_token);
        setStatus("success");
        setMessage("¡Sesión iniciada! Redirigiendo...");
        setTimeout(() => {
          window.location.href = "/dashboard";
        }, 1000);
      } catch (err) {
        setStatus("error");
        setMessage(err instanceof Error ? err.message : "Token inválido o expirado.");
      }
    })();
  }, [searchParams]);

  return (
    <main className="max-w-md mx-auto px-6 py-32 text-center">
      {status === "verifying" && (
        <>
          <div className="animate-spin text-4xl mb-6">⚙️</div>
          <h1 className="text-2xl font-bold mb-2">Verificando</h1>
          <p className="text-gray-400">{message}</p>
        </>
      )}
      {status === "success" && (
        <>
          <div className="text-4xl mb-6">✅</div>
          <h1 className="text-2xl font-bold mb-2">¡Listo!</h1>
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
