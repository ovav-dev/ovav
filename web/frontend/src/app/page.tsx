"use client";

import { motion } from "framer-motion";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Section } from "@/components/ui/section";
import { GradientText } from "@/components/ui/gradient-text";

const PILLARS = [
  {
    icon: (
      <svg className="w-8 h-8" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5" strokeLinecap="round" strokeLinejoin="round" />
      </svg>
    ),
    title: "Boundary Law",
    desc: "Cada agente opera en su carril. Ramas protegidas, secretos blindados, privilegios controlados. Si un agente intenta salirse, OVAV lo bloquea sin preguntar.",
    gradient: "from-emerald-500/20 to-emerald-600/5",
    border: "border-emerald-500/20",
  },
  {
    icon: (
      <svg className="w-8 h-8" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" strokeLinecap="round" strokeLinejoin="round" />
      </svg>
    ),
    title: "Evidence Pipeline",
    desc: "Cada cambio pasa por 5 gates de validación. 72 validators determinísticos. Si algo falla, se bloquea. Trazabilidad completa de cada decisión.",
    gradient: "from-blue-500/20 to-blue-600/5",
    border: "border-blue-500/20",
  },
  {
    icon: (
      <svg className="w-8 h-8" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" strokeLinecap="round" strokeLinejoin="round" />
      </svg>
    ),
    title: "Session Capsule",
    desc: "Cada sesión de agente está aislada. Presupuesto de tokens, firewall de contexto, memoria sellada. Sin fugas. Sin contaminación entre sesiones.",
    gradient: "from-purple-500/20 to-purple-600/5",
    border: "border-purple-500/20",
  },
];

const PLANS = [
  {
    name: "Core",
    price: "0",
    period: "",
    features: [
      "Boundary Law",
      "Output Guard",
      "Secrets Hygiene",
      "Drift Detection",
      "Session Capsule",
      "6 validators esenciales",
    ],
    cta: "Comenzar gratis",
    href: "/checkout?tier=core",
    highlight: false,
  },
  {
    name: "Pro",
    price: "19",
    period: "/mes",
    features: [
      "Todo lo de Core",
      "12 validators avanzados",
      "Context Firewall",
      "ConnectorBus",
      "Memory Governor",
      "CLI Visual + SBOM",
    ],
    cta: "Prueba gratuita",
    href: "/checkout?tier=pro",
    highlight: true,
    badge: "Más popular",
  },
  {
    name: "Enterprise",
    price: "49",
    period: "/usuario/mes",
    features: [
      "Todo lo de Pro",
      "SSO (Okta, Azure AD)",
      "Audit Log completo",
      "Team Management",
      "Custom Rules",
      "Priority Support + SLA",
    ],
    cta: "Hablar con ventas",
    href: "/checkout?tier=enterprise",
    highlight: false,
  },
];

const SOCIAL_PROOF = [
  { value: "72+", label: "Validators activos" },
  { value: "14", label: "Controles de seguridad" },
  { value: "35", label: "Páginas documentadas" },
  { value: "<80ms", label: "Latencia de gate" },
];

export default function LandingPage() {
  return (
    <main className="min-h-screen bg-[#030712] text-gray-100 overflow-x-hidden">
      {/* ─── Hero ─── */}
      <section className="relative max-w-5xl mx-auto px-6 pt-28 pb-24 md:pt-40 md:pb-32 text-center">
        {/* Background glow */}
        <div className="absolute inset-0 -top-40 flex justify-center overflow-hidden pointer-events-none">
          <div className="w-[600px] h-[400px] bg-emerald-500/10 rounded-full blur-[120px]" />
        </div>

        <motion.div
          initial={{ opacity: 0, y: 30 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.8, ease: [0.25, 0.1, 0.25, 1] }}
        >
          <Badge variant="success" className="mb-6">v1.1.0 — Disponible ahora</Badge>
        </motion.div>

        <motion.h1
          className="text-4xl sm:text-5xl md:text-6xl lg:text-7xl font-bold tracking-tight leading-[1.1] mb-6"
          initial={{ opacity: 0, y: 30 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.8, delay: 0.1, ease: [0.25, 0.1, 0.25, 1] }}
        >
          Tus agentes de IA,
          <br />
          <GradientText>bajo tu control</GradientText>
        </motion.h1>

        <motion.p
          className="text-lg md:text-xl text-gray-400 max-w-2xl mx-auto mb-10 leading-relaxed"
          initial={{ opacity: 0, y: 30 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.8, delay: 0.2, ease: [0.25, 0.1, 0.25, 1] }}
        >
          OVAV es el governor de workstation AI que se asegura de que cada
          comando, cada push, y cada secreto pase por tus reglas. Integra
          Copilot, Cursor, o cualquier tool — OVAV gobierna.
        </motion.p>

        <motion.div
          className="flex flex-col sm:flex-row gap-4 justify-center"
          initial={{ opacity: 0, y: 30 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.8, delay: 0.3, ease: [0.25, 0.1, 0.25, 1] }}
        >
          <Button variant="primary" size="lg" href="/checkout?tier=pro">
            Comenzar prueba gratuita
          </Button>
          <Button variant="outline" size="lg" href="/docs">
            Ver documentación
          </Button>
        </motion.div>

        <motion.p
          className="text-sm text-gray-600 mt-4"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.8, delay: 0.5 }}
        >
          Pro 14 días gratis. Sin tarjeta. Cancelá cuando quieras.
        </motion.p>
      </section>

      {/* ─── Value Pillars ─── */}
      <Section className="max-w-5xl mx-auto px-6 py-20 md:py-28" delay={0.1}>
        <div className="text-center mb-16">
          <h2 className="text-3xl md:text-4xl font-bold mb-4">
            Gobernanza que <GradientText>no se negocia</GradientText>
          </h2>
          <p className="text-gray-500 max-w-xl mx-auto">
            Tres principios. Cero excepciones. Construido para desarrolladores que
            exigen control total sobre sus herramientas de IA.
          </p>
        </div>

        <div className="grid md:grid-cols-3 gap-6">
          {PILLARS.map((p, i) => (
            <motion.div
              key={p.title}
              initial={{ opacity: 0, y: 40 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true, margin: "-50px" }}
              transition={{ duration: 0.5, delay: i * 0.1 }}
            >
              <Card
                hover
                className={`relative overflow-hidden bg-gradient-to-b ${p.gradient}`}
              >
                <div className={`text-emerald-400 mb-4`}>{p.icon}</div>
                <h3 className="text-lg font-semibold mb-2">{p.title}</h3>
                <p className="text-gray-400 text-sm leading-relaxed">{p.desc}</p>
                <div className={`absolute top-0 right-0 w-24 h-24 rounded-bl-full opacity-10 bg-gradient-to-br ${p.gradient}`} />
              </Card>
            </motion.div>
          ))}
        </div>
      </Section>

      {/* ─── Social Proof ─── */}
      <Section className="max-w-5xl mx-auto px-6 py-20" delay={0.1}>
        <div className="border border-gray-800 rounded-2xl bg-gray-950/50 p-10 md:p-16">
          <div className="grid grid-cols-2 md:grid-cols-4 gap-8 text-center">
            {SOCIAL_PROOF.map((stat) => (
              <div key={stat.label}>
                <div className="text-3xl md:text-4xl font-bold text-emerald-400 mb-1 font-mono">
                  {stat.value}
                </div>
                <div className="text-sm text-gray-500">{stat.label}</div>
              </div>
            ))}
          </div>
        </div>
      </Section>

      {/* ─── Pricing ─── */}
      <Section className="max-w-5xl mx-auto px-6 py-20 md:py-28" id="pricing">
        <div className="text-center mb-16">
          <h2 className="text-3xl md:text-4xl font-bold mb-4">
            Planes <GradientText>simples</GradientText>
          </h2>
          <p className="text-gray-500 max-w-xl mx-auto">
            Empezá gratis. Upgradeá cuando necesités más control.
            Sin contratos. Sin sorpresas.
          </p>
        </div>

        <div className="grid md:grid-cols-3 gap-6 items-start">
          {PLANS.map((plan, i) => (
            <motion.div
              key={plan.name}
              initial={{ opacity: 0, y: 40 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ duration: 0.5, delay: i * 0.1 }}
            >
              <Card
                className={`relative ${
                  plan.highlight
                    ? "border-emerald-500/30 bg-gradient-to-b from-emerald-500/5 to-transparent shadow-lg shadow-emerald-500/5"
                    : ""
                }`}
              >
                {plan.badge && (
                  <Badge variant="success" className="absolute -top-3 left-1/2 -translate-x-1/2">
                    {plan.badge}
                  </Badge>
                )}
                <h3 className="text-xl font-semibold mb-1">{plan.name}</h3>
                <div className="flex items-baseline gap-1 mb-6">
                  <span className="text-4xl font-bold">${plan.price}</span>
                  {plan.period && (
                    <span className="text-gray-500 text-sm">{plan.period}</span>
                  )}
                </div>
                <ul className="space-y-3 mb-8">
                  {plan.features.map((f) => (
                    <li key={f} className="flex items-start gap-2 text-sm text-gray-400">
                      <svg className="w-4 h-4 text-emerald-400 mt-0.5 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                        <path d="M5 13l4 4L19 7" strokeLinecap="round" strokeLinejoin="round" />
                      </svg>
                      {f}
                    </li>
                  ))}
                </ul>
                <Button
                  variant={plan.highlight ? "primary" : "outline"}
                  href={plan.href}
                  className="w-full"
                >
                  {plan.cta}
                </Button>
              </Card>
            </motion.div>
          ))}
        </div>
      </Section>

      {/* ─── CTA Final ─── */}
      <Section className="max-w-3xl mx-auto px-6 py-20 md:py-28 text-center">
        <h2 className="text-3xl md:text-4xl font-bold mb-4">
          ¿Listo para <GradientText>tomar el control</GradientText>?
        </h2>
        <p className="text-gray-500 mb-8 max-w-lg mx-auto">
          Instalá OVAV en tu proyecto. Sin configuraciones complejas.
          Sin dependencias externas. Gobernanza real desde el primer comando.
        </p>
        <div className="flex flex-col sm:flex-row gap-4 justify-center">
          <Button variant="primary" size="lg" href="/checkout?tier=pro">
            Probar gratis 14 días
          </Button>
          <Button variant="secondary" size="lg" href="/docs/quickstart">
            Quickstart
          </Button>
        </div>
      </Section>

      {/* ─── Footer ─── */}
      <footer className="border-t border-gray-800/50">
        <div className="max-w-5xl mx-auto px-6 py-12">
          <div className="grid grid-cols-2 md:grid-cols-4 gap-8 mb-12">
            <div>
              <h4 className="text-sm font-semibold mb-3">Producto</h4>
              <div className="space-y-2 text-sm text-gray-500">
                <a href="/#pricing" className="block hover:text-gray-300 transition">Planes</a>
                <a href="/docs" className="block hover:text-gray-300 transition">Documentación</a>
                <a href="/docs/quickstart" className="block hover:text-gray-300 transition">Quickstart</a>
                <a href="/docs/security" className="block hover:text-gray-300 transition">Seguridad</a>
              </div>
            </div>
            <div>
              <h4 className="text-sm font-semibold mb-3">Empresa</h4>
              <div className="space-y-2 text-sm text-gray-500">
                <a href="#" className="block hover:text-gray-300 transition">Blog</a>
                <a href="#" className="block hover:text-gray-300 transition">GitHub</a>
                <a href="#" className="block hover:text-gray-300 transition">Discord</a>
                <a href="mailto:hola@ovav.dev" className="block hover:text-gray-300 transition">Contacto</a>
              </div>
            </div>
            <div>
              <h4 className="text-sm font-semibold mb-3">Legal</h4>
              <div className="space-y-2 text-sm text-gray-500">
                <a href="/privacy" className="block hover:text-gray-300 transition">Privacidad</a>
                <a href="/terms" className="block hover:text-gray-300 transition">Términos</a>
                <a href="mailto:security@ovav.dev" className="block hover:text-gray-300 transition">Seguridad</a>
              </div>
            </div>
            <div>
              <h4 className="text-sm font-semibold mb-3">OVAV</h4>
              <p className="text-sm text-gray-500 leading-relaxed">
                AI Workstation Governor.
                Construido en Buenos Aires, Argentina.
              </p>
            </div>
          </div>
          <div className="border-t border-gray-800/50 pt-8 flex flex-col sm:flex-row justify-between items-center gap-4">
            <div className="flex items-center gap-2 text-sm text-gray-600">
              <span className="font-bold text-gray-400">OVAV</span>
              <span>v1.1.0</span>
            </div>
            <p className="text-xs text-gray-700">
              &copy; {new Date().getFullYear()} OVAV Technologies. Todos los derechos reservados.
            </p>
          </div>
        </div>
      </footer>
    </main>
  );
}
