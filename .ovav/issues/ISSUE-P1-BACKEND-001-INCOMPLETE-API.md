# ISSUE #P1-BACKEND-001: Backend API Incompleta

## metadata
- **severity**: high
- **created**: 2026-08-07
- **layer**: backend
- **plan_ref**: plans/OVAV-WEB-FULLSTACK-2026-08-07.md §1.2
- **status**: IN_PROGRESS
- **owner**: thavren

## diagnosis
### Current State
Backend FastAPI existe pero falta endpoints críticos según plan:

| Endpoint | Estado | Gap |
|----------|--------|-----|
| `/auth/register` | ✅ | - |
| `/auth/login` | ✅ | - |
| `/auth/verify` | ✅ | - |
| `/auth/refresh` | ❌ | Falta |
| `/auth/logout` | ❌ | Falta |
| `/auth/me` | ❌ | Falta |
| `/users/me` | ❌ | Falta |
| `/users/me` PATCH | ❌ | Falta |
| `/licenses` | ⚠️ | Parcial |
| `/licenses/validate` | ✅ | - |
| `/licenses/{id}` | ❌ | Falta |
| `/licenses/{id}/activate` | ❌ | Falta |
| `/api-keys/*` | ❌ | No existe |
| `/billing/*` | ❌ | No existe |
| `/download/*` | ❌ | No existe |
| `/checkout/*` | ✅ | - |

### Models Faltantes
- [ ] `api_key.py` - API Keys table
- [ ] `invoice.py` - Invoices table  
- [ ] `refresh_token.py` - Token rotation
- [ ] `user.py` - Extensions (email_verified, etc.)

## tasks
- [ ] **T1** Crear `/app/api/v1/users.py`
- [ ] **T2** Crear `/app/api/v1/api_keys.py`
- [ ] **T3** Crear `/app/api/v1/billing.py`
- [ ] **T4** Crear `/app/api/v1/download.py`
- [ ] **T5** Crear `/app/models/api_key.py`
- [ ] **T6** Crear `/app/models/invoice.py`
- [ ] **T7** Completar `/app/api/auth.py` (refresh, logout, me)
- [ ] **T8** Crear migrations con Alembic

## validation
```bash
cd /home/braka/Systems/OVAV/web/backend
pytest tests/ -v --cov=. --cov-report=term-missing
fastapi dev app/main.py
# Probar endpoints con curl
```

## resolution
END
