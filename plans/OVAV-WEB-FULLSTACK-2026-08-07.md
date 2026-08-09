# OVAV Full-Stack Implementation Plan
## WEB 100% → Mobile iOS/Android → Desktop Install Program

```
VERSión: 1.0.0
FECHA: 2026-08-08
AUTHORITY: Thavren — Platform Engineering Lead
STATUS: ACTIVE
NEXT_PHASE: PHASE-1-WEB
```

---

## 📋 Tabla de Contenidos

1. [Resumen Ejecutivo](#1-resumen-ejecutivo)
2. [Arquitectura Objetivo](#2-arquitectura-objetivo)
3. [PHASE 1: Web Full-Stack (8 semanas)](#3-phase-1-web-full-stack-8-semanas)
   - [1.1 Landing Page](#11-landing-page)
   - [1.2 Backend API](#12-backend-api)
   - [1.3 Frontend Dashboard](#13-frontend-dashboard)
   - [1.4 Auth Integration](#14-auth-integration)
   - [1.5 CLI ↔ Backend Bridge](#15-cli--backend-bridge)
4. [PHASE 2: Mobile iOS/Android (6 semanas)](#4-phase-2-mobile-iosandroid-6-semanas)
   - [2.1 Mobile Architecture](#21-mobile-architecture)
   - [2.2 Capacitor Setup](#22-capacitor-setup)
   - [2.3 Mobile Screens](#23-mobile-screens)
5. [PHASE 3: Desktop Install Program (4 semanas)](#5-phase-3-desktop-install-program-4-semanas)
   - [3.1 Install Script](#31-install-script)
   - [3.2 Platform Installers](#32-platform-installers)
6. [Gobernanza y Validación](#6-gobernanza-y-validación)
7. [Definition of Done](#7-definition-of-done)

---

## 1. Resumen Ejecutivo

### Visión del Sistema

```
┌──────────────────────────────────────────────────────────────────────┐
│                         OVAV ECOSYSTEM                              │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│   ┌──────────┐     ┌──────────┐     ┌──────────┐                 │
│   │   WEB    │     │  MOBILE  │     │  DESKTOP │                 │
│   │  ovav.dev│     │ iOS/Android│   │ Windows/Mac│               │
│   └────┬─────┘     └────┬─────┘     └────┬─────┘                 │
│        │                   │                │                       │
│        └───────────────────┼────────────────┘                       │
│                            │                                        │
│                            ▼                                        │
│              ┌─────────────────────────┐                          │
│              │    UNIFIED API GATEWAY   │                          │
│              │    (FastAPI Backend)     │                          │
│              └────────────┬────────────┘                          │
│                           │                                        │
│        ┌─────────────────┼─────────────────┐                    │
│        │                 │                 │                    │
│        ▼                 ▼                 ▼                    │
│   ┌─────────┐     ┌─────────┐     ┌─────────┐                  │
│   │PostgreSQL│     │  Redis  │     │ Stripe  │                  │
│   └─────────┘     └─────────┘     └─────────┘                  │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

### Objetivos por Phase

| Phase | Duración | Objetivo |
|-------|----------|---------|
| **PHASE 1** | 8 semanas | Web al 100% funcional |
| **PHASE 2** | 6 semanas | App iOS/Android |
| **PHASE 3** | 4 semanas | Desktop install program |

### Stack Tecnológico

| Capa | Tecnología | Estado |
|------|-----------|--------|
| **Frontend Web** | Next.js 14, React 18, TypeScript | ✅ Existe |
| **Backend API** | FastAPI, Python 3.11+, Pydantic v2 | ✅ Base existe |
| **Database** | PostgreSQL, SQLAlchemy 2.0, Alembic | ✅ Existe |
| **Cache** | Redis | ⚠️ Por integrar |
| **Auth** | JWT, python-jose | ✅ Base existe |
| **Payments** | Stripe | ✅ Integración existe |
| **Mobile** | Capacitor, TypeScript | ❌ Por crear |
| **Install** | Bash scripts, platform installers | ❌ Por crear |

---

## 2. Arquitectura Objetivo

### Estructura de Directorios Final

```
OVAV/
│
├── web/                              # ⭐ NUEVO: Todo web unificado
│   ├── frontend/                     # Next.js 14
│   │   ├── src/
│   │   │   ├── app/                # App Router
│   │   │   │   ├── page.tsx        # Landing
│   │   │   │   ├── pricing/
│   │   │   │   ├── login/
│   │   │   │   ├── register/
│   │   │   │   ├── dashboard/       # User dashboard
│   │   │   │   │   ├── page.tsx
│   │   │   │   │   ├── profile/
│   │   │   │   │   ├── api-keys/
│   │   │   │   │   ├── subscription/
│   │   │   │   │   ├── billing/
│   │   │   │   │   └── settings/
│   │   │   │   ├── checkout/
│   │   │   │   ├── download/
│   │   │   │   ├── docs/           # Docs integration
│   │   │   │   └── admin/          # Admin panel
│   │   │   ├── components/         # Shared UI
│   │   │   │   ├── ui/             # shadcn/ui
│   │   │   │   ├── layout/
│   │   │   │   └── forms/
│   │   │   └── lib/
│   │   │       ├── api.ts         # API client
│   │   │       ├── auth.ts        # Auth helpers
│   │   │       └── utils.ts
│   │   ├── public/
│   │   ├── package.json
│   │   ├── next.config.js
│   │   └── tailwind.config.ts
│   │
│   └── backend/                    # FastAPI
│       ├── app/
│       │   ├── api/              # Routers
│       │   │   ├── v1/           # API v1
│       │   │   │   ├── auth.py
│       │   │   │   ├── users.py
│       │   │   │   ├── licenses.py
│       │   │   │   ├── api_keys.py
│       │   │   │   ├── billing.py
│       │   │   │   ├── checkout.py
│       │   │   │   ├── download.py
│       │   │   │   └── health.py
│       │   │   └── __init__.py
│       │   ├── models/           # SQLAlchemy
│       │   │   ├── user.py
│       │   │   ├── license.py
│       │   │   ├── api_key.py
│       │   │   └── invoice.py
│       │   ├── schemas/          # Pydantic
│       │   │   ├── auth.py
│       │   │   ├── user.py
│       │   │   ├── license.py
│       │   │   └── billing.py
│       │   ├── services/         # Business logic
│       │   │   ├── auth_service.py
│       │   │   ├── stripe_service.py
│       │   │   └── license_service.py
│       │   └── core/
│       │       ├── config.py
│       │       ├── database.py
│       │       ├── security.py
│       │       └── dependencies.py
│       ├── migrations/           # Alembic
│       ├── tests/
│       ├── pyproject.toml
│       └── Dockerfile
│
├── mobile/                          # ⭐ NUEVO: App iOS/Android
│   ├── capacitor/                   # Capacitor project
│   │   ├── src/
│   │   │   ├── screens/
│   │   │   │   ├── LoginScreen.tsx
│   │   │   │   ├── DashboardScreen.tsx
│   │   │   │   ├── TasksScreen.tsx
│   │   │   │   ├── ProjectsScreen.tsx
│   │   │   │   ├── SettingsScreen.tsx
│   │   │   │   └── ScanScreen.tsx
│   │   │   ├── components/
│   │   │   ├── services/
│   │   │   │   └── api.ts
│   │   │   ├── stores/
│   │   │   │   └── authStore.ts
│   │   │   └── App.tsx
│   │   ├── capacitor.config.ts
│   │   ├── ios/                  # iOS native
│   │   ├── android/               # Android native
│   │   └── package.json
│   │
│   └── src/                       # Shared TypeScript
│       └── shared/                # Libraries compartidas
│
├── install/                         # ⭐ NUEVO: Desktop Install
│   ├── scripts/
│   │   ├── install.sh            # Linux/macOS
│   │   ├── install.ps1           # Windows
│   │   └── verify.sh             # Post-install verification
│   ├── packages/
│   │   ├── deb/                  # Debian packages
│   │   ├── rpm/                  # RPM packages
│   │   ├── windows/             # MSI/NSIS
│   │   └── macos/               # DMG/pkg
│   └── src/                      # Install scripts source
│
├── go-runtime/                    # ✅ Existe (NO CAMBIA)
├── landing/                       # ⚠️ DEPRECATED → migrar a web/
├── docs-site/                     # ⚠️ DEPRECATED → migrar a web/frontend
└── [resto de OVAV]              # ✅ Existe
```

### API Gateway Specification

```yaml
# OpenAPI 3.1.0
info:
  title: OVAV API
  version: 1.0.0
  description: OVAV Platform API - AI Workstation Governor

servers:
  - url: https://api.ovav.dev/v1
    description: Production
  - url: https://staging.api.ovav.dev/v1
    description: Staging

paths:
  # ── Auth ──────────────────────────────────────
  /auth/register:
    post:
      tags: [Auth]
      summary: Register new user
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/RegisterRequest'
      responses:
        '201':
          description: User created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/AuthResponse'

  /auth/login:
    post:
      tags: [Auth]
      summary: Login with email magic link
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/LoginRequest'
      responses:
        '200':
          description: Magic link sent

  /auth/verify:
    post:
      tags: [Auth]
      summary: Verify magic link token
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/VerifyRequest'
      responses:
        '200':
          description: Token validated
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/AuthResponse'

  /auth/refresh:
    post:
      tags: [Auth]
      summary: Refresh access token
      security:
        - RefreshToken: []
      responses:
        '200':
          description: New tokens

  /auth/logout:
    post:
      tags: [Auth]
      summary: Logout
      security:
        - BearerAuth: []
      responses:
        '204':
          description: Logged out

  # ── Users ─────────────────────────────────────
  /users/me:
    get:
      tags: [Users]
      summary: Get current user
      security:
        - BearerAuth: []
      responses:
        '200':
          description: User profile
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/UserProfile'

    patch:
      tags: [Users]
      summary: Update current user
      security:
        - BearerAuth: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/UpdateUserRequest'
      responses:
        '200':
          description: User updated

  # ── Licenses ──────────────────────────────────
  /licenses:
    get:
      tags: [Licenses]
      summary: List user licenses
      security:
        - BearerAuth: []
      responses:
        '200':
          description: License list
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/License'

  /licenses/validate:
    post:
      tags: [Licenses]
      summary: Validate license from CLI
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/ValidateLicenseRequest'
      responses:
        '200':
          description: Validation result

  # ── API Keys ──────────────────────────────────
  /api-keys:
    get:
      tags: [API Keys]
      summary: List API keys
      security:
        - BearerAuth: []
      responses:
        '200':
          description: API key list

    post:
      tags: [API Keys]
      summary: Create API key
      security:
        - BearerAuth: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateApiKeyRequest'
      responses:
        '201':
          description: API key created

  /api-keys/{key_id}:
    delete:
      tags: [API Keys]
      summary: Delete API key
      security:
        - BearerAuth: []
      responses:
        '204':
          description: API key deleted

  # ── Billing ───────────────────────────────────
  /billing/invoices:
    get:
      tags: [Billing]
      summary: List invoices
      security:
        - BearerAuth: []
      responses:
        '200':
          description: Invoice list

  /billing/subscription:
    get:
      tags: [Billing]
      summary: Get subscription
      security:
        - BearerAuth: []
      responses:
        '200':
          description: Subscription details

    patch:
      tags: [Billing]
      summary: Update subscription
      security:
        - BearerAuth: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/UpdateSubscriptionRequest'
      responses:
        '200':
          description: Subscription updated

  # ── Download ─────────────────────────────────
  /download/cli:
    get:
      tags: [Download]
      summary: Get CLI download URLs
      security:
        - BearerAuth: []
      responses:
        '200':
          description: Download URLs for all platforms

  # ── Checkout ──────────────────────────────────
  /checkout/session:
    post:
      tags: [Checkout]
      summary: Create Stripe checkout session
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CheckoutRequest'
      responses:
        '200':
          description: Checkout session URL

components:
  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
    RefreshToken:
      type: http
      scheme: bearer

  schemas:
    RegisterRequest:
      type: object
      required: [email]
      properties:
        email:
          type: string
          format: email
        name:
          type: string

    LoginRequest:
      type: object
      required: [email]
      properties:
        email:
          type: string
          format: email

    VerifyRequest:
      type: object
      required: [token]
      properties:
        token:
          type: string

    AuthResponse:
      type: object
      properties:
        access_token:
          type: string
        refresh_token:
          type: string
        user:
          $ref: '#/components/schemas/UserProfile'

    UserProfile:
      type: object
      properties:
        id:
          type: string
          format: uuid
        email:
          type: string
        name:
          type: string
        avatar_url:
          type: string
        created_at:
          type: string
          format: date-time

    License:
      type: object
      properties:
        id:
          type: string
        license_key:
          type: string
        tier:
          type: string
          enum: [core, pro, enterprise]
        status:
          type: string
          enum: [active, trial, grace, expired, revoked]
        instances_max:
          type: integer
        instances_active:
          type: integer
        current_period_end:
          type: string
          format: date-time

    CreateApiKeyRequest:
      type: object
      properties:
        name:
          type: string
        expires_in_days:
          type: integer

    Invoice:
      type: object
      properties:
        id:
          type: string
        amount:
          type: number
        currency:
          type: string
        status:
          type: string
        created_at:
          type: string
          format: date-time

    CheckoutRequest:
      type: object
      required: [tier, email]
      properties:
        tier:
          type: string
          enum: [pro_monthly, pro_annual, enterprise]
        email:
          type: string
          format: email
        success_url:
          type: string
        cancel_url:
          type: string
```

---

## 3. PHASE 1: Web Full-Stack (8 semanas)

### 1.1 Landing Page

**Lead:** Sofia (Commercial Growth) + Elena (UX Design)

#### Tasks

- [ ] **1.1.1** Actualizar pricing tiers
  - Free tier description
  - Solo tier ($9/mes) - Mobile app access
  - Pro tier ($29/mes) - Full features
  - Team tier ($79/mes) - 5 seats
  - Enterprise tier ($199/mes) - SSO + SLA

- [ ] **1.1.2** Nueva hero section
  - "OVAV: AI Workstation Governor"
  - "Control profesional sobre AI agentes"
  - CTA: "Comenzar gratis" / "Ver demo"

- [ ] **1.1.3** Features comparison table
  - Core vs Solo vs Pro vs Team vs Enterprise
  - Mobile app column
  - AI Assistant column
  - Jira-like tasks column

- [ ] **1.1.4** Social proof section
  - Stats actualizados
  - Testimonios
  - Logos de empresas (placeholder)

- [ ] **1.1.5** Download section
  - "Descargar OVAV CLI"
  - Windows, macOS, Linux
  - Links temporales hasta implementar install program

- [ ] **1.1.6** Footer completo
  - Links a docs
  - Privacy policy
  - Terms of service
  - Contacto

#### Deliverables
- Landing page funcional en get.ovav.dev
- SEO optimization
- Mobile responsive
- Performance: 90+ Lighthouse

---

### 1.2 Backend API

**Lead:** Thavren (Platform Engineering)

#### Tasks

- [ ] **1.2.1** Estructurar API v1
  ```
  /app/api/v1/
  ├── __init__.py
  ├── auth.py
  ├── users.py
  ├── licenses.py
  ├── api_keys.py
  ├── billing.py
  ├── checkout.py
  └── download.py
  ```

- [ ] **1.2.2** Models completos
  ```python
  # user.py
  class User(Base):
      __tablename__ = "users"
      id: UUID
      email: str (unique)
      name: str | None
      avatar_url: str | None
      email_verified: bool
      created_at: datetime
      updated_at: datetime
      
      # Relations
      licenses: list[License]
      api_keys: list[ApiKey]
      invoices: list[Invoice]

  # license.py
  class License(Base):
      __tablename__ = "licenses"
      id: UUID
      user_id: UUID (FK)
      license_key: str (unique)
      tier: LicenseTier (core, solo, pro, team, enterprise)
      status: LicenseStatus
      machine_fingerprint: str | None
      instances_max: int
      instances_active: int
      current_period_start: datetime
      current_period_end: datetime | None
      trial_ends_at: datetime | None

  # api_key.py
  class ApiKey(Base):
      __tablename__ = "api_keys"
      id: UUID
      user_id: UUID (FK)
      name: str
      key_hash: str (hashed)
      key_prefix: str (last 8 chars visible)
      expires_at: datetime | None
      created_at: datetime
      last_used_at: datetime | None

  # invoice.py
  class Invoice(Base):
      __tablename__ = "invoices"
      id: UUID
      user_id: UUID (FK)
      license_id: UUID (FK)
      stripe_invoice_id: str | None
      amount: Decimal
      currency: str
      status: InvoiceStatus
      created_at: datetime
      paid_at: datetime | None
  ```

- [ ] **1.2.3** Auth endpoints completos
  ```python
  POST /auth/register       # Create user + send magic link
  POST /auth/login        # Request magic link
  POST /auth/verify       # Verify token → JWT
  POST /auth/refresh      # Refresh token
  POST /auth/logout       # Invalidate refresh token
  GET  /auth/me          # Get current user
  ```

- [ ] **1.2.4** User CRUD
  ```python
  GET    /users/me           # Get profile
  PATCH  /users/me          # Update profile
  DELETE /users/me           # Delete account
  GET    /users/me/licenses  # List licenses
  ```

- [ ] **1.2.5** License management
  ```python
  GET  /licenses                    # List user licenses
  POST /licenses/validate           # CLI validation
  GET  /licenses/{id}              # Get license detail
  POST /licenses/{id}/activate     # Activate instance
  ```

- [ ] **1.2.6** API Keys CRUD
  ```python
  GET    /api-keys                 # List keys
  POST   /api-keys                 # Create key
  GET    /api-keys/{id}           # Get key detail
  DELETE /api-keys/{id}           # Revoke key
  POST   /api-keys/{id}/rotate    # Rotate key
  ```

- [ ] **1.2.7** Billing endpoints
  ```python
  GET  /billing/invoices          # List invoices
  GET  /billing/invoices/{id}    # Invoice detail
  GET  /billing/subscription     # Current subscription
  PATCH /billing/subscription    # Update (cancel, change tier)
  POST /billing/portal           # Stripe customer portal
  ```

- [ ] **1.2.8** Checkout integration
  ```python
  POST /checkout/session          # Create Stripe session
  POST /checkout/webhook          # Stripe webhook handler
  ```

- [ ] **1.2.9** Download endpoints
  ```python
  GET /download/cli               # Get CLI download URLs
  GET /download/cli/{platform}    # Specific platform URL
  GET /download/version          # Latest version info
  ```

- [ ] **1.2.10** Database migrations
  ```bash
  # New tables
  - api_keys
  - invoices
  - refresh_tokens
  
  # Modified tables
  - users (add email_verified)
  - licenses (add tier: solo, team)
  ```

#### Deliverables
- API completa y documentada en api.ovav.dev
- Base de datos migrada
- Tests unitarios >80% coverage

---

### 1.3 Frontend Dashboard

**Lead:** Elena (UX Design) + Dante (Digital Product)

#### Tasks

- [ ] **1.3.1** Setup Next.js App Router
  ```tsx
  /app
  ├── (auth)/
  │   ├── login/page.tsx
  │   └── register/page.tsx
  ├── (dashboard)/
  │   ├── layout.tsx          # Dashboard layout with sidebar
  │   ├── page.tsx            # Dashboard home
  │   ├── profile/page.tsx
  │   ├── api-keys/page.tsx
  │   ├── subscription/page.tsx
  │   └── billing/page.tsx
  ├── checkout/page.tsx
  └── download/page.tsx
  ```

- [ ] **1.3.2** Auth pages
  - Login con magic link
  - Register con email
  - Forgot password
  - Email verification flow

- [ ] **1.3.3** Dashboard layout
  ```tsx
  // components/layout/DashboardLayout.tsx
  - Sidebar
    - OVAV logo
    - Dashboard
    - API Keys
    - Subscription
    - Billing
    - Settings
    - Logout
  - Header
    - User avatar
    - Notifications
  - Main content area
  ```

- [ ] **1.3.4** Profile page
  - Avatar upload
  - Name editing
  - Email (readonly)
  - Password change
  - Delete account (confirmation modal)

- [ ] **1.3.5** API Keys page
  - List of keys with copy button
  - Create new key modal
  - Expiry date picker
  - Delete key with confirmation
  - Last used timestamp

- [ ] **1.3.6** Subscription page
  - Current plan badge
  - Plan comparison table
  - Upgrade/Downgrade buttons
  - Cancel subscription
  - Reactivate subscription

- [ ] **1.3.7** Billing page
  - Invoice list with pagination
  - Invoice detail modal
  - Download PDF
  - Payment methods (Stripe portal link)

- [ ] **1.3.8** Download page
  - Platform detection
  - Download buttons per OS
  - Installation instructions
  - Version info

#### Deliverables
- Dashboard funcional en dashboard.ovav.dev
- All CRUD operations working
- Responsive design
- 90+ Lighthouse score

---

### 1.4 Auth Integration

**Lead:** Nora (API & Security Engineer)

#### Tasks

- [ ] **1.4.1** JWT implementation
  ```python
  # Access token: 15 min expiry
  # Refresh token: 7 days expiry
  
  # Token payload:
  {
    "sub": user_id,
    "email": email,
    "tier": license_tier,
    "type": "access" | "refresh",
    "exp": timestamp,
    "iat": timestamp
  }
  ```

- [ ] **1.4.2** Token storage
  - HTTP-only cookies for web
  - Secure storage for CLI
  - Refresh token rotation

- [ ] **1.4.3** OAuth providers (future)
  - Google OAuth
  - GitHub OAuth
  - Design for extensibility

#### Deliverables
- Auth flow completo
- Token refresh working
- Logout clearing all tokens

---

### 1.5 CLI ↔ Backend Bridge

**Lead:** Thavren (Platform Engineering)

#### Tasks

- [ ] **1.5.1** CLI login command
  ```bash
  ovav login              # Opens browser for magic link
  ovav login --token XYZ   # Direct token entry
  ovav logout             # Clear stored token
  ovav whoami             # Show current user
  ```

- [ ] **1.5.2** License sync
  ```bash
  ovav license            # Show current license
  ovav license --refresh  # Fetch from API
  ovav license --validate # Validate with backend
  ```

- [ ] **1.5.3** API key management
  ```bash
  ovav api-key list       # List keys
  ovav api-key create     # Create new key
  ovav api-key delete     # Revoke key
  ```

- [ ] **1.5.4** Version check
  ```bash
  ovav update            # Check for updates
  ovav update --install   # Download and install
  ```

#### Deliverables
- CLI fully authenticated
- License status from API
- API key management in CLI

---

## 4. PHASE 2: Mobile iOS/Android (6 semanas)

### 2.1 Mobile Architecture

**Lead:** Dante (Digital Product)

```
┌─────────────────────────────────────────────────────────────────┐
│                    MOBILE ARCHITECTURE                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐       │
│  │    iOS     │     │  Android   │     │  Web View  │       │
│  │  (Swift)  │     │  (Kotlin)  │     │  (Capacitor)│       │
│  └──────┬──────┘     └──────┬──────┘     └──────┬──────┘       │
│         │                   │                   │                │
│         └───────────────────┼───────────────────┘                │
│                             │                                │
│                             ▼                                │
│                   ┌─────────────────┐                      │
│                   │   Capacitor     │                      │
│                   │   Bridge Layer  │                      │
│                   └────────┬────────┘                      │
│                            │                               │
│                            ▼                               │
│                   ┌─────────────────┐                      │
│                   │  React/TS App  │                      │
│                   │   (Shared)     │                      │
│                   └────────┬────────┘                      │
│                            │                               │
│         ┌─────────────────┼─────────────────┐              │
│         │                 │                 │              │
│         ▼                 ▼                 ▼              │
│  ┌───────────┐    ┌───────────┐    ┌───────────┐           │
│  │   Auth    │    │ Dashboard │    │  Tasks    │           │
│  │  Screen   │    │  Screen   │    │  Screen   │           │
│  └───────────┘    └───────────┘    └───────────┘           │
│                                                                  │
│         ┌─────────────────┼─────────────────┐              │
│         │                 │                 │              │
│         ▼                 ▼                 ▼              │
│  ┌───────────┐    ┌───────────┐    ┌───────────┐           │
│  │ Projects  │    │  Vault    │    │ Settings  │           │
│  │  Screen   │    │  Screen   │    │  Screen   │           │
│  └───────────┘    └───────────┘    └───────────┘           │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 Capacitor Setup

#### Tasks

- [ ] **2.2.1** Initialize Capacitor project
  ```bash
  cd mobile
  npm init capacitor-app
  npm install @capacitor/core @capacitor/cli
  npx cap init OVAV com.ovav.app
  npm install @capacitor/android @capacitor/ios
  npx cap add android
  npx cap add ios
  ```

- [ ] **2.2.2** Configure Capacitor
  ```typescript
  // capacitor.config.ts
  import { CapacitorConfig } from '@capacitor/cli';
  
  const config: CapacitorConfig = {
    appId: 'com.ovav.app',
    appName: 'OVAV',
    webDir: '../../web/frontend/out',
    server: {
      androidScheme: 'https'
    },
    plugins: {
      SplashScreen: {
        launchShowDuration: 2000,
        backgroundColor: '#030712',
        showSpinner: false
      },
      StatusBar: {
        style: 'DARK',
        backgroundColor: '#030712'
      }
    }
  };
  ```

- [ ] **2.2.3** Install plugins
  ```bash
  npm install @capacitor/geolocation
  npm install @capacitor/push-notifications
  npm install @capacitor/haptics
  npm install @capacitor/browser
  npm install @capacitor/share
  ```

### 2.3 Mobile Screens

#### Tasks

- [ ] **2.3.1** Auth screens
  - Login with magic link
  - Register flow
  - Password reset
  - Biometric auth (future)

- [ ] **2.3.2** Dashboard screen
  - Quick stats cards
  - Recent activity
  - Active license badge
  - Quick actions

- [ ] **2.3.3** Tasks screen (Jira-like)
  - Task list with filters
  - Task detail view
  - Create task
  - Edit task
  - Comments
  - Attachments (future)

- [ ] **2.3.4** Projects screen
  - Project list
  - Project detail
  - Project members
  - Project settings

- [ ] **2.3.5** Vault screen (read-only)
  - Asset list
  - Asset detail
  - Encryption status
  - Audit log

- [ ] **2.3.6** Settings screen
  - Profile settings
  - Notification preferences
  - Security settings
  - About / Version

#### Mobile API Client
```typescript
// mobile/src/services/api.ts
const API_BASE = 'https://api.ovav.dev/v1';

interface ApiResponse<T> {
  data: T;
  error?: string;
}

class MobileApi {
  private token: string | null = null;
  
  async setToken(token: string) {
    this.token = token;
    await SecureStorage.set({ key: 'auth_token', value: token });
  }
  
  async get<T>(path: string): Promise<T> {
    const res = await fetch(`${API_BASE}${path}`, {
      headers: this.authHeaders()
    });
    return res.json();
  }
  
  async post<T>(path: string, body: object): Promise<T> {
    const res = await fetch(`${API_BASE}${path}`, {
      method: 'POST',
      headers: this.authHeaders(),
      body: JSON.stringify(body)
    });
    return res.json();
  }
  
  private authHeaders() {
    return {
      'Authorization': `Bearer ${this.token}`,
      'Content-Type': 'application/json'
    };
  }
}

export const api = new MobileApi();
```

#### Deliverables
- App en TestFlight (iOS)
- App in Play Store Beta (Android)
- Auth flow working
- Dashboard functional
- Tasks CRUD functional

---

## 5. PHASE 3: Desktop Install Program (4 semanas)

### 3.1 Install Script

**Lead:** Thavren (Platform Engineering)

#### Tasks

- [ ] **3.1.1** Bash install script
  ```bash
  #!/bin/bash
  set -e
  
  VERSION="3.0.0"
  INSTALL_DIR="${HOME}/.local/ovav"
  BIN_DIR="${HOME}/.local/bin"
  
  # Detect OS
  detect_os() {
    case "$(uname -s)" in
      Linux*)   echo "linux" ;;
      Darwin*)  echo "macos" ;;
      MINGW*)   echo "windows" ;;
      *)        echo "unknown" ;;
    esac
  }
  
  # Download binary
  download_binary() {
    local os=$(detect_os)
    local url="https://api.ovav.dev/v1/download/cli/${os}"
    local binary="${INSTALL_DIR}/ovav"
    
    mkdir -p "${INSTALL_DIR}"
    curl -fsSL "${url}" -o "${binary}"
    chmod +x "${binary}"
  }
  
  # Create symlink
  create_symlink() {
    ln -sf "${INSTALL_DIR}/ovav" "${BIN_DIR}/ovav"
  }
  
  # Verify installation
  verify_install() {
    ovav --version
    ovav doctor
  }
  
  main() {
    download_binary
    create_symlink
    verify_install
    echo "OVAV installed successfully!"
  }
  
  main "$@"
  ```

- [ ] **3.1.2** PowerShell install script
  ```powershell
  # install-ovav.ps1
  
  $Version = "3.0.0"
  $InstallDir = "$env:LOCALAPPDATA\OVAV"
  $BinDir = "$env:LOCALAPPDATA\Programs\OVAV"
  
  function Install-OVAV {
      $os = if ($IsWindows) { "windows" } else { "unknown" }
      $url = "https://api.ovav.dev/v1/download/cli/$os"
      
      New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
      Invoke-WebRequest -Uri $url -OutFile "$InstallDir\ovav.exe"
      
      # Add to PATH
      [Environment]::SetEnvironmentVariable(
          "PATH",
          "$env:PATH;$InstallDir",
          "User"
      )
  }
  
  Install-OVAV
  ```

- [ ] **3.1.3** Verification script
  ```bash
  #!/bin/bash
  ovav --version
  ovav doctor --quick
  ovav login --check
  ```

### 3.2 Platform Installers

#### Tasks

- [ ] **3.2.1** macOS DMG
  - Create DMG with app bundle
  - Include uninstaller
  - Code signed (if certificate available)

- [ ] **3.2.2** Windows MSI/NSIS
  - MSI with WiX
  - Start menu shortcuts
  - Add/Remove programs entry

- [ ] **3.2.3** Linux packages
  - .deb for Debian/Ubuntu
  - .rpm for Fedora/RHEL
  - .AppImage portable

#### Deliverables
- Install scripts in install/scripts/
- Platform packages in install/packages/
- Documentation in docs/install/

---

## 6. Gobernanza y Validación

### Quality Gates

| Gate | Criteria | Owner |
|------|----------|--------|
| **Code Review** | 2 approvals | Pablo (Code Review) |
| **Tests** | >80% coverage | Clara (QA) |
| **Security** | F4 validators pass | Diana (Security) |
| **Performance** | Lighthouse >90 | Óscar (Performance) |
| **Accessibility** | WCAG 2.1 AA | Elena (UX) |

### Validation Commands

```bash
# Backend
cd web/backend
pytest tests/ -v --cov=. --cov-report=html
fastapi dev app/main.py

# Frontend
cd web/frontend
npm run build
npm run lint
npm run test

# Mobile
cd mobile
npx cap sync
npm run build
npx cap copy

# Install
bash install/scripts/verify.sh
```

---

## 7. Definition of Done

### PHASE 1: Web Full-Stack

| Criteria | Evidence |
|----------|----------|
| Landing page live | get.ovav.dev responding |
| Checkout working | Stripe payment flow complete |
| Dashboard functional | All CRUD operations working |
| Auth integrated | CLI ↔ Web auth working |
| API documented | OpenAPI spec valid |
| Tests passing | >80% coverage |

### PHASE 2: Mobile

| Criteria | Evidence |
|----------|----------|
| iOS app in TestFlight | Build successful |
| Android app in Play Store Beta | APK uploaded |
| Auth working | Magic link login functional |
| Core features | Tasks, Projects, Vault screens |

### PHASE 3: Desktop Install

| Criteria | Evidence |
|----------|----------|
| Bash script working | `curl | sh` installs OVAV |
| Windows script working | PowerShell install works |
| macOS package | DMG installs app |
| Linux packages | .deb/.rpm install correctly |

---

## 📅 Timeline

```
WEEK  1   2   3   4   5   6   7   8   9   10  11  12  13  14  15  16  17  18
       ├───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┤
PH1    [  Landing + Backend + Dashboard + Auth + CLI Bridge                     ]
PH2                        [  Mobile Setup + Screens                                    ]
PH3                                                        [  Install Scripts         ]
```

---

## 👥 Asignación de Leads

| Task | Lead | Squad |
|------|------|-------|
| Landing page | Sofia, Elena | Design |
| Backend API | Thavren | Platform |
| Frontend Dashboard | Elena, Dante | Digital Product |
| Auth integration | Nora | API & Security |
| CLI Bridge | Thavren | Platform |
| Mobile iOS/Android | Dante | Digital Product |
| Desktop Install | Thavren | Platform |

---

## 📁 Archivos de Referencia

| Archivo | Descripción |
|---------|-------------|
| `.ovav/plan/caps.yaml` | Plan canonical |
| `plans/OVAV-INFRASTRUCTURE-CONSOLIDATION-2026-08-06.md` | Estructura repo |
| `landing/backend/app/api/` | API actual |
| `landing/frontend/src/app/` | Frontend actual |

---

*Thavren — Platform Engineering Lead — 2026-08-08*
