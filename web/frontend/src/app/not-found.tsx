import Link from "next/link";
import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "404 — Página no encontrada | OVAV",
};

export default function NotFound() {
  return (
    <main className="min-h-screen bg-gray-950 flex items-center justify-center px-6">
      <div className="text-center max-w-md">
        <div className="text-6xl mb-6">🔍</div>
        <h1 className="text-4xl font-bold mb-3">404</h1>
        <p className="text-lg text-gray-400 mb-2">Página no encontrada</p>
        <p className="text-sm text-gray-600 mb-8">
          La página que buscás no existe o fue movida.
          Si llegaste desde un enlace interno, reportalo.
        </p>
        <div className="flex gap-3 justify-center">
          <Link
            href="/"
            className="px-6 py-2.5 bg-emerald-500 hover:bg-emerald-400 text-gray-950 rounded-lg font-medium transition"
          >
            Ir al inicio
          </Link>
          <Link
            href="/docs"
            className="px-6 py-2.5 border border-gray-700 hover:border-gray-500 rounded-lg transition"
          >
            Documentación
          </Link>
        </div>
      </div>
    </main>
  );
}
