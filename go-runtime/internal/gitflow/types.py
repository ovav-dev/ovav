# tools/git/branch/types.py
# ============================================================
# OVAV Branch Types — Single Source of Truth (SSOT)
# ============================================================
# This module is the canonical source for branch taxonomy,
# lifetime policies, tier assignments, and helper functions.
# All other modules (security, workspace_safety, etc.) import from here.
# ============================================================

# Protected branches (never force-push, never delete)
PROTECTED_BRANCHES = {"main", "master", "develop", "staging", "prod", "production"}

# Protected prefixes (patterns that create protected branches)
PROTECTED_PREFIXES = {"main", "master", "develop", "staging", "prod", "production", "release/", "hotfix/"}

# Ultra branches — highest criticality (main, master only)
ULTRA_BRANCHES = {"main", "master"}

# High branches — high criticality (release/hotfix branches)
HIGH_BRANCHES = {"release/", "hotfix/"}

# Production branches — directly affect production
PRODUCTION_BRANCHES = {"main", "master", "prod", "production"}

# Primary branches — main development targets
PRIMARY_BRANCHES = {"main", "master", "develop", "staging"}

# Work branch prefixes
WORK_BRANCH_PREFIXES = {"feature/", "fix/", "hotfix/", "refactor/", "docs/", "test/", "ci/"}

# All branch keys
ALL_BRANCH_KEYS = {"ultra", "high", "production", "primary", "work", "experimental"}

# Conventional commits branch map
CC_BRANCH_MAP = {
    "feature": "feature/",
    "fix": "fix/",
    "hotfix": "hotfix/",
    "refactor": "refactor/",
    "docs": "docs/",
    "test": "test/",
    "ci": "ci/",
    "build": "build/",
    "chore": "chore/",
    "perf": "perf/",
    "security": "security/",
}

# OVAV-specific branch map
OVAV_BRANCH_MAP = {
    "feature": "feature/",
    "fix": "fix/",
    "hotfix": "hotfix/",
    "feature/feat": "feature/",
    "fix/fix": "fix/",
}

# Task prefixes
TASK_PREFIXES = {"T", "TASK", "BUG", "FIX", "HOTFIX"}

# Valid task patterns
VALID_TASK_PATTERNS = [
    r"^(T|TASK|BUG|FIX|HOTFIX)[-_]\d+(\.\d+)*$",
]

# Lifetime policies (in days)
LIFETIME_POLICIES = {
    "ultra": 0,        # Never expires
    "high": 30,        # 30 days
    "production": 90,  # 90 days
    "primary": 180,    # 6 months
    "work": 14,        # 2 weeks
    "experimental": 7,  # 1 week
}

# Prefix to tier mapping
PREFIX_TIER_MAP = {
    "main": "ultra",
    "master": "ultra",
    "release/": "high",
    "hotfix/": "high",
    "prod": "production",
    "production": "production",
    "develop": "primary",
    "staging": "primary",
    "feature/": "work",
    "fix/": "work",
    "refactor/": "work",
    "docs/": "work",
    "test/": "work",
    "ci/": "work",
}


def is_work_branch(branch: str) -> bool:
    """Check if branch is a work branch (feature/fix/etc.)"""
    return any(branch.startswith(p) for p in WORK_BRANCH_PREFIXES)


def is_protected_branch(branch: str) -> bool:
    """Check if branch is a protected branch"""
    if branch in PROTECTED_BRANCHES:
        return True
    return any(branch.startswith(p) for p in PROTECTED_PREFIXES)


def is_primary_branch(branch: str) -> bool:
    """Check if branch is a primary branch"""
    return branch in PRIMARY_BRANCHES


def get_branch_tier(branch: str) -> str:
    """Get the tier of a branch"""
    if branch in ULTRA_BRANCHES:
        return "ultra"
    if branch in PRODUCTION_BRANCHES:
        return "production"
    if any(branch.startswith(p) for p in HIGH_BRANCHES):
        return "high"
    if branch in PRIMARY_BRANCHES:
        return "primary"
    if is_work_branch(branch):
        return "work"
    return "experimental"


def get_branch_prefix(branch: str) -> str:
    """Get the prefix of a branch"""
    for prefix in WORK_BRANCH_PREFIXES:
        if branch.startswith(prefix):
            return prefix
    return ""


def get_branch_type_key(branch: str) -> str:
    """Get the branch type key for a branch"""
    return get_branch_tier(branch)


def get_lifetime_policy(branch: str) -> int:
    """Get the lifetime policy in days for a branch"""
    tier = get_branch_tier(branch)
    return LIFETIME_POLICIES.get(tier, 7)


def validate_branch_prefix(branch: str) -> bool:
    """Validate that a branch has a correct prefix"""
    if is_protected_branch(branch):
        return True
    if is_work_branch(branch):
        return True
    return False