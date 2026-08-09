"""License validation and management — bridge between OVAV CLI and web platform."""
import hashlib
import uuid
from datetime import UTC, datetime

from fastapi import APIRouter, HTTPException, Depends, Request
from pydantic import BaseModel
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.database import get_db
from app.models.user import License, LicenseStatus, LicenseTier, User
from app.api.auth import get_current_user

router = APIRouter()


class LicenseValidateRequest(BaseModel):
    license_key: str
    machine_fingerprint: str | None = None
    ovav_version: str = "1.0.0"


class LicenseValidateResponse(BaseModel):
    valid: bool
    tier: str
    status: str
    features: list[str]
    message: str


FEATURES_BY_TIER = {
    LicenseTier.CORE: [
        "validators_basic", "session_capsule", "boundary_law",
        "secrets_hygiene", "drift_detection", "output_guard",
    ],
    LicenseTier.PRO: [
        "validators_basic", "session_capsule", "boundary_law",
        "secrets_hygiene", "drift_detection", "output_guard",
        "validators_advanced", "context_firewall", "connector_bus",
        "memory_governor", "cli_visual", "sbom",
    ],
    LicenseTier.ENTERPRISE: [
        "validators_basic", "session_capsule", "boundary_law",
        "secrets_hygiene", "drift_detection", "output_guard",
        "validators_advanced", "context_firewall", "connector_bus",
        "memory_governor", "cli_visual", "sbom",
        "sso", "audit_log", "team_management", "custom_rules",
        "priority_support", "sla",
    ],
}


@router.post("/validate", response_model=LicenseValidateResponse)
async def validate_license(
    body: LicenseValidateRequest,
    request: Request,
    db: AsyncSession = Depends(get_db),
):
    """Validate a license key. Called by OVAV CLI on startup."""
    # Hash fingerprint for privacy
    fingerprint_hash = None
    if body.machine_fingerprint:
        fingerprint_hash = hashlib.sha256(
            body.machine_fingerprint.encode()
        ).hexdigest()

    result = await db.execute(
        select(License).where(License.license_key == body.license_key)
    )
    lic = result.scalar_one_or_none()

    if not lic:
        raise HTTPException(status_code=404, detail="License not found")

    if lic.status == LicenseStatus.REVOKED:
        return LicenseValidateResponse(
            valid=False, tier="none", status="revoked",
            features=[], message="License has been revoked."
        )

    if lic.status == LicenseStatus.EXPIRED:
        return LicenseValidateResponse(
            valid=False, tier=lic.tier.value, status="expired",
            features=[], message="License has expired. Renew to continue."
        )

    if lic.status == LicenseStatus.GRACE:
        if lic.current_period_end and lic.current_period_end < datetime.now(UTC):
            lic.status = LicenseStatus.EXPIRED
            await db.commit()
            raise HTTPException(status_code=402, detail="Grace period expired")

    # Update machine fingerprint on first activation
    if fingerprint_hash and lic.machine_fingerprint is None:
        lic.machine_fingerprint = fingerprint_hash
        lic.instances_active = 1
        await db.commit()
    elif fingerprint_hash and lic.machine_fingerprint != fingerprint_hash:
        if lic.instances_active >= lic.instances_max:
            return LicenseValidateResponse(
                valid=False, tier=lic.tier.value, status=lic.status.value,
                features=[], message=f"Instance limit reached ({lic.instances_active}/{lic.instances_max})."
            )
        lic.instances_active += 1
        await db.commit()

    features = FEATURES_BY_TIER.get(lic.tier, FEATURES_BY_TIER[LicenseTier.CORE])

    return LicenseValidateResponse(
        valid=True,
        tier=lic.tier.value,
        status=lic.status.value,
        features=features,
        message=f"License valid. Tier: {lic.tier.value}. {len(features)} features active.",
    )


class LicenseResponse(BaseModel):
    id: str
    license_key: str
    tier: str
    status: str
    instances_max: int
    instances_active: int
    trial_ends_at: str | None
    current_period_end: str | None
    created_at: str

    model_config = {"from_attributes": True}


class LicenseListResponse(BaseModel):
    licenses: list[LicenseResponse]


@router.get("", response_model=LicenseListResponse)
async def list_licenses(user: User = Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    """List all licenses for the authenticated user."""
    result = await db.execute(
        select(License).where(License.user_id == user.id).order_by(License.created_at.desc())
    )
    licenses = result.scalars().all()

    return LicenseListResponse(
        licenses=[
            LicenseResponse(
                id=str(lic.id),
                license_key=lic.license_key,
                tier=lic.tier.value,
                status=lic.status.value,
                instances_max=lic.instances_max,
                instances_active=lic.instances_active,
                trial_ends_at=lic.trial_ends_at.isoformat() if lic.trial_ends_at else None,
                current_period_end=lic.current_period_end.isoformat() if lic.current_period_end else None,
                created_at=lic.created_at.isoformat() if lic.created_at else "",
            )
            for lic in licenses
        ]
    )
