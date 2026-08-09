# OVAV Rego Policies — F1.1 OPA/Rego Engine
# Policy-as-code definitions for OVAV permission evaluation.
# Schema: ovav.rego_policy.v1

# ── Core deny rules ──────────────────────────────────────────────────────────

package ovav.security

# Default: deny if no explicit allow
default allow = false

# ── Bash command rules ───────────────────────────────────────────────────────

deny_bash[rules] {
    input.action == "bash"
    input.command == cmd
    rules := [
        "sudo",
        "pip install",
        "npm install",
        "apt install",
        "gh auth token",
        "gh auth login",
        "gh release",
    ]
    contains(input.command, rules[_])
}

# ── File path rules ──────────────────────────────────────────────────────────

deny_path_traversal {
    input.action == "file_write"
    contains(input.path, "..")
}

deny_system_path_write {
    input.action == "file_write"
    not startswith(input.path, input.workspace_root)
    not startswith(input.path, "/tmp/opencode/")
}

# ── Git rules ────────────────────────────────────────────────────────────────

deny_force_push {
    input.action == "git_push"
    contains(input.flags, "--force")
}

deny_force_delete_branch {
    input.action == "git_branch_delete"
    input.flag == "-D"
}

deny_protected_branch_push {
    input.action == "git_push"
    input.branch == protected_branch
}

protected_branch := branch {
    branch := ["main", "master", "develop", "production", "prod", "staging"][_]
    input.branch == branch
}

# ── Plugin and extension rules ───────────────────────────────────────────────

deny_plugin_install {
    input.action == "install_plugin"
    input.operator != "thavren"
}

deny_extension_install {
    input.action == "install_extension"
    input.operator != "thavren"
}

deny_external_mcp {
    input.action == "register_mcp_server"
    input.source != "ovav_internal"
}

# ── Network rules ────────────────────────────────────────────────────────────

deny_external_network {
    input.action == "external_request"
    not network_allowed(input.url)
}

network_allowed(url) {
    allowed_domains := [
        "github.com",
        "api.github.com",
        "*.githubusercontent.com",
        "pypi.org",
        "files.pythonhosted.org",
        "docs.python.org",
        "*.readthedocs.io",
        "arxiv.org",
        "scholar.google.com",
        "ovav.dev",
        "*.ovav.dev",
    ]
    domain := domain_from_url(url)
    domain_matches(domain, allowed_domains[_])
}

domain_from_url(url) := domain {
    # Extract hostname from URL
    parts := split(url, "://")
    host_part := parts[count(parts) - 1]
    host_parts := split(host_part, "/")
    domain := host_parts[0]
}

domain_matches(domain, pattern) {
    domain == pattern
}

domain_matches(domain, pattern) {
    startswith(pattern, "*.")
    suffix := trim_prefix(pattern, "*")
    endswith(domain, suffix)
}

# ── Rate limiting rules ──────────────────────────────────────────────────────

deny_rate_limit {
    input.action == "external_request"
    input.request_count > input.rate_limit
}

# ── Secret protection rules ──────────────────────────────────────────────────

deny_secret_in_output {
    input.action == "output"
    secret_patterns := [
        "ghp_",
        "github_pat_",
        "gho_",
        "AKIA",
        "sk-",
        "sk-ant-",
        "-----BEGIN RSA PRIVATE KEY-----",
        "-----BEGIN EC PRIVATE KEY-----",
    ]
    contains(input.content, secret_patterns[_])
}

# ── Operator profile rules ───────────────────────────────────────────────────

allow_operator_action {
    input.operator == "thavren"
    input.scope == "repo_local"
}

allow_operator_action {
    input.operator == "eidren"
    input.action == "research"
    input.scope == "repo_local"
}

# ── Bootstrap integrity requirement ──────────────────────────────────────────

deny_without_bootstrap {
    not input.bootstrap_valid
}

# ── Composite allow decision ─────────────────────────────────────────────────

allow {
    # Operator has explicit permission for this scope
    input.operator == "thavren"
    input.scope == "repo_local"
    not denied
}

allow {
    # Research operations for Eidren
    input.operator == "eidren"
    input.action == "research"
    not denied
}

allow {
    # Explicit grant in permission authority
    input.explicit_grant == true
    not denied
}

# ── Deny aggregation ─────────────────────────────────────────────────────────

denied {
    deny_bash[_]
}

denied {
    deny_path_traversal
}

denied {
    deny_system_path_write
}

denied {
    deny_force_push
}

denied {
    deny_force_delete_branch
}

denied {
    deny_protected_branch_push
}

denied {
    deny_plugin_install
}

denied {
    deny_extension_install
}

denied {
    deny_external_mcp
}

denied {
    deny_external_network
}

denied {
    deny_rate_limit
}

denied {
    deny_secret_in_output
}

denied {
    deny_without_bootstrap
}
