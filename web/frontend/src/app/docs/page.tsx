const SECTIONS = [
  {
    title: "Primeros pasos",
    pages: [
      { href: "/docs/install", title: "Instalación" },
      { href: "/docs/quickstart", title: "Quickstart" },
      { href: "/docs/hora-cero", title: "Guía Hora Cero" },
    ],
  },
  {
    title: "Conceptos",
    pages: [
      { href: "/docs/boundary-law", title: "Boundary Law" },
      { href: "/docs/output-guard", title: "Output Guard" },
      { href: "/docs/session-capsule", title: "Session Capsule" },
      { href: "/docs/connector-bus", title: "ConnectorBus" },
    ],
  },
  {
    title: "Guías",
    pages: [
      { href: "/docs/validators", title: "Validators" },
      { href: "/docs/backup", title: "Backup gobernado" },
      { href: "/docs/enterprise", title: "Deploy enterprise" },
    ],
  },
  {
    title: "Referencia",
    pages: [
      { href: "/docs/cli", title: "CLI Reference" },
      { href: "/docs/api", title: "API Reference" },
      { href: "/docs/security", title: "Seguridad" },
    ],
  },
];

export default function DocsPage() {
  return (
    <main className="max-w-4xl mx-auto px-6 py-12">
      <h1 className="text-3xl font-bold mb-2">Documentación</h1>
      <p className="text-gray-400 mb-10">
        Todo lo que necesitás para dominar OVAV.
      </p>

      <div className="grid md:grid-cols-2 gap-8">
        {SECTIONS.map((section) => (
          <div key={section.title}>
            <h2 className="font-semibold text-sm text-gray-400 uppercase tracking-wide mb-3">
              {section.title}
            </h2>
            <ul className="space-y-2">
              {section.pages.map((page) => (
                <li key={page.href}>
                  <a
                    href={page.href}
                    className="text-gray-300 hover:text-emerald-400 transition text-sm"
                  >
                    {page.title}
                  </a>
                </li>
              ))}
            </ul>
          </div>
        ))}
      </div>

      <div className="mt-12 p-6 border border-emerald-500/20 bg-emerald-500/5 rounded-xl">
        <h2 className="font-semibold mb-2">¿No encontrás lo que buscás?</h2>
        <p className="text-sm text-gray-400">
          Escribinos a{" "}
          <a href="mailto:docs@ovav.dev" className="text-emerald-400">
            docs@ovav.dev
          </a>{" "}
          o unite a nuestro{" "}
          <a href="#" className="text-emerald-400">
            Discord
          </a>
          .
        </p>
      </div>
    </main>
  );
}
