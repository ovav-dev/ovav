# OVAV Security Audit & Commit Identity System — Implementation Plan
## thavren · Platform Engineering · 2026-08-09 · v1.0

---

## FASE 1: AUDITORÍA DE SEGURIDAD (Security Cleanup)

### 1.1 Estado Actual — SSH Keys en GitHub

| Key ID | Tipo | Antigüedad | Etiqueta | Riesgo | Acción |
|--------|------|-----------|----------|--------|--------|
| 152319648 | RSA-4096 | Pre-2025 | (sin nombre) | 🔴 CRÍTICO — expuesta ~1 año | ELIMINAR |
| 152683836 | RSA-4096 | Pre-2025 | (sin nombre) | 🔴 CRÍTICO — expuesta ~1 año | ELIMINAR |
| 153129574 | RSA-4096 | Pre-2025 | (sin nombre) | 🔴 CRÍTICO — expuesta ~1 año | ELIMINAR |
| 159316185 | RSA-4096 | 2024-2026 | "OVAV WSL2", "OVAV RAIZ", "OVAV Signing Key 2026" | 🟡 MEDIO — múltiples nombres = posible uso compartido | REVISAR después de迁移 |

### 1.2 Estado Actual — Keys SSH Locales

| Archivo | Creado | Tipo | Uso | Riesgo | Acción |
|---------|--------|------|-----|--------|--------|
| `id_ed25519_ovav_github` | May 2024 | Ed25519 | GitHub auth | 🟡 MEDIO — 14 meses expuesta | MOVER a backup/ |
| `id_rsa` | Sep 2025 | RSA-2048 | GitHub auth (posiblemente) | 🟡 MEDIO — 11 meses expuesta | MOVER a backup/ |
| `ovav_signing_ed25519` | Aug 2026 | Ed25519 | SSH signing actual | 🟢 BAJO — 5 días expuesta | MANTENER |
| `ovav_signing_ed25519.pub` | Aug 2026 | Ed25519 | SSH signing público | 🟢 BAJO | MANTENER |
| `ovav_thavren_ed25519` | May 2026 | Ed25519 | Agent identity | 🟡 MEDIO — posible uso compartido | REVISAR contexto |
| `ovav-thavren-agent.env` | Jul 2026 | ENV file | Agent config | 🟡 MEDIO — puede contener secrets | REVISAR contenido |

### 1.3 GH Token

| Atributo | Valor | Riesgo |
|----------|-------|--------|
| Tipo | `gho_*` (OAuth token) | 🟡 MEDIO — tiene scopes de escritura |
| Protocolo | HTTPS | ✅ Correcto |
| Scopes | `user`, `repo`, `workflows` | ⚠️ Amplios — revisar necesidades mínimas |
| Owner | Alexander-Salvador | ✅ Correcto |

### 1.4 Org ovav-dev

| Recurso | Estado | Notas |
|---------|--------|-------|
| Miembros | 1 (Alexander-Salvador) | Solo tú por ahora |
| Repos | 1 (`ovav`) | Solo el repo principal |
| Secrets scanning | Disabled | 🔴 ACTIVAR en settings |
| Dependabot security | Disabled | 🟡 EVALUAR |
| Branch protection | No configurado | 🔴 CONFIGURAR |
| Admin permissions | Solo tú | ✅ Correcto |

---

## FASE 2: LIMPIEZA DE CLAVES SSH — ACCIONES INMEDIATAS

### ⚠️ ACCIONES REQUIEREN TU AUTORIZACIÓN MANUAL

GitHub no permite eliminar SSH keys via API con tokens OAuth — debés hacerlo manualmente:

**Pasos para eliminar keys antiguas (GitHub UI):**
1. Ir a: https://github.com/settings/keys
2. Para cada key con ID 152319648, 152683836, 153129574 → click "Delete"
3. Dejar SOLO la key 159316185 ("OVAV Signing Key 2026") por ahora

**Alternativa**: Si preferís que todas las keys se eliminen y se generen nuevas limpias:
- Eliminá TODAS las 4 keys de https://github.com/settings/keys
- Generá una nueva key específica para signing

### Keys locales a respaldar y remover del ~/.ssh/ activo:

```bash
# Crear directorio de backup
mkdir -p ~/.ssh/backup/$(date +%Y%m%d)

# Mover keys antiguas a backup (NO eliminar, guardar por seguridad)
mv ~/.ssh/id_ed25519_ovav_github ~/.ssh/backup/$(date +%Y%m%d)/
mv ~/.ssh/id_ed25519_ovav_github.pub ~/.ssh/backup/$(date +%Y%m%d)/
mv ~/.ssh/id_rsa ~/.ssh/backup/$(date +%Y%m%d)/
mv ~/.ssh/id_rsa.pub ~/.ssh/backup/$(date +%Y%m%d)/

# Mover backup a ubicación segura
mv ~/.ssh/backup /media/external/  # o Dropbox, etc.
```

---

## FASE 3: ARQUITECTURA DEL SISTEMA DE IDENTIDAD OVAV (MEJORADO)

### 3.1 Cómo OVAV Entiende las Identidades

OVAV ya tiene un sistema de agentes con identidades ricas:

```
OVAV Agent Identity Stack:
─────────────────────────────────────────────────────────────
Layer 1: HUMAN IDENTITY (CEO — Alexander Salvador)
         └─ GitHub Account: Alexander-Salvador
            └─ Email verificado: alexander_mya@outlook.com (u otro real)
               └─ SSH Key vinculada

Layer 2: LEAD IDENTITY (10 leads)
         └─ id: thavren, uriel, elena, eidren, dante, kenji, renata, sofia, valeria, camila
            └─ name: Thavren, Uriel, Elena...
               └─ area: platform_engineering, devops_infrastructure...
                  └─ origin: 🇳🇴 Norway, 🇮🇱 Israel...
                     └─ color: #2563eb...
                        └─ funciones, limitaciones, squad members

Layer 3: AGENT IDENTITY (Agents vivos)
         └─ Sesión activa con contexto, memoria, worktree
            └─ Puede ser: thavren, eidren, u otro lead
               └─ identity-guard valida que se mantenga en área
```

### 3.2 Sistema de Firma SSH por Lead — Diseño Mejorado

Cada lead tiene:
1. **Email de área**: `{lead}@ovav.{dev|ai|work}` — email ficticio verificado en GitHub
2. **SSH Signing Key**: key única Ed25519 por lead
3. **allowed_signers entry**: mapeo email → public key en `~/.config/ovav/allowed_signers`
4. **Git local config**: `user.name`, `user.email`, `user.signingkey` por worktree

### 3.3 Integración con Vault OVAV

OVAV ya tiene vault con:
- `vault/secrets/` — secretos con AES-256-GCM
- `vault/secrets/providers/github.go` — integración GitHub
- `vault/secrets/rotator.go` — rotación de secrets
- `vault/secrets/audit.go` — auditoría

**Nuevo**: `vault/secrets/identities.go` — almacenar identidades SSH firmadas

### 3.4 Flujo de Commit con Identidad OVAV

```
┌─────────────────────────────────────────────────────────────┐
│ CEO dice: "Thavren, implementa feature X"                  │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│ 1. OWS crea worktree desde develop                          │
│    git worktree add ../worktrees/feature-x -b feature/x     │
│    cd ../worktrees/feature-x                                │
│    git config --local user.name "Thavren"                   │
│    git config --local user.email "thavren@ovav.dev"         │
│    git config --local user.signingkey ~/.ssh/ovav-lead-thavren│
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│ 2. Cambios realizados + git add + git commit                │
│    └─ Git usa signingkey para firmar con SSH key            │
│    └─ Commit message incluye metadata OVAV en footer       │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│ 3. OWD para finalizar (antes de push)                       │
│    └─ OWS valida: ¿identity confirmada?                    │
│    └─ ¿capability actual en sync con caps.yaml?            │
│    └─ ¿scope del cambio dentro del área?                   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│ 4. Push a origin/develop                                    │
│    └─ GitHub recibe commit con:                            │
│       - Email: thavren@ovav.dev                            │
│       - SSH key fingerprint                                  │
│       - Firma SSH                                            │
│    └─ GitHub busca: ¿thavren@ovav.dev verificado en       │
│       alguna cuenta GitHub CON esa SSH key?                 │
│    └─ ¿SSH key subida a GitHub?                             │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│ RESULTADO:                                                 │
│ ✅ "Verified" + Badge de OVAV Lead en GitHub               │
│ ✅ thavren [platform_engineering] · Norway 🇳🇴            │
│ ✅ Commits tienen trace de área, lead, origin               │
└─────────────────────────────────────────────────────────────┘
```

### 3.5 Commit Message Format — OVAV Enhanced

```bash
# Formato completo
<type>(<scope>): <subject>

{OVAV_IDENTITY}
lead: {lead_name} [{area}]
origin: {origin} · capability: v{capa}
commit_id: {short_sha}
timestamp: {ISO8601}

{body}

Co-Authored-By: {squad_member_name} <{email}>
Signed-off-by: {lead_name} <{lead_email}>
OVAV-CIS/v1 · {git_branch} · {worktree_path}
```

**Ejemplo real:**
```
feat(platform): add SSH signing per lead identity

- Configured SSH signing for all 10 OVAV leads
- Added allowed_signers entry for each lead email
- Integrated with OWS worktree creation

OVAV_IDENTITY
lead: thavren [platform_engineering]
origin: 🇳🇴 Norway · capability: v11
commit_id: a3f8c2d
timestamp: 2026-08-09T14:30:00-05:00

Co-Authored-By: Marco <marco@ovav.dev>
Signed-off-by: Thavren <thavren@ovav.dev>
OVAV-CIS/v1 · develop · /home/braka/Systems/OVAV/.ovav/worktrees/feature-cis
```

---

## FASE 4: IMPLEMENTACIÓN DEL SISTEMA

### 4.1 Infraestructura de Keys (Automatizable)

```bash
# Directorio de trabajo
OVAV_KEYS_DIR=~/.config/ovav/keys
mkdir -p $OVAV_KEYS_DIR

# Leads que necesitan keys SSH
LEADS="thavren uriel elena eidren dante kenji renata sofia valeria camila"

for lead in $LEADS; do
  KEY_PATH="$OVAV_KEYS_DIR/ovav-lead-${lead}_ed25519"

  # Solo generar si no existe
  if [ ! -f "$KEY_PATH" ]; then
    ssh-keygen -t ed25519 \
      -C "${lead}@ovav.dev (OVAV Lead - ${lead})" \
      -f "$KEY_PATH" \
      -N "" \
      -q
    chmod 600 "$KEY_PATH"
    chmod 644 "${KEY_PATH}.pub"
    echo "Generated: $KEY_PATH"
  fi
done

# Generar allowed_signers
> $OVAV_KEYS_DIR/allowed_signers
for lead in $LEADS; do
  KEY_PATH="$OVAV_KEYS_DIR/ovav-lead-${lead}_ed25519.pub"
  if [ -f "$KEY_PATH" ]; then
    echo "${lead}@ovav.dev $(cat $KEY_PATH)" >> $OVAV_KEYS_DIR/allowed_signers
  fi
done

echo "=== allowed_signers generado ==="
cat $OVAV_KEYS_DIR/allowed_signers
```

### 4.2 Git Global Config (Base — Human Identity)

```bash
# Configurar como USER identidad (Alexander Salvador)
git config --global user.name "Alexander Salvador"
git config --global user.email "TU_EMAIL_REAL@dominio.com"  # Email verificado en GitHub
git config --global user.signingkey ~/.ssh/ovav_signing_ed25519
git config --global gpg.format ssh
git config --global commit.gpgsign true

# Configurar allowed_signers global
git config --global gpg.ssh.allowedsignersfile ~/.config/ovav/keys/allowed_signers
```

### 4.3 Git Local Config (Repo — Lead Identity Override)

```bash
# En el repo OVAV, sobreescribir con identidad de lead activo
cd /home/braka/Systems/OVAV

# Configurar identidad de Thavren para este repo
git config --local user.name "Thavren"
git config --local user.email "thavren@ovav.dev"
git config --local user.signingkey ~/.config/ovav/keys/ovav-lead-thavren_ed25519
git config --local gpg.format ssh
git config --local commit.gpgsign true
git config --local gpg.ssh.allowedsignersfile ~/.config/ovav/keys/allowed_signers
```

### 4.4 Commit Template

```bash
git config --global commit.template ~/.config/ovav/commit_template
```

Template (`~/.config/ovav/commit_template`):
```
{subject}

# OVAV Commit Identity
# Lines starting with OVAV_IDENTITY are parsed by OWS
# OVAV_IDENTITY
# OVAV_LEAD: {lead_name}
# OVAV_AREA: {area}
# OVOV_ORIGIN: {origin}
# OVAV_CAPA: {capa}
# OVAV_TIMESTAMP: {timestamp}

{body}

Signed-off-by: {lead_name} <{lead_email}>
OVAV-CIS/v1 · {branch}
```

### 4.5 Script de Cambio de Lead Identity

Para cambiar la identidad de commit según el lead activo:

```bash
#!/bin/bash
# ovav-switch-identity — Switch commit identity per lead
# Usage: ovav-switch-identity thavren

LEAD="${1:-thavren}"
KEYS_DIR="$HOME/.config/ovav/keys"
EMAIL="${LEAD}@ovav.dev"
LEAD_NAMES_FILE="$HOME/.config/ovav/lead_names.sh"

# Load lead names
source "$LEAD_NAMES_FILE"

NAME="${LEAD_NAMES[$LEAD]:-$LEAD}"
KEY="$KEYS_DIR/ovav-lead-${LEAD}_ed25519"

if [ ! -f "$KEY" ]; then
  echo "ERROR: Key for lead '$LEAD' not found at $KEY"
  echo "Run 'ovav-identity-setup' first"
  exit 1
fi

git config --local user.name "$NAME"
git config --local user.email "$EMAIL"
git config --local user.signingkey "$KEY"

echo "Switched to lead identity: $NAME <$EMAIL>"
echo "Signing key: $KEY"
```

---

## FASE 5: CONFIGURACIÓN GH TOKEN — MÍSIMOS PERMISOS

### Scopes actuales del token GH

Token actual: `gho_...` con scopes `user:email, repo, workflow`

**Scopes mínimos recomendados:**

| Scope | Justificación |
|-------|---------------|
| `repo` | Necesario para push/pull al repo privado |
| `read:org` | Leer membresía org ovav-dev |
| `user` | Verificar email verificado |
| `admin:public_key` | Eliminar/crear SSH keys via API |

### Acciones para mejorar token:

1. Ir a https://github.com/settings/tokens
2. Regenerar token con scopes mínimos
3. Guardar en `~/.config/ovav/credentials/gh_token.enc` (encriptado con vault)

---

## FASE 6: CONFIGURACIÓN DE ORGANIZACIÓN OVAV-DEV

### 6.1 Settings críticos

| Setting | Valor recomendado |
|---------|------------------|
| Default branch | `main` |
| Allow merge commits | ✅ Habilitado |
| Allow squash merging | ✅ Habilitado |
| Require PR reviews | ✅ Habilitado (1 reviewer para main) |
| Dismiss stale reviews | ✅ Habilitado |
| Require status checks | ✅ Habilitado antes de merge |
| Branch protection `main` | ✅ Enabled + restricciones |
| Branch protection `develop` | ✅ Enabled (no force push) |
| Secret scanning | 🔴 ACTIVAR |
| Secret scanning push protection | 🔴 ACTIVAR |
| Dependabot alerts | 🟡 Evaluar |
|强制pull request reviews before merging| ✅ Enabled |

### 6.2 Branch Protection via CLI

```bash
# Proteger main
gh api repos/ovav-dev/ovav/branches/main/protection \
  -X PUT \
  -f required_status_checks='{"strict":true,"contexts":[]}' \
  -f enforce_admins=true \
  -f required_pull_request_reviews='{"required_approving_review_count":1}' \
  -f restrictions=null

# Proteger develop
gh api repos/ovav-dev/ovav/branches/develop/protection \
  -X PUT \
  -f enforce_admins=true \
  -f required_pull_request_reviews=null \
  -f restrictions=null
```

---

## FASE 7: OWS INTEGRATION — AUTO-IDENTITY EN WORKTREES

### 7.1 Comportamiento Deseado

Cuando `owc` crea una worktree:

1. Detectar lead activo de la sesión
2. Leer identidad del lead desde `go-runtime/internal/agents/leads/`
3. Configurar `git config --local` con identity del lead
4. Registrar worktree en ledger de identidad
5. Al hacer `owd`, validar que la identity sea consistente con el área del cambio

### 7.2 Registro de Worktree con Identidad

OWS debería registrar en `.ovav/registry/worktrees.yaml`:
```yaml
worktrees:
  feature-cis:
    path: /home/braka/Systems/OVAV/.ovav/worktrees/feature-cis
    branch: feature/cis
    created_by: thavren
    lead: thavren
    area: platform_engineering
    email: thavren@ovav.dev
    created_at: 2026-08-09T14:30:00-05:00
    last_commit:
      sha: a3f8c2d
      signed: true
      verified: true
```

---

## RESUMEN — ACCIONES REQUERIDAS

### 🔴 ACCIONES MANUALES (Requiere tu intervención)

| # | Acción | Link/Ubicación |
|---|--------|----------------|
| M1 | Eliminar 3 SSH keys antiguas de GitHub (152319648, 152683836, 153129574) | https://github.com/settings/keys |
| M2 | **Pendiente:** ¿Mantener o eliminar key 159316185? (multiple names = risk) | https://github.com/settings/keys |
| M3 | Agregar email `thavren@ovav.dev` a tu cuenta GitHub y verificarlo | https://github.com/settings/emails |
| M4 | Regenerar GH token con scopes mínimos (después de cleanup) | https://github.com/settings/tokens |
| M5 | Activar Secret Scanning en org ovav-dev | https://github.com/organizations/ovav-dev/settings/security_analysis |
| M6 | Configurar branch protection rules | https://github.com/organizations/ovav-dev/rules |

### 🟡 ACCIONES AUTORIZABLES (Puedo ejecutar si confirmás)

| # | Acción | Comando |
|---|--------|---------|
| A1 | Mover keys SSH antiguas a backup | `mv ~/.ssh/id_* ~/.ssh/backup/$(date +%Y%m%d)/` |
| A2 | Generar SSH keys para 10 leads | Script en sección 4.1 |
| A3 | Crear `allowed_signers` con 10 leads | Script en sección 4.1 |
| A4 | Configurar git global con human identity | Sección 4.2 |
| A5 | Configurar git local con lead identity | Sección 4.3 |
| A6 | Crear commit template | Sección 4.4 |
| A7 | Subir public keys de leads a GitHub | Script + `gh ssh-key add` |
| A8 | Proteger branches main y develop | Sección 6.2 |

---

## PRIORIDADES DE IMPLEMENTACIÓN

```
Semana 1 (Hoy):
  1. [M1] Limpiar keys SSH antiguas de GitHub ← CRÍTICO
  2. [M2] Decidir sobre key 159316185
  3. [A1] Mover keys locales antiguas a backup
  4. [M3] Verificar email en GitHub

Semana 2:
  5. [M5] Activar secret scanning en org
  6. [M6] Configurar branch protection
  7. [A2-A3] Generar keys para leads + allowed_signers
  8. [A4-A5] Configurar git identity

Semana 3:
  9. [A6] Crear commit template
  10. [A7] Subir public keys a GitHub
  11. [M4] Regenerar token con scopes mínimos
  12. [A8] Proteger branches

Semana 4:
  13. OWS integration — auto-identity en worktrees
  14. vault/secrets/identities.go —存储SSH keys firmadas
  15. Validación end-to-end del sistema
```

---

*Document: docs/OVAV_SECURITY_AUDIT_AND_CIS.md*
*Authority: thavren (Platform Engineering)*
*Security classification: INTERNAL — SENSITIVE*
