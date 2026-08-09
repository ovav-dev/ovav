# DevOps & Infrastructure — Mandatory Questions

> Loaded by `ovav-brainstorm` skill when deploy/infra/Docker/K8s request detected.
> Ask 3-5 questions ONE AT A TIME. Wait for CEO response before next.

## DevOps Questions

### DOQ1: Deployment Platform
"¿Dónde se deploya?"
- Vercel / Railway / Fly.io (泛化平台, zero config)
- AWS (EC2, ECS/EKS, máximo control)
- GCP (Cloud Run, GKE)
- Self-hosted ( VPS, bare metal)
- Multi-cloud (2+ providers)

Why: Vercel = speed. AWS = enterprise control. Self-hosted = data sovereignty.

### DOQ2: Container Strategy
"¿Docker para qué?"
- No containers (single process, deployment simple)
- Docker Compose (multi-service en una máquina)
- Kubernetes (multi-node, auto-scale)
- Container sin K8s (Docker en VPS con systemd)

Why: Compose = simplicity. K8s = resilience pero complejidad operacional alta.

### DOQ3: CI/CD Pipeline
"¿Cómo se automatiza el deploy?"
- GitHub Actions (estándar, integrado con GitHub)
- GitLab CI (si repo está en GitLab)
- ArgoCD (GitOps, Kubernetes-native)
- Deploy manual (CI/CD overkill para proyecto simple)

Why: GitOps = audit trail + rollback rápido. GitHub Actions = simplicidad para la mayoría.

### DOQ4: Environment Strategy
"¿Cuántos entornos?"
- Dev (local) + Production (único)
- Dev + Staging + Production
- Feature branches → Preview URLs automáticas

Why: Staging reduce errores en producción pero añade costo y latency de CI.

### DOQ5: Secrets Management
"¿Cómo se manejan secrets?"
- Environment variables (simple, pero rotatable con restart)
- Vault (HashiCorp Vault, rotation automática, audit log)
- Cloud-native (AWS Secrets Manager, GCP Secret Manager)
- Sin secrets centralizado (en `.env` en cada server)

Why: Vault = rotation sin downtime + audit. Cloud-native = buena integración con el cloud del provider.

## Deliverable
After all questions answered → Uriel adds infrastructure and deployment sections to PLAN.md.
