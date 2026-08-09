"""Application configuration via environment variables."""
from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    ENV: str = "development"
    DEBUG: bool = True

    # Server
    HOST: str = "0.0.0.0"
    PORT: int = 8000

    # Database
    DATABASE_URL: str = "postgresql+asyncpg://ovav:ovav@localhost:5432/ovav"

    # Auth
    JWT_SECRET: str = "change-me-in-production"
    SECRET_KEY: str = "change-me-in-production-super-secret-key"
    JWT_ALGORITHM: str = "RS256"
    JWT_EXPIRE_MINUTES: int = 60
    MAGIC_LINK_EXPIRE_MINUTES: int = 15

    # Stripe
    STRIPE_SECRET_KEY: str = ""
    STRIPE_WEBHOOK_SECRET: str = ""
    STRIPE_PRICE_PRO_MONTHLY: str = ""
    STRIPE_PRICE_PRO_ANNUAL: str = ""
    STRIPE_PRICE_ENTERPRISE: str = ""

    # Resend
    RESEND_API_KEY: str = ""

    # Frontend
    FRONTEND_URL: str = "http://localhost:3000"

    # CORS
    CORS_ORIGINS: list[str] = ["http://localhost:3000", "http://localhost:8080"]

    # OAuth
    GOOGLE_CLIENT_ID: str = ""
    GOOGLE_CLIENT_SECRET: str = ""
    GITHUB_CLIENT_ID: str = ""
    GITHUB_CLIENT_SECRET: str = ""

    # OVAV License Bridge
    OVAV_LICENSE_GATEWAY_URL: str = ""

    class Config:
        env_file = ".env"
        env_file_encoding = "utf-8"


settings = Settings()
