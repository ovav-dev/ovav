"use client";

import { usePathname } from "next/navigation";
import Link from "next/link";
import api from "@/lib/api";
import { useEffect, useState } from "react";

const NAV_ITEMS = [
  { href: "/admin", label: "Dashboard", icon: "📊" },
  { href: "/admin/customers", label: "Clientes", icon: "👥" },
  { href: "/admin/licenses", label: "Licencias", icon: "🔑" },
  { href: "/admin/revenue", label: "Ingresos", icon: "💰" },
  { href: "/admin/webhooks", label: "Webhooks", icon: "🔗" },
  { href: "/admin/alerts", label: "Alertas", icon: "🚨" },
  { href: "/admin/settings", label: "Configuración", icon: "⚙️" },
];

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const [authed, setAuthed] = useState(false);
  const [loading, setLoading] = useState(true);

  if (pathname === "/admin/login") return <>{children}</>;

  useEffect(() => {
    if (!api.isAuthenticated()) {
      window.location.href = "/admin/login";
      return;
    }
    setAuthed(true);
    setLoading(false);
  }, []);

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-950 flex items-center justify-center">
        <div className="animate-spin text-3xl">⚙️</div>
      </div>
    );
  }

  if (!authed) return null;

  return (
    <div className="min-h-screen bg-gray-950 flex">
      {/* Sidebar */}
      <aside className="w-64 border-r border-gray-800 p-6 flex flex-col">
        <Link href="/admin" className="text-xl font-bold tracking-tight mb-8">
          OVAV <span className="text-emerald-400 text-xs font-normal ml-1">Admin</span>
        </Link>
        <nav className="flex-1 space-y-1">
          {NAV_ITEMS.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              className={`flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition ${
                pathname === item.href
                  ? "bg-emerald-500/10 text-emerald-400 border border-emerald-500/30"
                  : "text-gray-400 hover:text-gray-200 hover:bg-gray-900"
              }`}
            >
              <span>{item.icon}</span>
              {item.label}
            </Link>
          ))}
        </nav>
        <button
          onClick={() => { api.clearToken(); window.location.href = "/"; }}
          className="text-xs text-gray-600 hover:text-gray-400 transition mt-4"
        >
          Cerrar sesión
        </button>
      </aside>
      <main className="flex-1 overflow-auto">{children}</main>
    </div>
  );
}
