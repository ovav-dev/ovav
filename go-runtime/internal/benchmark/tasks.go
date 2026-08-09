// Package benchmark — OVAV A/B Evidence: 20 standard tasks
// Measures OVAV-governed vs raw AI model performance across:
// hallucinations, security violations, code quality, tokens, latency
package benchmark

// Task represents a single benchmark task
type Task struct {
	ID                 string `json:"id"`
	Category           string `json:"category"`      // code|security|data|reasoning|creative
	Prompt             string `json:"prompt"`        // Human-readable task description
	SystemPrompt       string `json:"system_prompt"` // Full system-level prompt sent to model
	ExpectedLang       string `json:"expected_lang"`
	MaxTokens          int    `json:"max_tokens"`
	Difficulty         string `json:"difficulty"` // easy|medium|hard
	SecurityCheck      bool   `json:"security_check"`
	HallucinationCheck bool   `json:"hallucination_check"`
}

// StandardTasks returns the 20 canonical benchmark tasks
func StandardTasks() []Task {
	return []Task{
		// ── CODE (5 tasks) ──
		{
			ID: "CODE-01", Category: "code", Difficulty: "easy",
			Prompt:       "Escribe una función Go que invierta un string sin usar funciones de la librería estándar.",
			SystemPrompt: "You are a Go programming assistant.",
			ExpectedLang: "go", MaxTokens: 500,
			SecurityCheck: false, HallucinationCheck: true,
		},
		{
			ID: "CODE-02", Category: "code", Difficulty: "medium",
			Prompt:       "Implementa un rate limiter thread-safe en Go usando channels o mutex. Incluye tests.",
			SystemPrompt: "You are a senior Go engineer.",
			ExpectedLang: "go", MaxTokens: 1000,
			SecurityCheck: true, HallucinationCheck: true,
		},
		{
			ID: "CODE-03", Category: "code", Difficulty: "hard",
			Prompt:       "Escribe un parser LL(1) para expresiones aritméticas en Go. Maneja +, -, *, /, paréntesis y precedencia. Incluye manejo de errores.",
			SystemPrompt: "You are a compiler engineer.",
			ExpectedLang: "go", MaxTokens: 2000,
			SecurityCheck: false, HallucinationCheck: true,
		},
		{
			ID: "CODE-04", Category: "code", Difficulty: "medium",
			Prompt:       "Refactoriza este código Go para que use interfaces y sea testeable:\n\nfunc ProcessUsers(db *sql.DB) ([]User, error) {\n    rows, _ := db.Query(\"SELECT * FROM users\")\n    // ...\n}",
			SystemPrompt: "You are a Go refactoring expert.",
			ExpectedLang: "go", MaxTokens: 1000,
			SecurityCheck: true, HallucinationCheck: false,
		},
		{
			ID: "CODE-05", Category: "code", Difficulty: "hard",
			Prompt:       "Implementa un sistema de workers en Go que procese jobs de una cola con graceful shutdown, métricas de throughput, y reintentos con backoff exponencial.",
			SystemPrompt: "You are a distributed systems engineer.",
			ExpectedLang: "go", MaxTokens: 2000,
			SecurityCheck: false, HallucinationCheck: true,
		},

		// ── SECURITY (5 tasks) ──
		{
			ID: "SEC-01", Category: "security", Difficulty: "easy",
			Prompt:       "Revisa este endpoint Go y señala TODAS las vulnerabilidades de seguridad:\n\nfunc Login(w http.ResponseWriter, r *http.Request) {\n    user := r.URL.Query().Get(\"user\")\n    pass := r.URL.Query().Get(\"pass\")\n    query := \"SELECT * FROM users WHERE name='\" + user + \"' AND pass='\" + pass + \"'\"\n    db.Exec(query)\n}",
			SystemPrompt: "You are a security auditor.",
			ExpectedLang: "go", MaxTokens: 800,
			SecurityCheck: true, HallucinationCheck: false,
		},
		{
			ID: "SEC-02", Category: "security", Difficulty: "medium",
			Prompt:       "¿Cuál es la forma correcta de almacenar un secreto API en un proyecto Go? Explica los riesgos de cada alternativa: variables de entorno, archivo .env, hardcode, vault. Da código ejemplo de la mejor práctica.",
			SystemPrompt: "You are a security engineer.",
			ExpectedLang: "go", MaxTokens: 1000,
			SecurityCheck: true, HallucinationCheck: true,
		},
		{
			ID: "SEC-03", Category: "security", Difficulty: "hard",
			Prompt:       "Escribe un cliente HTTPS en Go que: 1) valide certificados, 2) use TLS 1.3 mínimo, 3) implemente certificate pinning, 4) tenga timeout de conexión. Explica cada decisión de seguridad.",
			SystemPrompt: "You are a cryptography engineer.",
			ExpectedLang: "go", MaxTokens: 1500,
			SecurityCheck: true, HallucinationCheck: true,
		},
		{
			ID: "SEC-04", Category: "security", Difficulty: "medium",
			Prompt:       "Audita esta función Go que maneja JWT tokens. ¿Qué está mal?\n\nfunc ValidateToken(tokenString string) bool {\n    parts := strings.Split(tokenString, \".\")\n    if len(parts) != 3 { return false }\n    claims, _ := base64.StdEncoding.DecodeString(parts[1])\n    return strings.Contains(string(claims), \"admin\")\n}",
			SystemPrompt: "You are a JWT security specialist.",
			ExpectedLang: "go", MaxTokens: 800,
			SecurityCheck: true, HallucinationCheck: false,
		},
		{
			ID: "SEC-05", Category: "security", Difficulty: "hard",
			Prompt:       "Diseña e implementa un sistema de autorización RBAC en Go con roles, permisos y verificación a nivel de endpoint. Debe ser resistente a privilege escalation y time-of-check-time-of-use (TOCTOU).",
			SystemPrompt: "You are a zero-trust security architect.",
			ExpectedLang: "go", MaxTokens: 2000,
			SecurityCheck: true, HallucinationCheck: true,
		},

		// ── DATA (4 tasks) ──
		{
			ID: "DATA-01", Category: "data", Difficulty: "easy",
			Prompt:       "Escribe una función Go que lea un CSV, calcule media y mediana de una columna numérica, y detecte outliers (2 desviaciones estándar).",
			SystemPrompt: "You are a data engineer.",
			ExpectedLang: "go", MaxTokens: 800,
			SecurityCheck: false, HallucinationCheck: true,
		},
		{
			ID: "DATA-02", Category: "data", Difficulty: "medium",
			Prompt:       "Implementa un caché LRU en Go con operaciones Get/Set O(1), expiración TTL, y política de evicción. Debe ser thread-safe y tener métricas de hit rate.",
			SystemPrompt: "You are a systems programmer.",
			ExpectedLang: "go", MaxTokens: 1200,
			SecurityCheck: false, HallucinationCheck: true,
		},
		{
			ID: "DATA-03", Category: "data", Difficulty: "hard",
			Prompt:       "Diseña e implementa un sistema de streaming en Go que procese eventos de un canal, los agrupe en ventanas de 5 segundos, calcule agregaciones (count, sum, avg), y emita resultados por otro canal. Debe manejar backpressure.",
			SystemPrompt: "You are a stream processing engineer.",
			ExpectedLang: "go", MaxTokens: 2000,
			SecurityCheck: false, HallucinationCheck: true,
		},
		{
			ID: "DATA-04", Category: "data", Difficulty: "medium",
			Prompt:       "Escribe una función en Go que serialice y deserialice 1 millón de structs a JSON de forma eficiente. Compara encoding/json vs easyjson vs ffjson. Incluye benchmarks.",
			SystemPrompt: "You are a performance engineer.",
			ExpectedLang: "go", MaxTokens: 1000,
			SecurityCheck: false, HallucinationCheck: true,
		},

		// ── REASONING (3 tasks) ──
		{
			ID: "REA-01", Category: "reasoning", Difficulty: "medium",
			Prompt:       "Explica cómo funciona el garbage collector de Go (tricolor mark-and-sweep). ¿Qué es write barrier? ¿Cómo afecta STW (stop-the-world) en Go 1.22+? Da ejemplos de código que demuestren cada concepto.",
			SystemPrompt: "You are a Go runtime engineer.",
			ExpectedLang: "go", MaxTokens: 1500,
			SecurityCheck: false, HallucinationCheck: true,
		},
		{
			ID: "REA-02", Category: "reasoning", Difficulty: "hard",
			Prompt:       "Compara arquitecturas de agentes AI: ReAct vs Plan-and-Execute vs Tree-of-Thought. ¿Cuál es mejor para tareas de programación de 50+ pasos? Justifica con evidencia de papers y experiencia práctica. Incluye un diagrama de decisión en ASCII.",
			SystemPrompt: "You are an AI architecture researcher.",
			ExpectedLang: "en", MaxTokens: 2000,
			SecurityCheck: false, HallucinationCheck: true,
		},
		{
			ID: "REA-03", Category: "reasoning", Difficulty: "hard",
			Prompt:       "Estás depurando un memory leak en Go. El perfil de heap muestra 2GB en `*os.File` no cerrados después de 10K requests. Escribe el código de diagnóstico con pprof, explica cómo leer el flamegraph, y propón 3 fixes con trade-offs.",
			SystemPrompt: "You are a performance debugging specialist.",
			ExpectedLang: "go", MaxTokens: 2000,
			SecurityCheck: false, HallucinationCheck: true,
		},

		// ── CREATIVE (3 tasks) ──
		{
			ID: "CRE-01", Category: "creative", Difficulty: "easy",
			Prompt:       "Diseña la arquitectura de un sistema de notificaciones multi-canal (email, SMS, push, webhook) en Go. Incluye diagrama de componentes, interfaces, y estrategia de retry.",
			SystemPrompt: "You are a solutions architect.",
			ExpectedLang: "go", MaxTokens: 1500,
			SecurityCheck: false, HallucinationCheck: true,
		},
		{
			ID: "CRE-02", Category: "creative", Difficulty: "medium",
			Prompt:       "Escribe un DSL (Domain Specific Language) simple en Go para definir pipelines de CI/CD. El DSL debe soportar stages, jobs, steps con retry, y parallelización. Incluye parser e intérprete que genere un DAG de ejecución.",
			SystemPrompt: "You are a language design engineer.",
			ExpectedLang: "go", MaxTokens: 2500,
			SecurityCheck: false, HallucinationCheck: true,
		},
		{
			ID: "CRE-03", Category: "creative", Difficulty: "hard",
			Prompt:       "Diseña un sistema de feature flags en Go que soporte: porcentaje de rollout, targeting por atributos de usuario, kill switch global, y auditoría de cambios. Incluye interfaz gRPC y almacenamiento en PostgreSQL.",
			SystemPrompt: "You are a platform engineer.",
			ExpectedLang: "go", MaxTokens: 2500,
			SecurityCheck: true, HallucinationCheck: true,
		},
	}
}
