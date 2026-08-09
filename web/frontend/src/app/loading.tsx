export default function Loading() {
  return (
    <main className="min-h-screen bg-gray-950 flex items-center justify-center">
      <div className="text-center">
        <div className="animate-spin text-4xl mb-4">⚙️</div>
        <p className="text-gray-500 text-sm">Cargando...</p>
      </div>
    </main>
  );
}
