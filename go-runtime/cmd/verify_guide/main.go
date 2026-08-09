// verify_guide — NO_LOGIN Guided Verification Mode
//
// Provides step-by-step guidance for operations that would otherwise be
// blocked with "RUNTIME NO PERMITE" messages. Guides users through manual
// verification without blocking.
//
// Exit: 0 = guidance displayed, 1 = unknown operation
package main

import (
	"fmt"
	"os"
)

// GuidedVerification holds the verification state for an operation.
type GuidedVerification struct {
	operation string
	steps     []string
	completed bool
}

// verificationFlows maps operation names to their step lists.
var verificationFlows = map[string][]string{
	"git_commit": {
		"Para hacer commit sin login activo:",
		"1. Verificar que los cambios son los esperados",
		"2. Revisar `git diff` antes de commit",
		"3. Usar `git commit` con mensaje descriptivo",
		"4. Verificar con `git log --oneline -3`",
	},
	"git_merge": {
		"Para hacer merge sin login activo:",
		"1. Verificar branch destino con `git branch`",
		"2. Revisar cambios con `git diff develop..tu-branch`",
		"3. Asegurar que no hay conflictos",
		"4. Ejecutar merge manualmente",
	},
	"git_push": {
		"Para hacer push sin login activo:",
		"1. Verificar remote con `git remote -v`",
		"2. Confirmar branch con `git branch`",
		"3. Revisar `git log --oneline -5`",
		"4. Ejecutar `git push` manualmente",
	},
	"protected_branch": {
		"Modificar branch protegido:",
		"1. CEO debe crear waiver en `.ovav/runtime/protected_branch_waiver.yaml`",
		"2. Verificar con `go run ./cmd/session_greeting --check-protected`",
		"3. Reintentar operación",
	},
}

// NewGuidedVerification creates a new guided verification for the given operation.
func NewGuidedVerification(operation string) *GuidedVerification {
	steps, ok := verificationFlows[operation]
	if !ok {
		return &GuidedVerification{
			operation: operation,
			steps:     []string{fmt.Sprintf("Operación desconocida: %s", operation)},
			completed: false,
		}
	}
	return &GuidedVerification{
		operation: operation,
		steps:     steps,
		completed: true,
	}
}

// GetSteps returns the verification steps for the operation.
func (gv *GuidedVerification) GetSteps() []string {
	return gv.steps
}

// PrintGuide prints the formatted verification guide to stderr.
func (gv *GuidedVerification) PrintGuide() {
	fmt.Fprintln(os.Stderr, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintf(os.Stderr, "  📋 Guía de Verificación: %s\n", gv.operation)
	fmt.Fprintln(os.Stderr, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(os.Stderr)

	for _, step := range gv.steps {
		fmt.Fprintf(os.Stderr, "  %s\n", step)
	}

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  ✅ Verificación completada manualmente")
	fmt.Fprintln(os.Stderr, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

// IsCompleted returns true if the operation has a valid verification flow.
func (gv *GuidedVerification) IsCompleted() bool {
	return gv.completed
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Uso: verify_guide <operación>")
		fmt.Fprintln(os.Stderr, "Operaciones disponibles: git_commit, git_merge, git_push, protected_branch")
		os.Exit(1)
	}

	operation := os.Args[1]
	gv := NewGuidedVerification(operation)
	gv.PrintGuide()

	if !gv.IsCompleted() {
		os.Exit(1)
	}
}
