# OVAV Commit Identity System (CIS)
## thavren · Platform Engineering · 2026-08-09 · v1.0

---

## 1. Diagnóstico Actual

### Estado Encontrado

| Componente | Estado | Detalle |
|------------|--------|---------|
| SSH Signing | ✅ Configurado | `gpg.format=ssh`, `commit.gpgsign=true` |
| Signing Key | ✅ Existe | `~/.ssh/ovav_signing_ed25519` (Ed25519) |
| allowed_signers | ✅ Existe | `~/.config/ovav/allowed_signers` |
| identities.json | ✅ Existe | `~/.config/ovav/identities.json` |
| Repo | ✅ Migrado | `github.com/ovav-dev/ovav` |
| GitHub Verified | ❌ | Email `thavren@ovav.worktree` no asociado a cuenta GitHub |

### El Problema

GitHub muestra **"No user is associated with the committer email"** porque:
1. El email del commit (`thavren@ovav.worktree`) no está verificado en ninguna cuenta GitHub
2. La SSH key está subida a GitHub (Alexander-Salvador), pero el email no coincide
3. GitHub verifica que el email del commit corresponda a una cuenta con la key SSH subida

---

## 2. Arquitectura del Sistema de Identidades OVAV

### Modelo de Identidades

OVAV tiene **3 niveles de identidad commit**:

```
Nivel 1 — HUMANOS (CEO y colaboradores reales)
├── CEO: Alexander Salvador (Alexander-Salvador@github)
│   └── Email verificado en github.com
├── (futuro) otros humanos con cuentas GitHub
│
Nivel 2 — LEADS OVAV (10 agentes principales)
├── Cada lead tiene: id, nombre, área, origen, color
├── Cada lead puede tener su propia SSH signing key
└── Sus commits reflejan su identidad de área
    │
Nivel 3 — SQUAD MEMBERS (12 miembros por lead)
├── Comparten la identidad del lead padre
└── Sus commits llevan metadata del lead + area

NOTA: En el modelo actual, todos los commits de una sesión
de agente usan la identidad del lead activo (thavren, eidren, etc.)
```

### Flujo de Verificación GitHub

```
Commit firmado SSH
       │
       ▼
GitHub recibe: email + SSH public key fingerprint + firma
       │
       ▼
GitHub busca: ¿hay alguna cuenta con este email VERIFICADO
              Y con esta SSH key subida?
       │
   ┌───┴───┐
   │       │
  SÍ       NO
   │       │
   ▼       ▼
"Verified"  "No user associated"
```

---

## 3. Plan de Implementación

### Fase 1: Configuración de Email Verificado (CRÍTICO)

**Problema raíz**: `thavren@ovav.worktree` no es un email real.

**Solución**: Usar el email de la cuenta GitHub como identidad de commit.

| Opción | Pros | Contras |
|--------|------|---------|
| A) Agregar email `thavren@ovav.worktree` a Alexander-Salvador en GitHub | Rápido | Email no real, puede verse extraño |
| B) Cambiar commits para usar `alexander_mya@outlook.com` | Email real, verified instantáneo | Pierde identidad thavren |
| C) Crear email real `thavren@ovav.dev` y verificar en GitHub | Profesional, identidad real | Requiere setup de email |

**Recomendación**: Opción C — crear identidad de email real.

### Fase 2: Sistema de Firmas por Lead

#### Estructura de Identidades SSH

```
~/.ssh/
├── ovav_signing_ed25519          # Key principal (Thavren/Human)
├── ovav_signing_ed25519.pub
├── ovav-lead-thavren_ed25519    # Key por lead
├── ovav-lead-eidren_ed25519
├── ovav-lead-dante_ed25519
... (10 leads)
```

#### allowed_signers actualizado

```
thavren@ovav.dev ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA...
eidren@ovav.dev ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA...
dante@ovav.dev ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA...
... (10 leads)
```

### Fase 3: Git Config por Lead

```bash
# Plantilla global para todos los leads
git config --global user.name "Thavren [platform_engineering]"
git config --global user.email "thavren@ovav.dev"
git config --global user.signingkey ~/.ssh/ovav-lead-thavren_ed25519
git config --global gpg.format ssh
git config --global commit.gpgsign true

# Override local por repo para el lead activo
git config --local user.name "Thavren"
git config --local user.email "thavren@ovav.worktree"
```

### Fase 4: Commit Message Template con Metadata

```bash
git config --global commit.template ~/.config/ovav/commit_template
```

Template:
```
{subject}

{OVAV_IDENTITY}
area: {area}
lead: {lead_name}
origin: {origin}
capability: {current_capa}
timestamp: {timestamp}

{body}

Signed-off-by: {lead_name} <{email}>
```

---

## 4. Leads OVAV — Identidades de Commit

| Lead | ID | Área | Email Proposed | Origin | Color |
|------|-----|------|----------------|--------|-------|
| Thavren | thavren | platform_engineering | thavren@ovav.dev | 🇳🇴 Norway | #2563eb |
| Uriel | uriel | devops_infrastructure | uriel@ovav.dev | 🇮🇱 Israel | #ca8a04 |
| Elena | elena | ux_design | elena@ovav.dev | 🇪🇸 Spain | — |
| Eidren | eidren | research_intelligence | eidren@ovav.dev | 🇮🇸 Iceland | — |
| Dante | dante | digital_product | dante@ovav.dev | 🇮🇹 Italy | — |
| Kenji | kenji | adversarial_intelligence | kenji@ovav.dev | 🇯🇵 Japan | — |
| Renata | renata | health_performance | renata@ovav.dev | 🇵🇱 Poland | — |
| Sofía | sofia | commercial_growth | sofia@ovav.dev | 🇬🇷 Greece | — |
| Valeria | valeria | education_career | valeria@ovav.dev | 🇨🇴 Colombia | #0891b2 |
| Camila | camila | legal_compliance | camila@ovav.dev | 🇨🇴 Colombia | — |

---

## 5. Pasos de Implementación Inmediata

### Paso 1: Verificar Email en GitHub (Requiere Acción Manual)

1. Ir a https://github.com/settings/emails
2. Agregar `thavren@ovav.dev` (o email elegido)
3. Verificar el email (recibir correo de GitHub)
4. Subir SSH signing key a GitHub si no está:
   - https://github.com/settings/keys
   - New SSH key
   - Pegar contenido de `~/.ssh/ovav_signing_ed25519.pub`

### Paso 2: Generar SSH Keys por Lead (Automatizable)

```bash
# Para cada lead, generar key única
for lead in thavren uriel elena eidren dante kenji sofia renata valeria camila; do
  ssh-keygen -t ed25519 -C "${lead}@ovav.dev" -f ~/.ssh/ovav-lead-${lead}_ed25519 -N ""
done

# Agregar todas las public keys a allowed_signers
cat ~/.ssh/ovav-lead-*_ed25519.pub >> ~/.config/ovav/allowed_signers
```

### Paso 3: Subir Keys a GitHub (Automatizable)

```bash
# Para cada lead, subir key a GitHub (requiere token con permisos)
gh ssh-key add ~/.ssh/ovav-lead-${lead}_ed25519.pub -t "OVAV ${lead} signing key"
```

### Paso 4: Configurar Git Global

```bash
# Configuración global
git config --global user.email "thavren@ovav.dev"  # Email verificado en GitHub
git config --global user.name "Thavren"
git config --global user.signingkey ~/.ssh/ovav_signing_ed25519
git config --global gpg.format ssh
git config --global commit.gpgsign true

# Configuración local del repo (identidad de trabajo)
git config --local user.email "thavren@ovav.worktree"  # Identity de área
git config --local user.name "Thavren"
```

### Paso 5: Re-firmar commits existentes (Opcional)

```bash
# Rebase interactivo para re-firmar todos los commits
git rebase -r --exec 'git commit --amend --no-edit -S' HEAD
```

---

## 6. Validación del Sistema

```bash
# Test: hacer commit y verificar firma
git commit -S -m "test: verify commit signature" --allow-empty
git log --show-signature -1

# Verificar en GitHub
gh api repos/ovav-dev/ovav/commits/$(git rev-parse HEAD) --jq '.commit.verification'
```

Resultado esperado:
```json
{
  "verified": true,
  "reason": "valid",
  "signature": "...",
  "signer": {
    "login": "Alexander-Salvador"
  }
}
```

---

## 7. Integración OWS (Worktree System)

Cuando `owc` crea una worktree, debe:
1. Leer el lead activo del contexto de sesión
2. Configurar `user.name` y `user.email` local de la worktree según el lead
3. Verificar que la SSH key del lead esté disponible
4. Registrar la worktree en el ledger de identidad

```bash
# En owc create — post-create hook
git worktree add ../feature-xxx -b feature/xxx
cd ../feature-xxx
git config --local user.name "${LEAD_NAME}"
git config --local user.email "${LEAD_EMAIL}"
```

---

## 8. Commit Message Format (Conventional Commits + OVAV)

```
<type>(<scope>): <subject>

[optional body]

[optional footer]

# Ejemplo:
feat(platform): add SSH signing per lead identity

- Configured SSH signing for all 10 OVAV leads
- Added allowed_signers entry for each lead email
- Integrated with OWS worktree creation

OVAV-CIS/lead: thavren
OVAV-CIS/area: platform_engineering
OVAV-CIS/capa: 11
```

---

## 9. Pendientes de Decisión

1. **Email real**: ¿Crear emails `@ovav.dev` o usar existente?
2. **SSH Keys**: ¿Una sola key compartida o una por lead?
3. **GitHub Org vs Personal**: ¿Subir keys a `ovav-dev` org o cuentas personales de leads?
4. **Re-firmar historial**: ¿Vale la pena re-firmar todo el historial?

---

*Document: docs/COMMIT_IDENTITY_SYSTEM.md*
*Authority: thavren (Platform Engineering)*
*Next review: After email verification + key upload*
