export default function InstallDocPage() {
  return (
    <main className="max-w-3xl mx-auto px-6 py-12">
      <a href="/docs" className="text-sm text-gray-400 hover:text-emerald-400 mb-6 inline-block">← Docs</a>
      <h1 className="text-3xl font-bold mb-6">Instalación</h1>
      <div className="prose prose-invert max-w-none">
        <h2>One-command install</h2>
        <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto">
          <code>curl -sSL https://ovav.dev/install | bash</code>
        </pre>
        <h2>Requisitos</h2>
        <ul>
          <li>Linux o macOS (WSL en Windows)</li>
          <li>Python 3.11+</li>
          <li>Git</li>
        </ul>
        <h2>Verificar instalación</h2>
        <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto">
          <code>ovav --version</code>
        </pre>
      </div>
    </main>
  );
}
