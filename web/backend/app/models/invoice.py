"""
OVAV Models — Invoice
"""
from sqlalchemy import Column, String, DateTime, ForeignKey, Numeric, Enum
from sqlalchemy.dialects.postgresql import UUID
from sqlalchemy.orm import relationship
from datetime import datetime
from uuid import uuid4
import enum

from app.core.database import Base


class InvoiceStatus(str, enum.Enum):
    PENDING = "pending"
    PAID = "paid"
    FAILED = "failed"
    REFUNDED = "refunded"
    VOID = "void"


class Invoice(Base):
    __tablename__ = "invoices"
    
    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid4)
    user_id = Column(UUID(as_uuid=True), ForeignKey("users.id"), nullable=False)
    license_id = Column(UUID(as_uuid=True), ForeignKey("licenses.id"), nullable=True)
    stripe_invoice_id = Column(String(100), nullable=True, unique=True)
    amount = Column(Numeric(10, 2), nullable=False, default=0)
    currency = Column(String(3), default="usd")
    status = Column(String(20), default=InvoiceStatus.PENDING.value)
    created_at = Column(DateTime, default=datetime.utcnow)
    paid_at = Column(DateTime, nullable=True)
    
    # Relationships
    user = relationship("User", back_populates="invoices")
    license = relationship("License", back_populates="invoices")
    
    def __repr__(self):
        return f"<Invoice {self.id} - {self.amount} {self.currency} ({self.status})>"
    
    def mark_paid(self):
        """Mark invoice as paid."""
        self.status = InvoiceStatus.PAID.value
        self.paid_at = datetime.utcnow()
