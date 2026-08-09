import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Términos de Servicio — OVAV",
  description: "Condiciones de uso de OVAV AI Workstation Governor.",
};

export default function TermsPage() {
  return (
    <main className="max-w-3xl mx-auto px-6 py-12">
      <a href="/" className="text-sm text-gray-400 hover:text-emerald-400 mb-6 inline-block transition">
        ← Inicio
      </a>
      <h1 className="text-3xl font-bold mb-3">Términos de Servicio</h1>
      <p className="text-gray-400 mb-10">Última actualización: Junio 2026</p>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">1. Aceptación</h2>
        <p className="text-gray-400 text-sm">
          Al usar OVAV, aceptás estos términos. Si no estás de acuerdo, no uses el producto.
          OVAV es un producto de OVAV Technologies, registrado en Buenos Aires, Argentina.
        </p>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">2. Licencia</h2>
        <p className="text-gray-400 text-sm mb-3">
          OVAV Core se distribuye bajo licencia Apache 2.0. OVAV Pro y Enterprise son licencias
          comerciales. Al adquirir una licencia Pro o Enterprise, recibís:
        </p>
        <ul className="space-y-1 text-gray-400 text-sm ml-4">
          <li className="list-disc">Derecho de uso en la cantidad de instancias contratadas</li>
          <li className="list-disc">Actualizaciones durante la vigencia de la licencia</li>
          <li className="list-disc">Soporte según el tier contratado</li>
        </ul>
        <p className="text-gray-400 text-sm mt-3">
          Está prohibido: redistribuir OVAV Pro/Enterprise, realizar ingeniería inversa del
          sistema de licencias, o usar una licencia en más instancias que las contratadas.
        </p>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">3. Planes y pagos</h2>
        <div className="space-y-3 text-gray-400 text-sm">
          <div className="p-3 border border-gray-800 rounded-xl">
            <h3 className="font-medium text-gray-300 mb-1">Core (Gratis)</h3>
            <p>6 features esenciales. Sin límite de tiempo. Sin tarjeta.</p>
          </div>
          <div className="p-3 border border-gray-800 rounded-xl">
            <h3 className="font-medium text-gray-300 mb-1">Pro ($19/mes)</h3>
            <p>14 días de prueba gratuita. Cancelación en cualquier momento. Sin penalidad.</p>
          </div>
          <div className="p-3 border border-gray-800 rounded-xl">
            <h3 className="font-medium text-gray-300 mb-1">Enterprise ($49/usuario/mes)</h3>
            <p>Contrato anual. SSO, auditoría, SLAs. Términos personalizados disponibles.</p>
          </div>
        </div>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">4. Disclaimer de salud</h2>
        <div className="p-4 border border-yellow-500/20 bg-yellow-500/5 rounded-xl">
          <p className="text-sm text-gray-400 font-semibold mb-2">
            ⚠️ OVAV no es un dispositivo médico ni sustituye el consejo de un profesional de la salud.
          </p>
          <p className="text-xs text-gray-500 leading-relaxed">
            Los módulos de salud y rendimiento de OVAV (nutrición, fitness, sueño, suplementación)
            proporcionan información educativa basada en evidencia científica. Esta información no
            constituye diagnóstico, tratamiento, ni prescripción médica. Consultá siempre a un
            profesional de la salud antes de realizar cambios en tu dieta, ejercicio, o suplementación.
            Si experimentás dolor en el pecho, dificultad para respirar, o cualquier síntoma preocupante
            durante el ejercicio, detenete inmediatamente y buscá atención médica.
          </p>
        </div>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">5. Limitación de responsabilidad</h2>
        <p className="text-gray-400 text-sm mb-3">
          OVAV se proporciona &ldquo;tal cual&rdquo;, sin garantías de ningún tipo. No nos hacemos responsables por:
        </p>
        <ul className="space-y-1 text-gray-400 text-sm ml-4">
          <li className="list-disc">Decisiones tomadas basadas en outputs de agentes de IA</li>
          <li className="list-disc">Pérdida de datos por no realizar backups (aunque OVAV los facilita)</li>
          <li className="list-disc">Daños derivados del uso de módulos de salud sin supervisión profesional</li>
          <li className="list-disc">Interrupciones del servicio por causas fuera de nuestro control</li>
        </ul>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">6. Cancelación</h2>
        <p className="text-gray-400 text-sm">
          Podés cancelar tu suscripción en cualquier momento desde el dashboard. La cancelación
          es efectiva al final del período de facturación actual. No hay penalidades ni cargos
          ocultos. Los datos de tu cuenta se conservan por 90 días por si querés reactivar.
        </p>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">7. Cambios en los términos</h2>
        <p className="text-gray-400 text-sm">
          Te notificaremos por email sobre cambios significativos con al menos 30 días de
          anticipación. El uso continuado después de los cambios implica aceptación.
        </p>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">8. Ley aplicable</h2>
        <p className="text-gray-400 text-sm">
          Estos términos se rigen por las leyes de la República Argentina. Cualquier disputa
          será resuelta en los tribunales de la Ciudad Autónoma de Buenos Aires.
        </p>
      </section>

      <section className="mb-10">
        <h2 className="text-xl font-semibold mb-3">9. Contacto</h2>
        <p className="text-gray-400 text-sm">
          Para cuestiones legales:{" "}
          <a href="mailto:legal@ovav.dev" className="text-emerald-400 hover:text-emerald-300">
            legal@ovav.dev
          </a>
        </p>
      </section>
    </main>
  );
}
