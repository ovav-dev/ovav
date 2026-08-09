"use client";

import type { Metadata } from "next";
import { AuthProvider } from "@/lib/auth";
import Link from "next/link";
import { usePathname } from "next/navigation";
import "./globals.css";

export default function RootLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();

  return (
    <html lang="es" className="dark">
      <body className="bg-[#030712] text-gray-100 min-h-screen antialiased">
        {/* Navbar */}
        <header className="fixed top-0 left-0 right-0 z-50 border-b border-white/[0.04] bg-[#030712]/80 backdrop-blur-xl">
          <nav className="max-w-5xl mx-auto px-6 h-16 flex items-center justify-between">
            <Link href="/" className="flex items-center gap-2 group">
              <div className="w-8 h-8 rounded-lg bg-emerald-500/10 border border-emerald-500/30 flex items-center justify-center group-hover:bg-emerald-500/20 transition-colors">
                <svg className="w-4 h-4 text-emerald-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5" strokeLinecap="round" strokeLinejoin="round" />
                </svg>
              </div>
              <span className="text-lg font-bold tracking-tight">OVAV</span>
            </Link>

            <div className="hidden md:flex items-center gap-1">
              {[
                { href: "/docs", label: "Docs" },
                { href: "/#pricing", label: "Planes" },
              ].map((item) => (
                <Link
                  key={item.href}
                  href={item.href}
                  className={`px-3 py-1.5 rounded-lg text-sm transition-colors ${
                    pathname === item.href
                      ? "text-emerald-400 bg-emerald-500/5"
                      : "text-gray-400 hover:text-gray-200 hover:bg-white/[0.03]"
                  }`}
                >
                  {item.label}
                </Link>
              ))}
            </div>

            <div className="flex items-center gap-3">
              <Link
                href="/login"
                className="text-sm text-gray-400 hover:text-gray-200 transition-colors hidden sm:block"
              >
                Iniciar sesión
              </Link>
              <Link
                href="/checkout?tier=pro"
                className="px-4 py-2 bg-emerald-500 hover:bg-emerald-400 text-gray-950 text-sm font-semibold rounded-lg transition-colors shadow-lg shadow-emerald-500/25 hover:shadow-emerald-500/40"
              >
                Comenzar
              </Link>
            </div>
          </nav>
        </header>

        {/* Spacer for fixed navbar */}
        <div className="h-16" />

        <AuthProvider>{children}</AuthProvider>
      </body>
    </html>
  );
}
