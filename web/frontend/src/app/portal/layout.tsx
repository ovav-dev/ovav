"use client";

import { usePathname } from "next/navigation";
import Link from "next/link";
import api from "@/lib/api";
import { useEffect, useState } from "react";

const NAV_ITEMS = [
  { href: "/portal/profile", label: "Perfil", icon: "👤" },
  { href: "/portal/subscriptions", label: "Suscripciones", icon: "📋" },
  { href: "/portal/invoices", label: "Facturación", icon: "🧾" },
];

export default function PortalLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const [ready, setReady] = useState(false);

  useEffect(() => {
    if (!api.isAuthenticated()) {
      window.location.href = "/login";
      return;
    }
    setReady(true);
  }, []);

  if (!ready) return <div className="min-h-screen bg-gray-950 flex items-center justify-center"><div className="animate-spin text-3xl">⚙️</div></div>;

  return (
    <div className="min-h-screen bg-gray-950">
      <nav className="border-b border-gray-800 px-6 py-4 flex items-center justify-between max-w-6xl mx-auto">
        <div className="flex items-center gap-8">
          <Link href="/dashboard" className="text-xl font-bold tracking-tight">OVAV</Link>
          <div className="flex gap-6 text-sm">
            {NAV_ITEMS.map((item) => (
              <Link key={item.href} href={item.href} className={`transition ${pathname === item.href ? "text-emerald-400" : "text-gray-400 hover:text-gray-200"}`}>
                {item.icon} {item.label}
              </Link>
            ))}
          </div>
        </div>
        <button onClick={() => { api.clearToken(); window.location.href = "/"; }} className="text-sm text-gray-600 hover:text-gray-400">Salir</button>
      </nav>
      <main>{children}</main>
    </div>
  );
}
