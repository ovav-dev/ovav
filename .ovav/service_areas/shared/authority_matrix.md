# OVAV — Matriz de Autoridad Delegada v2.0

# **Versión:** 2.0.0  
# **Fecha:** 2026-06-10  
# **CEO approval:** Alexander Salvador  

## Principio

Thavren es el platform lead — construye y mantiene las herramientas. Pero la coordinación de proyectos multi-área no debe pasar por Thavren. El lead coordinador (Dante, para proyectos de producto digital) necesita autoridad delegada para aprobar handoffs, establecer deadlines, y resolver bloqueos cross-área.


## 1. Niveles de autoridad por lead

| Lead | Repo-local mutación | Aprobar handoffs | Delegar a squads | Push | PR merge | External dirs | Global write |
|------|---------------------|-------------------|------------------|------|----------|----------------|--------------|
| **Thavren** | ✅ allow | ✅ allow | ✅ allow | ✅ gate | ⚠️ CEO | ✅ allow | ✅ allow |
| **Dante** | ✅ allow | ✅ allow [NUEVO] | ✅ allow | ✅ gate | ⚠️ CEO | ✅ allow [NUEVO] | ❌ deny |
| **Eidren** | ⚠️ read-only | ❌ deny | ✅ allow | ❌ deny | ❌ deny | ⚠️ research | ❌ deny |
| **Sofía** | ✅ allow | ✅ allow [NUEVO] | ✅ allow | ✅ gate | ⚠️ CEO | ✅ allow [NUEVO] | ❌ deny |
| **Valeria** | ✅ allow | ✅ allow [NUEVO] | ✅ allow | ✅ gate | ⚠️ CEO | ✅ allow [NUEVO] | ❌ deny |
| **Renata** | ✅ allow | ✅ allow [NUEVO] | ✅ allow | ✅ gate | ⚠️ CEO | ✅ allow [NUEVO] | ❌ deny |
| **Uriel** | ✅ allow | ✅ allow [NUEVO] | ✅ allow | ✅ gate | ⚠️ CEO | ✅ allow [NUEVO] | ❌ deny |
| **Elena** | ✅ allow | ✅ allow [NUEVO] | ✅ allow | ✅ gate | ⚠️ CEO | ✅ allow [NUEVO] | ❌ deny |
| **Kenji** | ⚠️ read-only | ✅ allow [auditoría] | ✅ allow | ❌ deny | ❌ deny | ❌ deny | ❌ deny |


## 2. Qué cambió

### 2.1 Handoff approval — delegado a todos los leads

**Antes:** Solo Thavren podía aprobar handoffs cross-área.  
**Ahora:** Cada lead puede enviar y recibir handoffs sin aprobación de Thavren. El protocolo de coordinación es vinculante entre pares.

### 2.2 External directories — delegado a todos los leads

**Antes:** Solo Thavren tenía `*: allow`.  
**Ahora:** Todos los leads tienen acceso a `/home/braka/*` para trabajo cross-sistema. Esto ya se implementó en G-01.

### 2.3 Coordinación de proyectos — Dante

Dante tiene autoridad para:
- Definir `integration_contract.yaml` para cualquier proyecto multi-área.
- Establecer deadlines vinculantes para otros leads.
- Escalar bloqueos a Thavren si un lead no responde en deadline.


## 3. Lo que NO se delega (siempre requiere CEO)

| Acción | Requiere |
|--------|----------|
| PR merge a main/master/develop | CEO approval |
| Protected branch waiver | CEO (crea el archivo manualmente) |
| Nuevos perfiles públicos | CEO authorization |
| Production ready claim | F0 validators PASS + CEO |
| Global ready claim | Production ready + smoke tests + CEO |
| Force push / force delete | PROHIBIDO sin waiver CEO |


## 4. Step-up requerido

| Lead | Step-up para |
|------|-------------|
| Thavren | global_write, plugin_install, production_claim |
| Todos los demás | PR merge, force push, producción |


## 5. Implementación técnica

La delegación de `external_directory: *: allow` ya se aplicó en `tools/permissions/ovav_permission_authority.py` (G-01).  
La delegación de handoff approval se implementa vía `coordination_protocol.yaml` (Capa 2).  
La autoridad de Dante como coordinador se define en `organizational_architecture.yaml` (Capa 1).  

**Sin cambios requeridos en permission_authority.json.** La matriz actual ya soporta estos permisos. Solo se documenta y activa.
