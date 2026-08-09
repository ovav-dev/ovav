"""Checkout session creation — Stripe integration with dev mode fallback."""
import uuid
from datetime import UTC, datetime, timedelta

from fastapi import APIRouter, HTTPException, Depends, Request
from pydantic import BaseModel, EmailStr
import stripe
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.config import settings
from app.core.database import get_db
from app.models.user import User, License, LicenseTier, LicenseStatus

router = APIRouter()
stripe.api_key = settings.STRIPE_SECRET_KEY

PRICE_MAP = {
    "pro_monthly": settings.STRIPE_PRICE_PRO_MONTHLY,
    "pro_annual": settings.STRIPE_PRICE_PRO_ANNUAL,
    "enterprise": settings.STRIPE_PRICE_ENTERPRISE,
}

TIER_MAP = {
    "pro_monthly": LicenseTier.PRO,
    "pro_annual": LicenseTier.PRO,
    "enterprise": LicenseTier.ENTERPRISE,
}


class CheckoutRequest(BaseModel):
    tier: str
    email: EmailStr
    success_url: str = "http://localhost:3000/dashboard?session_id={CHECKOUT_SESSION_ID}"
    cancel_url: str = "http://localhost:3000/checkout"


def _generate_license_key(tier: str) -> str:
    prefix = {"pro_monthly": "ovav-pro", "pro_annual": "ovav-pro", "enterprise": "ovav-ent"}.get(tier, "ovav-core")
    return f"{prefix}-{uuid.uuid4().hex[:32]}"


@router.post("/session")
async def create_checkout_session(body: CheckoutRequest, db: AsyncSession = Depends(get_db)):
    price_id = PRICE_MAP.get(body.tier)
    if not price_id:
        raise HTTPException(status_code=400, detail=f"Invalid tier: {body.tier}")

    # Find or create user
    result = await db.execute(select(User).where(User.email == body.email))
    user = result.scalar_one_or_none()
    if not user:
        user = User(id=uuid.uuid4(), email=body.email, name=body.email.split("@")[0])
        db.add(user)
        await db.flush()

    # In development mode or without Stripe keys, create a trial license directly
    if settings.ENV == "development" or not price_id:
        license_key = _generate_license_key(body.tier)
        license = License(
            id=uuid.uuid4(),
            user_id=user.id,
            license_key=license_key,
            tier=TIER_MAP.get(body.tier, LicenseTier.PRO),
            status=LicenseStatus.TRIAL,
            instances_max=3 if body.tier != "enterprise" else 99,
            trial_ends_at=datetime.now(UTC) + timedelta(days=14),
            current_period_start=datetime.now(UTC),
            current_period_end=datetime.now(UTC) + timedelta(days=14),
        )
        db.add(license)
        await db.flush()

        # Generate access token for immediate dashboard access
        from app.api.auth import _create_access_token
        access_token = _create_access_token(str(user.id), user.email)

        return {
            "url": None,
            "session_id": str(license.id),
            "access_token": access_token,
            "message": "Trial license created (dev mode)",
        }

    # Production: create Stripe checkout session
    try:
        session = stripe.checkout.Session.create(
            customer_email=body.email,
            line_items=[{"price": price_id, "quantity": 1}],
            mode="subscription",
            success_url=body.success_url,
            cancel_url=body.cancel_url,
            metadata={"tier": body.tier, "user_id": str(user.id)},
            subscription_data={"trial_period_days": 14},
        )
        return {"url": session.url, "session_id": session.id}
    except stripe.error.StripeError as e:
        raise HTTPException(status_code=400, detail=str(e))


@router.post("/webhook")
async def stripe_webhook(request: Request, db: AsyncSession = Depends(get_db)):
    """Handle Stripe webhook events."""
    payload = await request.body()
    sig_header = request.headers.get("stripe-signature", "")

    try:
        event = stripe.Webhook.construct_event(
            payload, sig_header, settings.STRIPE_WEBHOOK_SECRET
        )
    except (ValueError, stripe.error.SignatureVerificationError):
        raise HTTPException(status_code=400, detail="Invalid webhook signature")

    # Handle subscription events
    if event.type == "checkout.session.completed":
        session = event.data.object
        user_id = session.get("metadata", {}).get("user_id")
        tier = session.get("metadata", {}).get("tier")

        if user_id:
            result = await db.execute(select(User).where(User.id == uuid.UUID(user_id)))
            user = result.scalar_one_or_none()
            if user:
                license_key = _generate_license_key(tier or "pro_annual")
                license = License(
                    id=uuid.uuid4(),
                    user_id=user.id,
                    license_key=license_key,
                    tier=TIER_MAP.get(tier, LicenseTier.PRO),
                    status=LicenseStatus.ACTIVE,
                    stripe_subscription_id=session.get("subscription"),
                    stripe_customer_id=session.get("customer"),
                    instances_max=3 if tier != "enterprise" else 99,
                    current_period_start=datetime.now(UTC),
                    current_period_end=datetime.now(UTC) + timedelta(days=30),
                )
                db.add(license)
                await db.flush()

    elif event.type == "customer.subscription.deleted":
        subscription = event.data.object
        result = await db.execute(
            select(License).where(License.stripe_subscription_id == subscription.id)
        )
        license = result.scalar_one_or_none()
        if license:
            license.status = LicenseStatus.EXPIRED
            await db.flush()

    return {"status": "ok"}
