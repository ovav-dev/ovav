# OVAV Commit Identity System — Final Implementation
## thavren · Platform Engineering · 2026-08-09

---

## Modelo AI+Human para Commits Verificados

### El Problema Original
OVAV Agents son cerebros AI+HUMAN. Los modelos tradicionales de commit signing no contemplan esta dualidad.

### Investigación: Cómo Funciona GitHub Verified

| Factor | Requisito |
|--------|-----------|
| **Verified** | Firma válida + Key en GitHub + Email del **committer** verificado |
| **Author** | No afecta la verificación — solo identidad visible |
| **Firma SSH/GPG** | Solo verifica que el committer tiene la private key |

**Descubrimiento clave**: El badge "Verified" de GitHub se basa en el **committer** (quien hace `git commit`), no en el author (quien escribió el código).

### Modelo OVAV Adoptado

```
┌─────────────────────────────────────────────────────────┐
│ Author:  AI Agent (ej: "Thavren [platform_engineering]"│
│          thavren@ovav.agent)                           │
│                                                         │
│ Committer: HUMAN (Alexander Salvador)                    │
│            alexander_mya@outlook.com (verificado GitHub) │
│                                                         │
│ Firma:   GPG/SSH del humano — prueba criptográfica    │
│                                                         │
│ Resultado: ✅ Verified + Atribución AI visible        │
└─────────────────────────────────────────────────────────┘
```

GitHub muestra: ✅ "Verified" + "Alexander Salvador"
El author field muestra: "Thavren [platform_engineering]"

---

## Configuración Implementada

### Git Global (Humano - Committer Verificado)

```bash
git config --global user.name "Alexander Salvador"
git config --global user.email "alexander_mya@outlook.com"  # Email VERIFICADO en GitHub
git config --global user.signingkey 3DAC13769287AC80           # GPG Key en GitHub
git config --global gpg.format openpgp
git config --global commit.gpgsign true
```

### Para Atribuir AI Agent como Author

```bash
# Cuando el AI agent trabaja, usar environment vars:
GIT_AUTHOR_NAME="Thavren [platform_engineering]" \
GIT_AUTHOR_EMAIL="thavren@ovav.agent" \
git commit -S -m "feat(platform): description"
```

El commit resultante tiene:
- **Author**: Thavren [platform_engineering] <thavren@ovav.agent>
- **Committer**: Alexander Salvador <alexander_mya@outlook.com>
- **Verified**: ✅ (gracias al committer verificado)
- **Firma**: GPG del humano (prueba de autorización)

---

## SSH vs GPG — Por Qué GPG Ganó

| Aspecto | SSH | GPG |
|---------|-----|-----|
| GitHub Verified | ❌ unknown_key | ✅ valid |
| Email verification | No | ✅ El email del key debe coincidir con committer verificado |
| Complejidad | Simple | Media |
| Key management | Por cuenta | Por email del key |

SSH signing fallaba porque la key SSH no tiene email asociado directamente. GPG keys sí tienen email, y GitHub verifica ese email contra los verificados de la cuenta.

---

## Keys en el Sistema

### SSH Keys (10 leads - Firmas de AI Agents)
```
/home/braka/.config/ovav/keys/
├── ovav-lead-thavren_ed25519      # Para commits de Thavren agent
├── ovav-lead-uriel_ed25519
├── ovav-lead-elena_ed25519
...
```

### GPG Key (Humano - Verificada en GitHub)
```
/home/braka/.gnupg/
├── sec   rsa4096/3DAC13769287AC80
│   └── alexander_mya@outlook.com
```

### SSH Keys en GitHub (Alexander-Salvador account)
11 keys subidas:
- CEO: OVAV CEO Human Operator Signing Key
- 10 leads: OVAV {Lead} Lead Signing Key 2026

---

## Commits en el Repo

### Antes (Unverified)
```
committer: Thavren <thavren@ovav.worktree>
reason: no_user
verified: false
```

### Después (Verified)
```
committer: Alexander Salvador <alexander_mya@outlook.com>
author: Thavren [platform_engineering] <thavren@ovav.agent>
reason: valid
verified: true ✅
```

---

## Flujo de Trabajo

### 1. Humano Trabaja Directamente
```bash
git commit -S -m "fix: quick fix"
# Committer: Alexander Salvador (Verified)
# Author: Alexander Salvador
# Badge: ✅ Verified
```

### 2. AI Agent Trabaja en Nombre del Humano
```bash
GIT_AUTHOR_NAME="Thavren [platform_engineering]" \
GIT_AUTHOR_EMAIL="thavren@ovav.agent" \
git commit -S -m "feat(platform): new feature"
# Committer: Alexander Salvador (Verified)
# Author: Thavren [platform_engineering]
# Badge: ✅ Verified
# Attribution: AI agent visible
```

### 3. OWD Finaliza Worktree
```bash
owd  # Valida, firma, push
# El push usa las credenciales del humano
# Todos los commits mantienen Verified
```

---

## Seguridad

| Medida | Estado |
|--------|--------|
| 4 SSH keys antiguas eliminadas (RSA) | ✅ |
| Keys en ~/.ssh/backup/ | ✅ Respaldadas |
| GPG key en GitHub verificada | ✅ |
| Commits firmados por defecto | ✅ |
| Secrets Scanning activo | ✅ |
| allowed_signers actualizado | ✅ |

---

## Commits Verificados en GitHub

Para verificar cualquier commit:
```bash
gh api repos/ovav-dev/ovav/commits/{sha} --jq '.commit.verification'
```

Respuesta esperada:
```json
{"verified": true, "reason": "valid"}
```

---

*Document: docs/COMMIT_IDENTITY_FINAL.md*
*Authority: thavren (Platform Engineering)*
*Model: AI+Human dual identity — Verified commits with AI attribution*
