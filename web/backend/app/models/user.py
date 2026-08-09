"""User and License models."""
import uuid
from datetime import datetime

from sqlalchemy import String, Boolean, DateTime, ForeignKey, Enum as SAEnum, func
from sqlalchemy.dialects.postgresql import UUID
from sqlalchemy.orm import Mapped, mapped_column, relationship
import enum

from app.core.database import Base


class LicenseTier(str, enum.Enum):
    CORE = "core"
    PRO = "pro"
    ENTERPRISE = "enterprise"


class LicenseStatus(str, enum.Enum):
    ACTIVE = "active"
    TRIAL = "trial"
    GRACE = "grace"
    EXPIRED = "expired"
    REVOKED = "revoked"


class User(Base):
    __tablename__ = "users"

    id: Mapped[uuid.UUID] = mapped_column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    email: Mapped[str] = mapped_column(String(255), unique=True, nullable=False, index=True)
    name: Mapped[str | None] = mapped_column(String(255))
    avatar_url: Mapped[str | None] = mapped_column(String(1024))
    email_verified: Mapped[bool] = mapped_column(Boolean, default=False)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now())
    updated_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())

    licenses: Mapped[list["License"]] = relationship(back_populates="user", lazy="selectin")
    api_keys: Mapped[list["ApiKey"]] = relationship(back_populates="user", lazy="selectin")
    invoices: Mapped[list["Invoice"]] = relationship(back_populates="user", lazy="selectin")
    refresh_tokens: Mapped[list["RefreshToken"]] = relationship(back_populates="user", lazy="selectin")


class License(Base):
    __tablename__ = "licenses"

    id: Mapped[uuid.UUID] = mapped_column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    user_id: Mapped[uuid.UUID] = mapped_column(ForeignKey("users.id"), nullable=False)
    license_key: Mapped[str] = mapped_column(String(64), unique=True, nullable=False, index=True)
    tier: Mapped[LicenseTier] = mapped_column(SAEnum(LicenseTier), default=LicenseTier.CORE)
    status: Mapped[LicenseStatus] = mapped_column(SAEnum(LicenseStatus), default=LicenseStatus.TRIAL)
    machine_fingerprint: Mapped[str | None] = mapped_column(String(128))
    instances_max: Mapped[int] = mapped_column(default=3)
    instances_active: Mapped[int] = mapped_column(default=0)
    trial_ends_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True))
    current_period_start: Mapped[datetime | None] = mapped_column(DateTime(timezone=True))
    current_period_end: Mapped[datetime | None] = mapped_column(DateTime(timezone=True))
    stripe_subscription_id: Mapped[str | None] = mapped_column(String(255))
    stripe_customer_id: Mapped[str | None] = mapped_column(String(255))
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now())
    updated_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())

    user: Mapped["User"] = relationship(back_populates="licenses")
    invoices: Mapped[list["Invoice"]] = relationship(back_populates="license", lazy="selectin")
