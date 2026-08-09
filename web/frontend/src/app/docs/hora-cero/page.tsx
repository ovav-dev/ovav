import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Guía Hora Cero — OVAV Docs",
  description: "Dominá OVAV en tu primer proyecto en menos de 60 minutos.",
};

const STEPS = [
  {
    number: "1",
    title: "Instalación",
    time: "5 min",
    content: (
      <>
        <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm">
          <code>ovav init{"\n"}ovav status</code>
        </pre>
        <p className="text-gray-400 mt-4">
          Con dos comandos ya tenés OVAV funcionando en tu proyecto. Sin configuraciones
          complejas, sin dependencias externas.
        </p>
      </>
    ),
  },
  {
    number: "2",
    title: "Primer logro: backup gobernado",
    time: "15 min",
    content: (
      <>
        <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm">
          <code>ovav backup --create --consent --accept-risk{"\n"}ovav validate</code>
        </pre>
        <p className="text-gray-400 mt-4">
          Creá tu primer backup. OVAV verifica que todos los archivos estén íntegros antes
          de empaquetar. El comando <code className="text-emerald-400">validate</code> corre
          los validators para confirmar que todo está en orden.
        </p>
      </>
    ),
  },
  {
    number: "3",
    title: "Boundary Law en acción",
    time: "20 min",
    content: (
      <>
        <pre className="bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm">
          <code>ovav boundary set --branch main --action deny-push{"\n"}git push origin main{"\n"}# ❌ Bloqueado: rama protegida</code>
        </pre>
        <p className="text-gray-400 mt-4">
          Acá ves Boundary Law en vivo. Definiste que <code className="text-emerald-400">main</code> es
          intocable. OVAV bloquea el push automáticamente. Ningún agente puede saltarse esta regla.
        </p>
      </>
    ),
  },
  {
    number: "4",
    title: "Output Guard",
    time: "10 min",
    content: (
      <>
        <p className="text-gray-400">
          Cada respuesta que recibís de un agente está firmada criptográficamente por OVAV.
          Si alguien intenta modificar un output —ya sea un agente, un script, o un atacante— 
          OVAV lo detecta instantáneamente y bloquea la entrega.
        </p>
        <p className="text-gray-400 mt-3">
          Esto no es teoría. El Output Guard corre en cada respuesta, verificando integridad
          con HMAC-SHA256. No se puede falsificar porque el secreto nunca sale de tu máquina.
        </p>
      </>
    ),
  },
  {
    number: "5",
    title: "Próximos pasos",
    time: "10 min",
    content: (
      <>
        <ul className="space-y-2 text-gray-400">
          <li>
            <a href="/docs/validators" className="text-emerald-400 hover:text-emerald-300">
              Validators avanzados
            </a>{" "}
            — Configurá reglas específicas para tu stack
          </li>
          <li>
            <a href="/docs/connector-bus" className="text-emerald-400 hover:text-emerald-300">
              ConnectorBus
            </a>{" "}
            — Extendé OVAV con tus propias herramientas
          </li>
          <li>
            <a href="/docs/cli" className="text-emerald-400 hover:text-emerald-300">
              CLI Reference
            </a>{" "}
            — Todos los comandos documentados
          </li>
          <li>
            <a href="/docs/api" className="text-emerald-400 hover:text-emerald-300">
              API Reference
            </a>{" "}
            — Integrá OVAV en tu pipeline
          </li>
        </ul>
      </>
    ),
  },
];

export default function HoraCeroPage() {
  return (
    <main className="max-w-3xl mx-auto px-6 py-12">
      <a
        href="/docs"
        className="text-sm text-gray-400 hover:text-emerald-400 mb-6 inline-block transition"
      >
        ← Docs
      </a>

      <div className="mb-12">
        <h1 className="text-3xl font-bold mb-3">Guía Hora Cero</h1>
        <p className="text-lg text-gray-400">
          Dominá OVAV en tu primer proyecto. Menos de 60 minutos. Sin experiencia previa.
        </p>
      </div>

      <div className="space-y-12">
        {STEPS.map((step) => (
          <section key={step.number} className="relative pl-12">
            <div className="absolute left-0 top-0 w-8 h-8 rounded-full bg-emerald-500/20 border border-emerald-500/50 flex items-center justify-center text-sm font-bold text-emerald-400">
              {step.number}
            </div>
            <div className="flex items-baseline gap-3 mb-4">
              <h2 className="text-xl font-semibold">{step.title}</h2>
              <span className="text-xs text-gray-500 bg-gray-800 px-2 py-0.5 rounded">
                {step.time}
              </span>
            </div>
            {step.content}
          </section>
        ))}
      </div>

      <div className="mt-16 p-6 border border-gray-800 rounded-xl bg-gray-900/50">
        <h2 className="font-semibold mb-2">¿Algo no funcionó?</h2>
        <p className="text-sm text-gray-400">
          Si encontraste un problema o tenés una sugerencia,{" "}
          <a
            href="https://github.com/ovav-dev/ovav/issues"
            className="text-emerald-400 hover:text-emerald-300"
            target="_blank"
            rel="noopener noreferrer"
          >
            reportalo en GitHub
          </a>
          . El equipo de OVAV revisa cada issue.
        </p>
      </div>
    </main>
  );
}
