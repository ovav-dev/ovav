"use client";

import { useState } from "react";
import { useSearchParams } from "next/navigation";
import api from "@/lib/api";

const PLANS: Record<string, { name: string; price: string; period: string; priceId: string; badge?: string }> = {
  pro_monthly: { name: "Pro Mensual", price: "$19", period: "/mes", priceId: "pro_monthly" },
  pro_annual: { name: "Pro Anual", price: "$15", period: "/mes (facturación anual)", priceId: "pro_annual", badge: "2 meses gratis" },
  enterprise: { name: "Enterprise", price: "$49", period: "/usuario/mes", priceId: "enterprise" },
};

export default function CheckoutPage() {
  const searchParams = useSearchParams();
  const defaultTier = searchParams.get("tier") || "pro_annual";
  const [selected, setSelected] = useState(defaultTier);
  const [email, setEmail] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleCheckout = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!email) return;

    setLoading(true);
    setError("");

    try {
      // If user has a token, pass it for authenticated checkout
      const token = api.getToken();

      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api"}/checkout/session`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            ...(token ? { Authorization: `Bearer ${token}` } : {}),
          },
          body: JSON.stringify({
            tier: selected,
            email,
            success_url: `${window.location.origin}/dashboard?session_id={CHECKOUT_SESSION_ID}`,
            cancel_url: `${window.location.origin}/checkout`,
          }),
        }
      );

      if (!response.ok) {
        const errBody = await response.json().catch(() => ({ detail: "Checkout failed" }));
        throw new Error(errBody.detail || "Error al crear la sesión de pago");
      }

      const data = await response.json();

      if (data.url) {
        // Redirect to Stripe Checkout
        window.location.href = data.url;
      } else if (data.session_id) {
        // In dev mode without Stripe, redirect to dashboard
        api.saveToken(data.access_token || "");
        window.location.href = `/dashboard?tier=${selected}&status=trial`;
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Error al procesar el pago");
    } finally {
      setLoading(false);
    }
  };

  return (
    <main className="max-w-2xl mx-auto px-6 py-20">
      <h1 className="text-3xl font-bold mb-2">Elegí tu plan</h1>
      <p className="text-gray-400 mb-10">14 días de prueba gratuita en Pro. Sin tarjeta.</p>

      {error && (
        <div className="mb-6 p-3 border border-red-500/30 bg-red-500/10 rounded-lg text-sm text-red-400">
          {error}
        </div>
      )}

      <form onSubmit={handleCheckout}>
        <div className="space-y-4 mb-8">
          {Object.entries(PLANS).map(([key, plan]) => (
            <button
              key={key}
              type="button"
              onClick={() => setSelected(key)}
              className={`w-full text-left p-6 rounded-xl border-2 transition ${
                selected === key
                  ? "border-emerald-500 bg-emerald-500/5"
                  : "border-gray-800 hover:border-gray-600"
              }`}
            >
              <div className="flex justify-between items-center">
                <div>
                  <span className="font-semibold text-lg">{plan.name}</span>
                  {plan.badge && (
                    <span className="ml-2 text-xs bg-emerald-500/20 text-emerald-400 px-2 py-0.5 rounded">
                      {plan.badge}
                    </span>
                  )}
                </div>
                <div className="text-right">
                  <span className="text-2xl font-bold">{plan.price}</span>
                  <span className="text-gray-400 text-sm">{plan.period}</span>
                </div>
              </div>
            </button>
          ))}
        </div>

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
              Procesando...
            </>
          ) : (
            "Comenzar prueba gratuita"
          )}
        </button>
        <p className="text-xs text-gray-600 mt-3 text-center">
          Sin cargo durante 14 días. Cancelá cuando quieras.
        </p>
      </form>
    </main>
  );
}
