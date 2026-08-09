# Technical Implementation — OVAV

**Agent:** lead-dante  
**Fecha:** 2026-08-07  
**Status:** ✅ COMPLETADO

---

## Tech Stack

### Frontend
| Layer | Technology | Version |
|-------|------------|---------|
| Framework | Next.js | 14.x |
| Language | TypeScript | 5.x |
| Styling | Tailwind CSS | 3.x |
| Components | shadcn/ui | latest |
| State | Zustand | 4.x |
| Forms | React Hook Form + Zod | latest |
| Auth | Clerk | - |

### Backend
| Layer | Technology | Version |
|-------|------------|---------|
| API | FastAPI | 0.109.x |
| Runtime | OVAV Go Runtime | 3.4.x |
| Database | PostgreSQL (Neon) | 16.x |
| Cache | Redis (Upstash) | - |
| ORM | SQLAlchemy | 2.x |

### Infrastructure
| Service | Provider | Purpose |
|---------|----------|---------|
| Frontend | Cloudflare Pages | Static hosting |
| Edge | Cloudflare Workers | SSR, API |
| Backend | Fly.io | FastAPI |
| Database | Neon | PostgreSQL |
| Storage | Cloudflare R2 | Assets |
| CDN | Cloudflare | Global delivery |

---

## API Endpoints

### Authentication
```
POST   /api/auth/register     → Create account
POST   /api/auth/login        → Magic link / OAuth
POST   /api/auth/logout      → End session
GET    /api/auth/me          → Current user
PATCH  /api/auth/me          → Update profile
```

### Projects
```
GET    /api/projects          → List user projects
POST   /api/projects          → Create project
GET    /api/projects/:id     → Get project details
PATCH  /api/projects/:id     → Update project
DELETE /api/projects/:id     → Delete project
```

### Teams
```
GET    /api/teams            → List teams
POST   /api/teams            → Create team
GET    /api/teams/:id        → Get team details
PATCH  /api/teams/:id        → Update team
DELETE /api/teams/:id        → Delete team
POST   /api/teams/:id/members → Add member
DELETE /api/teams/:id/members/:userId → Remove member
```

### Billing
```
GET    /api/billing/plans    → List plans
POST   /api/billing/checkout → Create checkout session
GET    /api/billing/portal   → Customer portal
GET    /api/billing/subscription → Current subscription
```

### Webhooks
```
POST   /api/webhooks/stripe  → Stripe events
POST   /api/webhooks/github   → GitHub events
```

---

## Database Schema

### Users
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255),
    password_hash VARCHAR(255),
    avatar_url TEXT,
    plan VARCHAR(50) DEFAULT 'free',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

### Organizations
```sql
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    owner_id UUID REFERENCES users(id),
    plan VARCHAR(50) DEFAULT 'free',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

### Projects
```sql
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    org_id UUID REFERENCES organizations(id),
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

### Team Members
```sql
CREATE TABLE team_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID REFERENCES organizations(id),
    user_id UUID REFERENCES users(id),
    role VARCHAR(50) DEFAULT 'member',
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(org_id, user_id)
);
```

---

## Auth Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         AUTHENTICATION FLOW                              │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  1. User visits app.ovav.dev                                         │
│                                                                          │
│  2. Clerk checks session:                                             │
│     ├─► Has valid token → Dashboard                                   │
│     └─► No token → Login page                                         │
│                                                                          │
│  3. Login options:                                                   │
│     ├─► Magic link (email)                                           │
│     ├─► Google OAuth                                                 │
│     └─► GitHub OAuth                                                 │
│                                                                          │
│  4. After auth:                                                      │
│     ├─► Clerk creates session                                        │
│     ├─► JWT stored in httpOnly cookie                               │
│     └─► Redirect to dashboard                                         │
│                                                                          │
│  5. API requests include JWT:                                        │
│     Authorization: Bearer <token>                                     │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Component Inventory

### Base Components
| Component | States | Description |
|-----------|--------|-------------|
| Button | default, hover, active, disabled, loading | Primary actions |
| Input | default, focus, error, disabled | Form inputs |
| Card | default, hover | Container |
| Modal | open, closed | Dialog overlay |
| Dropdown | open, closed | Selection menu |
| Badge | default, success, warning, error | Status indicator |
| Avatar | with-image, initials | User image |
| Skeleton | loading | Placeholder |

### Feature Components
| Component | Description |
|-----------|-------------|
| PricingTable | Plan comparison |
| FeatureGrid | Features showcase |
| TestimonialSlider | Social proof |
| Navbar | Navigation |
| Footer | Site footer |
| Dashboard | User dashboard layout |
| ProjectCard | Project preview |
| TeamMemberList | Team management |

---

## Testing Strategy

### Unit Tests
```bash
# Jest for frontend
pnpm test

# pytest for backend
pytest
```

### Integration Tests
```bash
# API tests
pytest tests/integration/

# E2E tests
pnpm test:e2e
```

### Coverage Targets
| Layer | Target |
|-------|--------|
| Components | 80% |
| Hooks | 90% |
| API routes | 90% |
| Utils | 100% |

---

## Deployment

### CI/CD Pipeline
```yaml
name: Deploy
on:
  push:
    branches: [main, develop]
  release:
    types: [published]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: npm ci
      - run: npm test

  deploy:
    needs: test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Deploy to Cloudflare
        uses: cloudflare/pages-action@v1
        with:
          apiToken: ${{ secrets.CLOUDFLARE_API_TOKEN }}
          accountId: ${{ secrets.CLOUDFLARE_ACCOUNT_ID }}
          projectName: ovav-landing
```

### Environments
| Branch | URL | Purpose |
|--------|-----|---------|
| main | ovav.dev | Production |
| develop | staging.ovav.dev | Staging |
| pr-* | pr-*.ovav-dev.pages.dev | Preview |

---

## Environment Variables

```bash
# Frontend (.env.local)
NEXT_PUBLIC_API_URL=https://api.ovav.dev
NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY=pk_xxx
CLERK_SECRET_KEY=sk_xxx

# Backend (.env)
DATABASE_URL=postgresql://user:pass@xxx.neon.tech/db
REDIS_URL=redis://xxx.upstash.io
STRIPE_SECRET_KEY=sk_xxx
STRIPE_WEBHOOK_SECRET=whsec_xxx
CLERK_SECRET_KEY=sk_xxx
```

---

## Security

- [x] HTTPS everywhere
- [x] JWT with short expiry
- [x] Refresh token rotation
- [x] CSRF protection
- [x] Rate limiting (100 req/min)
- [x] Input validation (Zod)
- [x] SQL injection prevention (ORM)
- [x] XSS protection (React)
- [x] Secure headers (Cloudflare)

---

*Document: 2026-08-07*
