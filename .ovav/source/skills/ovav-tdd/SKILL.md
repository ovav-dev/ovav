# OVAV TDD Mode — Skill

> **Versión:** 1.0 | **Fecha:** 2026-07-30 | **Lead:** Thavren

---

## Nombre
`ovav-tdd`

## Descripción
Test-Driven Development con iron law obligatoria. Todo código de producción requiere test que falla primero.

---

## Iron Law

```
NUNCA CÓDIGO DE PRODUCCIÓN SIN TEST QUE FALLA PRIMERO
```

**No hay excepciones:**
- No "después escribo los tests"
- No "es código simple, no necesita test"
- No "ya corrió antes, seguro funciona"

Si escribiste código antes que el test → borralo. Empieza de nuevo.

---

## Red-Green-Refactor

### RED — Escribir test que falla
1. Escribir el test que describe el comportamiento esperado
2. Ejecutar el test
3. **Verificar que falla** (failure correcto = wrong output, no compile error)
4. Si no falla → el test está mal escrito → arreglar test

### GREEN — Código mínimo para pasar
1. Escribir EL MÍNIMO código necesario para que el test pase
2. No optimizaciones — solo lo suficiente
3. Ejecutar tests → todos pasan
4. Si algo falla → el código está mal → arreglar código

### REFACTOR — Limpiar sin romper
1. Limpiar código (nombres, duplicación, estructura)
2. EJECUTAR TESTS después de cada limpieza
3. Si algún test falla → undo inmediato → stay in REFACTOR
4. Una vez todos verdes → siguiente test

---

## Go TDD Pattern

```go
// 1. RED: Escribir test que falla
func TestHashPassword_GeneratesBcryptHash(t *testing.T) {
    // arrange
    password := "testpassword123"

    // act
    hash := hashPassword(password)

    // assert
    if hash == "" {
        t.Error("expected non-empty hash")
    }
    if hash == password {
        t.Error("hash should not equal plaintext")
    }
    // Este test FALLA porque hashPassword no existe aún
}
```

```go
// 2. GREEN: Código mínimo para pasar
func hashPassword(password string) string {
    bytes, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    return string(bytes)
}
```

```go
// 3. REFACTOR: Limpiar
// Agregar verifyPassword, edge cases, etc.
```

---

## Frontend TDD Pattern (Vitest)

```typescript
// 1. RED: Test que falla
describe('authStore', () => {
  it('should store userId and token after login', async () => {
    // arrange
    const { login } = useAuthStore()

    // act
    await login('test@example.com', 'password123')

    // assert
    expect(useAuthStore.getState().isAuthenticated).toBe(true)
    // FALLA porque login no existe aún
  })
})
```

```typescript
// 2. GREEN: Implementación mínima
const useAuthStore = create((set) => ({
  userId: null,
  token: null,
  isAuthenticated: false,
  login: async (email, password) => {
    const resp = await api.login(email, password)
    set({ userId: resp.user_id, token: resp.token, isAuthenticated: true })
  },
}))
```

---

## Coverage Mínimo

| Layer | Mínimo |
|---|---|
| Backend handlers | 80% coverage |
| Backend services | 80% coverage |
| Frontend stores | 80% coverage |
| Frontend hooks | 70% coverage |
| Utils | 90% coverage |

**Coverage bajo 80% → HARD STOP hasta mejorar tests.**

---

## Verification Commands

```bash
# Backend
cd backend && go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | tail -1

# Frontend
cd frontend && pnpm test --coverage
# Verificar coverage > 80% en cada package
```

---

## Metadata

- **Ubicación:** `.ovav/source/skills/ovav-tdd/SKILL.md`
- **Skill ID:** `ovav-tdd`
- **Trigger:** Cualquier implementación de código nuevo
- **Depende de:** Ninguno (independiente)
- **Companion:** `ovav-build` lo invoca automáticamente
