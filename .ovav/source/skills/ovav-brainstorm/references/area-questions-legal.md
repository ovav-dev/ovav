# Legal & Compliance — Mandatory Questions

> Loaded by `ovav-brainstorm` skill when legal/contract/GDPR/compliance request detected.
> Ask 3-5 questions ONE AT A TIME. Wait for CEO response before next.

## Legal Questions

### LQ1: Applicable Regulations
"¿Qué regulaciones aplican?"
- GDPR (EU users, data protection)
- LGPD (Brasil, similar a GDPR)
- CCPA (California consumers)
- HIPAA (US health data)
- PCI-DSS (payment card data)
- Ninguna específica (B2B only, sin consumers)

Why: GDPR = cookie banners + data portability + breach notification. PCI-DSS = si guardás credit card data (no recomendado).

### LQ2: Data Residency
"¿Dónde se guardan los datos de usuarios?"
- US only
- EU only (para GDPR compliance)
- User's choice (geo-routing)
- Multi-region (replicación cross-border)

Why: GDPR requiere datos de EU citizens en EU. Otros regulations pueden tener restricciones similares.

### LQ3: Privacy by Design
"¿Qué datos se colectan?"
- Mínimo (email + password solo)
- Moderate (email + profile + usage analytics)
- Full (location, device, behavioral tracking)

Why: Menos datos = menos liability. analytics anonimizado es menos risk que PII.

### LQ4: Documentos Legales
"¿Qué documentos se necesitan?"
- Terms of Service (ToS)
- Privacy Policy
- Cookie Policy (si hay cookies de tracking)
- DPA (Data Processing Agreement) para B2B
- Ninguno (proyecto hobby o internal)

Why: ToS + Privacy Policy son minimum para cualquier sitio público. Para B2B, el DPA es standard.

## Deliverable
After all questions answered → Camila drafts required legal documents.
