# Adversarial Intelligence — Mandatory Questions

> Loaded by `ovav-brainstorm` skill when security testing/penetration/red team request detected.
> Ask 3-5 questions ONE AT A TIME. Wait for CEO response before next.

## Adversarial Questions

### AQ1: Threat Model
"¿Qué tipo de atacante simulamos?"
- External attacker (OWASP Top 10)
- Malicious insider (privilege escalation)
- Supply chain attack (compromised dependency)
- API abuse (rate limiting bypass, credential stuffing)
- Social engineering (phishing, pretexting)

Why: Cada threat model requiere técnicas diferentes. OWASP Top 10 = web app testing.

### AQ2: Testing Scope
"¿Qué está en scope?"
- Web app (frontend + API)
- Backend services (infrastructure)
- Mobile app (iOS/Android)
- Full stack (todo)
- Specific component ( Auth, payments, etc.)

Why: Definir scope evita tiempo perdido en áreas out-of-bounds y clarify qué NO se testa.

### AQ3: Testing Type
"¿Qué tipo de testing?"
- Black box (sin conocimiento interno)
- Gray box ( credenciales limitadas, algo de conocimiento)
- White box (código fuente + architecture)
- Bug bounty (bug bounty público/privado)

Why: White box = máximo coverage. Black box = realista. Bug bounty = escalabilidad.

### AQ4: Remediation SLA
"¿Qué pasa cuando se encuentra algo?"
- Immediate fix (critical = emergency patch)
- Standard fix (within sprint)
- Triage + risk assessment first (clasificar antes de fix)
- Document only (no fix required, solo awareness)

Why: Sin SLA claro, vulnerabilities pueden quedar open indefinidamente.

## Deliverable
After all questions answered → Kenji produces threat model and penetration testing plan.
