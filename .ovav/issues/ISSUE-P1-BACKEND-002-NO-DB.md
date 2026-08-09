# ISSUE #P1-BACKEND-002: Database Connection Failed

## metadata
- **severity**: high
- **created**: 2026-08-07
- **layer**: backend
- **plan_ref**: plans/OVAV-WEB-FULLSTACK-2026-08-07.md §1.2.10
- **status**: IN_PROGRESS
- **owner**: thavren

## diagnosis
PostgreSQL no está disponible en entorno local. Error: `TimeoutError` al conectar a `localhost:5432`.

### Solución
Modificar database.py para fallback a SQLite en desarrollo local.

## tasks
- [ ] **T1** Modificar database.py con fallback SQLite
- [ ] **T2** Agregar modo test/development

## resolution
END
