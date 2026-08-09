package runtime

import "testing"

// ── EvaluateNewIdea tests ──────────────────────────────────────────

func TestEvaluateNewIdea_NewProject(t *testing.T) {
	tests := []struct {
		input       string
		wantNewIdea bool
		wantCat     string
	}{
		{"quiero hacer una nueva app", true, "new_project"},
		{"nuevo sistema de gestión", true, "new_project"},
		{"construir un dashboard", true, "new_project"},
		{"crear un CRM", true, "new_project"},
		{"start a project", true, "new_project"},
		{"new app", true, "new_project"},
		{"completo de cero", true, "new_project"},
		{"full system", true, "new_project"},
		{"fix el bug del login", false, ""},
		{"continuar con el plan", false, ""},
	}

	for _, tt := range tests {
		result := EvaluateNewIdea(tt.input)
		if result.IsNewIdea != tt.wantNewIdea {
			t.Errorf("EvaluateNewIdea(%q).IsNewIdea = %v, want %v", tt.input, result.IsNewIdea, tt.wantNewIdea)
		}
		if tt.wantNewIdea && result.Category != tt.wantCat {
			t.Errorf("EvaluateNewIdea(%q).Category = %q, want %q", tt.input, result.Category, tt.wantCat)
		}
	}
}

func TestEvaluateNewIdea_FeatureRequest(t *testing.T) {
	tests := []struct {
		input       string
		wantNewIdea bool
		wantCat     string
	}{
		{"agregar dark mode", true, "feature_request"},
		{"implementar caching", true, "feature_request"},
		{"nueva feature de exportación", true, "feature_request"},
		{"añadir soporte para WebDAV", true, "feature_request"},
		{"add support for S3", true, "feature_request"},
		{"me gustaria agregar un dashboard", true, "feature_request"},
	}

	for _, tt := range tests {
		result := EvaluateNewIdea(tt.input)
		if result.IsNewIdea != tt.wantNewIdea {
			t.Errorf("EvaluateNewIdea(%q).IsNewIdea = %v, want %v", tt.input, result.IsNewIdea, tt.wantNewIdea)
		}
		if tt.wantNewIdea && result.Category != tt.wantCat {
			t.Errorf("EvaluateNewIdea(%q).Category = %q, want %q", tt.input, result.Category, tt.wantCat)
		}
	}
}

func TestEvaluateNewIdea_ScopeDeviation(t *testing.T) {
	tests := []struct {
		input       string
		wantNewIdea bool
		wantCat     string
	}{
		{"en vez de React usa Vue", true, "scope_deviation"},
		{"cambiar a PostgreSQL", true, "scope_deviation"},
		{"migrate to go", true, "scope_deviation"},
		{"instead of the current approach", true, "scope_deviation"},
	}

	for _, tt := range tests {
		result := EvaluateNewIdea(tt.input)
		if result.IsNewIdea != tt.wantNewIdea {
			t.Errorf("EvaluateNewIdea(%q).IsNewIdea = %v, want %v", tt.input, result.IsNewIdea, tt.wantNewIdea)
		}
		if tt.wantNewIdea && result.Category != tt.wantCat {
			t.Errorf("EvaluateNewIdea(%q).Category = %q, want %q", tt.input, result.Category, tt.wantCat)
		}
	}
}

func TestEvaluateNewIdea_ExternalSystem(t *testing.T) {
	tests := []struct {
		input       string
		wantNewIdea bool
		wantCat     string
	}{
		{"conectar a Stripe", true, "external_system"},
		{"integrar con Salesforce", true, "external_system"},
		{"usar API de GitHub", true, "external_system"},
		{"webhook de autenticación", true, "external_system"},
		{"conectar con Firebase", true, "external_system"},
	}

	for _, tt := range tests {
		result := EvaluateNewIdea(tt.input)
		if result.IsNewIdea != tt.wantNewIdea {
			t.Errorf("EvaluateNewIdea(%q).IsNewIdea = %v, want %v", tt.input, result.IsNewIdea, tt.wantNewIdea)
		}
		if tt.wantNewIdea && result.Category != tt.wantCat {
			t.Errorf("EvaluateNewIdea(%q).Category = %q, want %q", tt.input, result.Category, tt.wantCat)
		}
	}
}

func TestEvaluateNewIdea_Severity(t *testing.T) {
	// new_project and scope_deviation should be "hard"
	result := EvaluateNewIdea("quiero hacer una nueva app")
	if result.Severity != "hard" {
		t.Errorf("severity = %q, want hard for new_project", result.Severity)
	}

	result2 := EvaluateNewIdea("en vez de React usa Vue")
	if result2.Severity != "hard" {
		t.Errorf("severity = %q, want hard for scope_deviation", result2.Severity)
	}

	// feature_request should be "soft"
	result3 := EvaluateNewIdea("agregar dark mode")
	if result3.Severity != "soft" {
		t.Errorf("severity = %q, want soft for feature_request", result3.Severity)
	}
}

func TestEvaluateNewIdea_Recommendation(t *testing.T) {
	// Any match = BRAINSTORM
	result := EvaluateNewIdea("quiero hacer una nueva app")
	if result.Recommendation != "BRAINSTORM" {
		t.Errorf("recommendation = %q, want BRAINSTORM", result.Recommendation)
	}

	// No match = EXECUTE
	result2 := EvaluateNewIdea("fix the login bug")
	if result2.Recommendation != "EXECUTE" {
		t.Errorf("recommendation = %q, want EXECUTE", result2.Recommendation)
	}
}

func TestEvaluateNewIdea_TaskPreserved(t *testing.T) {
	task := "quiero hacer una nueva app de gestión"
	result := EvaluateNewIdea(task)
	if result.Task != task {
		t.Errorf("Task = %q, want %q", result.Task, task)
	}
}

// ── DetectMultiArea tests ──────────────────────────────────────────

func TestDetectMultiArea_SingleArea(t *testing.T) {
	result := DetectMultiArea("fix the login bug in the backend api")
	if result.IsMultiArea {
		t.Error("single area task should not be multi-area")
	}
	if result.PrimaryArea == "" {
		t.Error("PrimaryArea should not be empty")
	}
}

func TestDetectMultiArea_MultipleAreas(t *testing.T) {
	// Task mentions both frontend and backend
	result := DetectMultiArea("add dark mode to the frontend and database schema to the backend")
	if !result.IsMultiArea {
		t.Error("should detect multi-area")
	}
	if len(result.AreasFound) < 2 {
		t.Errorf("AreasFound len = %d, want >= 2", len(result.AreasFound))
	}
}

func TestDetectMultiArea_AreaKeywords(t *testing.T) {
	tests := []struct {
		task     string
		wantArea string
	}{
		{"backend api with database sql", "platform_engineering"},
		{"frontend react css design", "ux_design"},
		{"research analysis benchmark study", "research_intelligence"},
		{"devops kubernetes docker deployment", "devops_infrastructure"},
		{"fullstack app web product", "digital_product"},
	}

	for _, tt := range tests {
		result := DetectMultiArea(tt.task)
		if result.PrimaryArea != tt.wantArea {
			t.Errorf("DetectMultiArea(%q).PrimaryArea = %q, want %q", tt.task, result.PrimaryArea, tt.wantArea)
		}
	}
}

// ── CalibrateEffort tests ─────────────────────────────────────────

func TestCalibrateEffort_Epic(t *testing.T) {
	tests := []string{
		"todo el sistema completo",
		"full system from scratch",
		"entire system architecture",
	}

	for _, task := range tests {
		result := CalibrateEffort(task)
		if result.Effort != "epic" {
			t.Errorf("CalibrateEffort(%q).Effort = %q, want epic", task, result.Effort)
		}
	}
}

func TestCalibrateEffort_Complex(t *testing.T) {
	tests := []string{
		"nueva arquitectura de microservicios",
		"new system design with event sourcing",
	}

	for _, task := range tests {
		result := CalibrateEffort(task)
		if result.Effort != "complex" {
			t.Errorf("CalibrateEffort(%q).Effort = %q, want complex", task, result.Effort)
		}
	}
}

func TestCalibrateEffort_Simple(t *testing.T) {
	tests := []string{
		"fix the typo in README",
		"quick fix for null pointer",
		"un pequeño fix en el parser",
	}

	for _, task := range tests {
		result := CalibrateEffort(task)
		if result.Effort != "simple" {
			t.Errorf("CalibrateEffort(%q).Effort = %q, want simple", task, result.Effort)
		}
	}
}

func TestCalibrateEffort_NoSignals(t *testing.T) {
	// Use input that matches no signal patterns
	result := CalibrateEffort("process the user data through the pipeline")
	if result.Effort != "moderate" {
		t.Errorf("CalibrateEffort with no signals = %q, want moderate", result.Effort)
	}
}

// ── PlanExistsResult tests ─────────────────────────────────────────

func TestCheckPlanExists(t *testing.T) {
	result := CheckPlanExists("test-project")
	// Filesystem access is disabled in CLI context for this function
	// so it always returns false. This is the expected behavior.
	if result.Exists {
		t.Error("CheckPlanExists should return Exists=false (no filesystem)")
	}
}
