import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Política de Privacidad — OVAV",
  description: "Cómo OVAV maneja tus datos. Privacidad real, no letra chica.",
};

export default function PrivacyPage() {
  return (
    <main className="max-w-3xl mx-auto px-6 py-12">
      <a href="/" className="text-sm text-gray-400 hover:text-emerald-400 mb-6 inline-block transition">
        ← Inicio
      </a>
      <h1 className="text-3xl font-bold mb-3">Política de Privacidad</h1>
      <p className="text-gray-400 mb-10">Última actualización: Junio 2026</p>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">1. Qué datos recolectamos</h2>
        <p className="text-gray-400 text-sm mb-3">
          OVAV es local-first. La mayoría de tus datos nunca salen de tu máquina. 
          Recolectamos solo lo mínimo necesario para operar:
        </p>
        <ul className="space-y-2 text-gray-400 text-sm ml-4">
          <li className="list-disc">Email — para crear tu cuenta y enviar magic links</li>
          <li className="list-disc">Datos de licencia — tier, estado, instancias activas</li>
          <li className="list-disc">Machine fingerprint (hashed) — para evitar abuso de licencias</li>
          <li className="list-disc">Datos de pago — procesados por Stripe, nunca almacenados por OVAV</li>
        </ul>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">2. Qué NO recolectamos</h2>
        <ul className="space-y-2 text-gray-400 text-sm ml-4">
          <li className="list-disc">Tu código fuente — nunca sale de tu máquina</li>
          <li className="list-disc">Tus prompts o conversaciones con agentes</li>
          <li className="list-disc">Tus archivos de configuración (.env, secretos)</li>
          <li className="list-disc">Tus datos de navegación o telemetría</li>
          <li className="list-disc">Tus datos de salud — se procesan 100% en local</li>
        </ul>
        <div className="mt-4 p-4 border border-emerald-500/20 bg-emerald-500/5 rounded-xl">
          <p className="text-sm text-gray-400">
            <span className="text-emerald-400 font-semibold">OVAV es local-first.</span> El
            governor, los validators, el Output Guard, y el procesamiento de datos corren en tu
            máquina. El servidor solo gestiona licencias y checkout.
          </p>
        </div>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">3. Cómo usamos tus datos</h2>
        <div className="space-y-3 text-gray-400 text-sm">
          <div className="p-3 border border-gray-800 rounded-xl">
            <h3 className="font-medium text-gray-300 mb-1">Autenticación</h3>
            <p>Tu email se usa para magic links y OAuth. Nunca para marketing sin tu consentimiento.</p>
          </div>
          <div className="p-3 border border-gray-800 rounded-xl">
            <h3 className="font-medium text-gray-300 mb-1">Licencias</h3>
            <p>Validamos tu licencia contra nuestro servidor. Solo transmitimos el hash de tu máquina.</p>
          </div>
          <div className="p-3 border border-gray-800 rounded-xl">
            <h3 className="font-medium text-gray-300 mb-1">Pagos</h3>
            <p>Stripe procesa los pagos. OVAV nunca ve ni almacena los datos de tu tarjeta.</p>
          </div>
        </div>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">4. Seguridad de datos</h2>
        <ul className="space-y-2 text-gray-400 text-sm ml-4">
          <li className="list-disc">Conexiones encriptadas con TLS 1.3</li>
          <li className="list-disc">Secretos en reposo con AES-256-GCM</li>
          <li className="list-disc">Machine fingerprints hasheados con SHA-256 (no reversibles)</li>
          <li className="list-disc">Base de datos aislada por tenant lógico</li>
          <li className="list-disc">Acceso administrativo mínimo, auditado, con 2FA</li>
        </ul>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">5. Tus derechos</h2>
        <ul className="space-y-2 text-gray-400 text-sm ml-4">
          <li className="list-disc">Acceder a todos tus datos — escribinos a privacy@ovav.dev</li>
          <li className="list-disc">Eliminar tu cuenta y todos tus datos — irreversible en 30 días</li>
          <li className="list-disc">Exportar tus datos en formato portable (JSON)</li>
          <li className="list-disc">Rectificar información incorrecta</li>
          <li className="list-disc">Oponerte al procesamiento — implica cierre de cuenta</li>
        </ul>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">6. Contacto</h2>
        <p className="text-gray-400 text-sm">
          Para cualquier duda sobre privacidad:{" "}
          <a href="mailto:privacy@ovav.dev" className="text-emerald-400 hover:text-emerald-300">
            privacy@ovav.dev
          </a>
        </p>
      </section>
    </main>
  );
}
