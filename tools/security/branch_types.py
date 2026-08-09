# tools/security/branch_types.py
# ============================================================
# OVAV Branch Types — Security Surface Bridge
# ============================================================
# All branch-type logic lives in tools/git/branch/types.py (SSOT).
# This module is the official import surface for the security
# subsystem (workspace_safety_gate, rego_engine, etc.) and must
# be kept in sync by re-exporting only.
#
# DO NOT ADD LOGIC HERE.  All branch taxonomy, lifetime policies,
# tier assignments, and helper functions live in types.py.
# ============================================================

from tools.git.branch.types import (
    PROTECTED_BRANCHES,
    PROTECTED_PREFIXES,
    is_work_branch,
    is_protected_branch,
    is_primary_branch,
    get_branch_tier,
    get_branch_prefix,
    get_branch_type_key,
    get_lifetime_policy,
    validate_branch_prefix,
    # convenience sets
    ULTRA_BRANCHES,
    HIGH_BRANCHES,
    PRODUCTION_BRANCHES,
    PRIMARY_BRANCHES,
    # prefixes / maps
    WORK_BRANCH_PREFIXES,
    ALL_BRANCH_KEYS,
    CC_BRANCH_MAP,
    OVAV_BRANCH_MAP,
    TASK_PREFIXES,
    VALID_TASK_PATTERNS,
    # lifetime
    LIFETIME_POLICIES,
    PREFIX_TIER_MAP,
)

__all__ = [
    "PROTECTED_BRANCHES",
    "PROTECTED_PREFIXES",
    "is_work_branch",
    "is_protected_branch",
    "is_primary_branch",
    "get_branch_tier",
    "get_branch_prefix",
    "get_branch_type_key",
    "get_lifetime_policy",
    "validate_branch_prefix",
    "ULTRA_BRANCHES",
    "HIGH_BRANCHES",
    "PRODUCTION_BRANCHES",
    "PRIMARY_BRANCHES",
    "WORK_BRANCH_PREFIXES",
    "ALL_BRANCH_KEYS",
    "CC_BRANCH_MAP",
    "OVAV_BRANCH_MAP",
    "TASK_PREFIXES",
    "VALID_TASK_PATTERNS",
    "LIFETIME_POLICIES",
    "PREFIX_TIER_MAP",
]
