"use client";

export default function ErrorPage({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <main className="min-h-screen bg-gray-950 flex items-center justify-center px-6">
      <div className="text-center max-w-md">
        <div className="text-6xl mb-6">⚠️</div>
        <h1 className="text-4xl font-bold mb-3">Error</h1>
        <p className="text-lg text-gray-400 mb-2">Algo salió mal</p>
        <p className="text-sm text-gray-600 mb-2">
          {error.message || "Un error inesperado ocurrió."}
        </p>
        {error.digest && (
          <p className="text-xs text-gray-700 font-mono mb-6">
            Digest: {error.digest}
          </p>
        )}
        <div className="flex gap-3 justify-center">
          <button
            onClick={reset}
            className="px-6 py-2.5 bg-emerald-500 hover:bg-emerald-400 text-gray-950 rounded-lg font-medium transition"
          >
            Reintentar
          </button>
          <a
            href="/"
            className="px-6 py-2.5 border border-gray-700 hover:border-gray-500 rounded-lg transition"
          >
            Ir al inicio
          </a>
        </div>
      </div>
    </main>
  );
}
