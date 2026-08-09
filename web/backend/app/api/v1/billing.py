"""
OVAV API v1 — Billing endpoints
"""
from fastapi import APIRouter, Depends, HTTPException, status
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials
from sqlalchemy.ext.asyncio import AsyncSession
from pydantic import BaseModel
from typing import Optional, List
from uuid import UUID

from app.core.database import get_db
from app.core.security import decode_token
from app.models.user import User

router = APIRouter(prefix="/billing", tags=["Billing"])
security = HTTPBearer()


# Schemas
class InvoiceResponse(BaseModel):
    id: UUID
    amount: float
    currency: str
    status: str
    created_at: str
    paid_at: Optional[str] = None
    invoice_url: Optional[str] = None

    class Config:
        from_attributes = True


class SubscriptionResponse(BaseModel):
    tier: str
    status: str
    current_period_start: str
    current_period_end: Optional[str] = None
    cancel_at_period_end: bool = False

    class Config:
        from_attributes = True


class UpdateSubscriptionRequest(BaseModel):
    tier: Optional[str] = None
    cancel: Optional[bool] = None
    reactivate: Optional[bool] = None


# Dependencies
async def get_current_user(
    credentials: HTTPAuthorizationCredentials = Depends(security),
    db: AsyncSession = Depends(get_db)
) -> User:
    """Get current authenticated user."""
    token = credentials.credentials
    payload = decode_token(token)
    
    if not payload:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Token inválido o expirado"
        )
    
    user_id = payload.get("sub")
    if not user_id:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Token malformado"
        )
    
    result = await db.execute(
        "SELECT * FROM users WHERE id = :id",
        {"id": user_id}
    )
    user = result.fetchone()
    
    if not user:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Usuario no encontrado"
        )
    
    return user


# Endpoints
@router.get("/invoices", response_model=List[InvoiceResponse])
async def list_invoices(
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db)
):
    """
    GET /billing/invoices - Listar facturas del usuario
    """
    result = await db.execute(
        """
        SELECT * FROM invoices
        WHERE user_id = :user_id
        ORDER BY created_at DESC
        """,
        {"user_id": str(current_user.id)}
    )
    invoices = result.fetchall()
    
    return [
        InvoiceResponse(
            id=inv.id,
            amount=float(inv.amount) if inv.amount else 0,
            currency=inv.currency or "usd",
            status=inv.status,
            created_at=inv.created_at.isoformat() if inv.created_at else "",
            paid_at=inv.paid_at.isoformat() if inv.paid_at else None,
            invoice_url=None  # Stripe URL would go here
        )
        for inv in invoices
    ]


@router.get("/invoices/{invoice_id}", response_model=InvoiceResponse)
async def get_invoice(
    invoice_id: UUID,
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db)
):
    """
    GET /billing/invoices/{invoice_id} - Obtener detalle de factura
    """
    result = await db.execute(
        "SELECT * FROM invoices WHERE id = :id AND user_id = :user_id",
        {"id": str(invoice_id), "user_id": str(current_user.id)}
    )
    invoice = result.fetchone()
    
    if not invoice:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Factura no encontrada"
        )
    
    return InvoiceResponse(
        id=invoice.id,
        amount=float(invoice.amount) if invoice.amount else 0,
        currency=invoice.currency or "usd",
        status=invoice.status,
        created_at=invoice.created_at.isoformat() if invoice.created_at else "",
        paid_at=invoice.paid_at.isoformat() if invoice.paid_at else None,
        invoice_url=None
    )


@router.get("/subscription", response_model=SubscriptionResponse)
async def get_subscription(
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db)
):
    """
    GET /billing/subscription - Obtener suscripción actual
    """
    result = await db.execute(
        """
        SELECT * FROM licenses
        WHERE user_id = :user_id AND status IN ('active', 'trial', 'grace')
        ORDER BY created_at DESC LIMIT 1
        """,
        {"user_id": str(current_user.id)}
    )
    license = result.fetchone()
    
    if not license:
        return SubscriptionResponse(
            tier="core",
            status="none",
            current_period_start="",
            current_period_end=None,
            cancel_at_period_end=False
        )
    
    return SubscriptionResponse(
        tier=license.tier or "core",
        status=license.status,
        current_period_start=license.current_period_start.isoformat() if license.current_period_start else "",
        current_period_end=license.current_period_end.isoformat() if license.current_period_end else None,
        cancel_at_period_end=False
    )


@router.patch("/subscription", response_model=SubscriptionResponse)
async def update_subscription(
    request: UpdateSubscriptionRequest,
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db)
):
    """
    PATCH /billing/subscription - Actualizar suscripción
    """
    result = await db.execute(
        """
        SELECT * FROM licenses
        WHERE user_id = :user_id AND status IN ('active', 'trial', 'grace')
        ORDER BY created_at DESC LIMIT 1
        """,
        {"user_id": str(current_user.id)}
    )
    license = result.fetchone()
    
    if not license:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="No existe suscripción activa"
        )
    
    if request.tier:
        await db.execute(
            "UPDATE licenses SET tier = :tier WHERE id = :id",
            {"tier": request.tier, "id": str(license.id)}
        )
    
    if request.cancel:
        await db.execute(
            "UPDATE licenses SET status = 'canceled' WHERE id = :id",
            {"id": str(license.id)}
        )
    
    if request.reactivate:
        await db.execute(
            "UPDATE licenses SET status = 'active' WHERE id = :id",
            {"id": str(license.id)}
        )
    
    await db.commit()
    
    # Fetch updated
    result = await db.execute(
        "SELECT * FROM licenses WHERE id = :id",
        {"id": str(license.id)}
    )
    updated = result.fetchone()
    
    return SubscriptionResponse(
        tier=updated.tier or "core",
        status=updated.status,
        current_period_start=updated.current_period_start.isoformat() if updated.current_period_start else "",
        current_period_end=updated.current_period_end.isoformat() if updated.current_period_end else None,
        cancel_at_period_end=False
    )


@router.post("/portal")
async def create_portal_session(
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db)
):
    """
    POST /billing/portal - Crear sesión de portal de Stripe
    """
    # In production, this would create a Stripe Customer Portal session
    # For now, return placeholder
    return {
        "url": f"https://billing.stripe.com/session/{current_user.id}",
        "expires_at": None
    }
