# Model Body Router — Cambio de Cuerpo sin Pérdida de Identidad

## Principio

El modelo AI es el **cuerpo operativo**. La identidad profesional (Thavren, Eidren) es independiente del cuerpo. OVAV debe poder cambiar de modelo automáticamente sin que la identidad, el criterio, la memoria ni la personalidad se degraden.

---

## Arquitectura del Router

```text
                    ACTIVE IDENTITY PACKET
                    (invariable entre cuerpos)
                            │
                            ▼
┌───────────────────────────────────────────────────┐
│              MODEL BODY ROUTER                     │
│                                                    │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────┐ │
│  │ HEALTH       │  │ PROVIDER     │  │ IDENTITY │ │
│  │ CHECKS       │  │ ABSTRACTION  │  │ GUARD    │ │
│  │              │  │              │  │          │ │
│  │ · créditos   │  │ · prompt norm│  │ · mismo  │ │
│  │ · latencia   │  │ · tool norm  │  │   packet │ │
│  │ · errores    │  │ · output norm│  │ · mismo  │ │
│  │ · capability │  │ · context    │  │   contr. │ │
│  └──────┬───────┘  └──────┬───────┘  └────┬─────┘ │
└─────────┼─────────────────┼───────────────┼───────┘
          ▼                 ▼               ▼
   ┌──────────┐      ┌──────────┐     ┌──────────┐
   │ DeepSeek │      │  Claude  │     │  Gemini  │
   │ (cuerpo) │      │ (cuerpo) │     │ (cuerpo) │
   └──────────┘      └──────────┘     └──────────┘
```

---

## Health Checks — ¿Cuándo cambiar de cuerpo?

| Trigger | Threshold | Acción |
|---|---|---|
| **Créditos agotados** | API retorna 402/429 | Switch inmediato a fallback |
| **Latencia excesiva** | p95 > 3x baseline | Degradar a modelo más rápido |
| **Tasa de error** | >10% en últimos N requests | Switch a fallback |
| **Capability mismatch** | Tarea requiere tool-use y el cuerpo no lo soporta | Enrutar a cuerpo compatible |
| **Ventana de contexto** | Tarea requiere >200K tokens y el cuerpo solo soporta 128K | Enrutar a cuerpo con ventana mayor |
| **Costo excesivo** | Costo/task > threshold configurado | Degradar a modelo más barato |

---

## Provider Abstraction Layer — El Sistema Nervioso

Cada modelo interpreta prompts, tool schemas y formatos de manera distinta. La capa de abstracción normaliza:

### 1. System Prompt Normalizer

```text
Packet YAML → System Prompt para DeepSeek
Packet YAML → System Prompt para Claude  
Packet YAML → System Prompt para Gemini

El packet es invariante. El normalizer adapta:
  · Tono y estructura por provider
  · Instrucciones de tool-use específicas
  · Formato de output esperado
  · Reglas de seguridad nativas del provider
```

### 2. Tool Schema Normalizer

```text
OVAV Tool Schema (canónico)
  → OpenAI function calling format
  → Anthropic tool_use format
  → DeepSeek function calling format

Cada provider recibe el schema en su formato nativo.
Las respuestas se normalizan de vuelta al formato OVAV.
```

### 3. Output Format Normalizer

```text
Respuesta del modelo (formato nativo)
  → Normalizar JSON
  → Normalizar streaming
  → Normalizar function calls
  → Verificar contra delivery contract
```

### 4. Context Window Adapter

```text
Si el cuerpo destino tiene menos contexto que el actual:
  · Comprimir contexto (summarization)
  · Recortar ventana (sliding window)
  · Preservar identity packet + decisiones activas
  · Descartar logs crudos y ruido
```

---

## Identity Guard — Verificación post-switch

Después de cada cambio de cuerpo, el Identity Guard verifica:

```text
1. ¿El packet se preservó intacto?
   → Hash del packet antes y después del switch

2. ¿El delivery contract se mantiene?
   → Sampleo de output: tono, idioma, estructura

3. ¿Las herramientas funcionan?
   → Smoke test: una tool read, una tool status

4. ¿La memoria operativa está presente?
   → active_decisions y blocked_surfaces en el nuevo contexto

Si alguna verificación falla:
   → Reintentar con otro cuerpo
   → Si todos fallan, emitir SAFE STOP
```

---

## Referencias de la Industria

| Sistema | Patrón | Lo que OVAV mejora |
|---|---|---|
| **OpenRouter** | Round-robin + error-rate-weighted routing | OVAV añade identity preservation |
| **LangChain** | `with_fallbacks()` a nivel prompt | OVAV lo hace a nivel infraestructura |
| **CrewAI** | LLM wrapper classes | OVAV abstrae también tool schemas y output format |
| **AWS Bedrock** | Cross-region inference routing | OVAV añade capability matching |

**Ventaja competitiva de OVAV**: ningún sistema en 2026 preserva identidad profesional completa entre modelos distintos. El Active Identity Packet + Provider Abstraction Layer es una arquitectura única.

---

## Estado Actual

| Componente | Estado |
|---|---|
| Health Checks | ✅ Implementado / integrado en runtime health and self-diagnosis |
| Provider Abstraction | ✅ Implementado |
| Identity Guard | ✅ Implementado |
| Active Identity Packet | ✅ Implementado para Thavren + Eidren |

**Dependencia actual**: El Model Body Router requiere el Active Identity Packet Compiler (Layer 0) como prerrequisito, ya implementado. La siguiente mejora estratégica es que el Knowledge Compiler aporte evidencia histórica, criterios compilados y señales de transición para decisiones de fallback y continuidad entre cuerpos.
