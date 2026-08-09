# SPEC: `ovav delegate` — Native OVAV Delegation Command

## Problema

`actor` tool de MiMoCode solo acepta `explore`/`general` subagent_types.
OVAV tiene 10 leads + 60 teams con perfiles ricos, pero no puede invocarlos
directamente — cae a `general` (sin perfil, sin skills, sin contexto).

## Solución

Nuevo subcommand `ovav delegate` en go-runtime que:

1. Recibe `agent_id` (lead-eidren, team-clara, etc.) + `task` description
2. Carga el perfil del agente desde `.ovav/service_areas/` o `runtimes/`
3. Ejecuta el task INLINE dentro del proceso Go (sin subprocess)
4. Retorna resultado estructurado en JSON

## Diseño

```
ovav delegate <agent_id> <task_text>
ovav delegate --agent lead-eidren --task "Investigar A2A mesh runtime"
ovav delegate --agent team-clara --task "Ejecutar coverage sprint en validators/"
```

### Flag --json-output
Retorna JSON estructurado para consumo por scripts/CI.

### Flag --inline
Ejecuta el task inline (en el mismo proceso Go) — para tasks pequeños.

### Flag --spawn
Spawns el agente como subprocess con stdio pipes — para tasks largos
con feedback interactivo.

## Arquitectura de Ejecución Inline

Para ejecución inline, el delegate command:

1. **Carga agente**: Lee `.runtimes/opencode/agents/<agent_id>.md`
   o `.ovav/service_areas/<area>/agents/<agent_id>.md`

2. **Construye contexto**: Combina:
   - `AGENTS.md` (perfil OVAV del agente)
   - Contexto del workspace actual (git state, archivos)
   - Task description
   - Skills del área

3. **Invoca modelo**: Usa `model` package del go-runtime para generar
   respuesta con el system prompt del agente.

4. **Retorna**: JSON con resultado + metadata.

## Beneficio

OVAVGoverna. Sin pasar por actor tool de MiMoCode.
OVAV agents son ciudadanos de primera clase.
