# OVAV MCP Security Best Practices

## Overview

This document outlines security considerations for using OVAV MCP servers in production environments. Follow these guidelines to protect your data and systems.

---

## 🔐 Authentication & Authorization

### API Keys & Tokens

```bash
# ✅ Use environment variables
export FIGMA_TOKEN="figd_xxxxxxxxxxxxx"
export DATABASE_URL="postgresql://user:pass@host/db"

# ❌ Never hardcode credentials
const token = "figd_1234567890"  # DANGEROUS
```

### Token Scope

| Service | Recommended Scope | Purpose |
|---------|-------------------|---------|
| Figma | Read-only | Extract design tokens |
| PostgreSQL | SELECT only (unless needed) | Query data |
| API Gateway | Minimal required | Specific endpoints only |

### Rotation Policy

- **Figma tokens:** Rotate every 90 days
- **Database credentials:** Rotate every 30 days
- **API keys:** Rotate every 60 days

---

## 🛡️ Tool Security

### 1. Tool Name Namespacing

All OVAV MCP tools are prefixed with `ovav_` to prevent conflicts:

```typescript
// ✅ OVAV tools (namespaced)
ovav_figma_get_layout
ovav_postgres_query
ovav_api_call

// ❌ Generic names (conflict risk)
get_layout
query_db
call_api
```

### 2. Input Validation

```typescript
// ✅ Validate all inputs
const validateInput = (input: any) => {
  if (!input.fileKey?.match(/^[a-zA-Z0-9]+$/)) {
    throw new Error("Invalid file key");
  }
  if (!input.nodeId?.match(/^\d+:\d+$/)) {
    throw new Error("Invalid node ID");
  }
};

// ❌ Trust user input
const getNode = (input) => figma.getNode(input.nodeId);
```

### 3. Output Sanitization

```typescript
// ✅ Sanitize outputs before returning
const sanitizeOutput = (data: any) => {
  // Remove sensitive fields
  delete data.password;
  delete data.token;
  
  // Truncate large outputs
  if (JSON.stringify(data).length > 10000) {
    return { truncated: true, preview: data.slice(0, 100) };
  }
  
  return data;
};
```

---

## 🔒 Transport Security

### Local MCP (stdio)

```json
{
  "mcp": {
    "server": {
      "command": "npx",
      "args": ["-y", "ovav-mcp-figma"],
      "env": {
        "FIGMA_TOKEN": "${FIGMA_TOKEN}"
      }
    }
  }
}
```

**Security:** Process runs locally, no network exposure.

### Remote MCP (HTTP)

```json
{
  "mcp": {
    "server": {
      "url": "https://mcp.ovav.dev/figma",
      "headers": {
        "Authorization": "Bearer ${API_TOKEN}"
      }
    }
  }
}
```

**Security requirements:**
- ✅ HTTPS only (no HTTP)
- ✅ Authentication required
- ✅ Rate limiting enabled
- ✅ Audit logging active

---

## 📋 OWASP Compliance

Per **OWASP MCP Top 10 2026**:

| Risk | Mitigation | Status |
|------|------------|--------|
| **Tool Poisoning** | Namespace isolation + description sanitization | ✅ Implemented |
| **Prompt Injection** | Input validation on all parameters | ✅ Implemented |
| **Memory Poisoning** | Scope isolation for memory MCP | ✅ Implemented |
| **Tool Interference** | Unique tool names across servers | ✅ Implemented |
| **Excessive Agency** | Role-based tool access | ✅ Implemented |
| **Supply Chain** | Verified package signatures | ✅ Implemented |

---

## 🚨 Threat Model

### Attack Vectors

1. **Malicious Tool Descriptions**
   - Attacker modifies tool description to inject prompts
   - **Defense:** Sanitize all descriptions, treat as untrusted

2. **Token Theft**
   - Attacker extracts tokens from environment
   - **Defense:** Use secret managers, rotate tokens regularly

3. **SQL Injection**
   - Attacker injects SQL via user input
   - **Defense:** Always use parameterized queries

4. **SSRF via API Gateway**
   - Attacker uses API gateway to access internal services
   - **Defense:** Whitelist allowed endpoints, block internal IPs

### Defensive Layers

```
┌─────────────────────────────────────┐
│  Layer 1: Input Validation          │
│  - Validate all tool parameters     │
│  - Reject malformed inputs          │
├─────────────────────────────────────┤
│  Layer 2: Authentication            │
│  - Verify tokens before execution   │
│  - Check token scope/permissions    │
├─────────────────────────────────────┤
│  Layer 3: Authorization             │
│  - Role-based tool access           │
│  - Restrict dangerous operations    │
├─────────────────────────────────────┤
│  Layer 4: Audit Logging             │
│  - Log all tool invocations         │
│  - Monitor for anomalies            │
└─────────────────────────────────────┘
```

---

## 📊 Monitoring & Alerting

### Metrics to Track

| Metric | Alert Threshold | Action |
|--------|-----------------|--------|
| Failed auth attempts | >5 in 5 min | Block IP |
| Query execution time | >5s | Investigate |
| Error rate | >1% | Check logs |
| Unusual data access | Pattern change | Audit review |

### Audit Log Format

```json
{
  "timestamp": "2026-07-23T10:30:00Z",
  "tool": "ovav_postgres_query",
  "user": "agent-123",
  "parameters": {
    "sql": "SELECT * FROM users WHERE id = $1",
    "params": [123]
  },
  "result": "success",
  "duration_ms": 45,
  "ip": "127.0.0.1"
}
```

---

## ✅ Security Checklist

### Before Deployment

- [ ] All tokens stored in environment variables
- [ ] No hardcoded credentials in code
- [ ] Database uses read-only user (unless write needed)
- [ ] API gateway whitelists only required endpoints
- [ ] Audit logging enabled
- [ ] Rate limiting configured
- [ ] HTTPS enforced for remote servers

### Regular Maintenance

- [ ] Rotate tokens every 30-90 days
- [ ] Review audit logs weekly
- [ ] Update MCP servers monthly
- [ ] Security scan before releases
- [ ] Penetration testing quarterly

---

## 🆘 Incident Response

### If Token Compromised

1. **Immediately revoke** the compromised token
2. **Generate new token** with same scope
3. **Update all configurations** with new token
4. **Review audit logs** for unauthorized access
5. **Notify team** of potential breach

### If Data Breach Suspected

1. **Isolate affected systems**
2. **Preserve audit logs**
3. **Notify security team**
4. **Begin investigation**
5. **Document findings**

---

## 📚 References

- [OWASP MCP Top 10 2026](https://genai.owasp.org/resource/owasp-top-10-for-agentic-applications-for-2026/)
- [MCP Security Best Practices](https://modelcontextprotocol.io/docs/tutorials/security/security_best_practices.md)
- [MCP Security Interest Group](https://modelcontextprotocol.io/community/interest-groups/security.md)

---

**Last Updated:** 2026-07-23
**Version:** 1.0.0
