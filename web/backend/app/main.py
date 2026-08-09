"""
OVAV Product — Backend API
FastAPI application entry point.
"""
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from slowapi import Limiter, _rate_limit_exceeded_handler
from slowapi.errors import RateLimitExceeded
from slowapi.util import get_remote_address

from app.api import auth, checkout, licenses, health
from app.api.v1 import users, api_keys, billing, download
from app.core.config import settings
from app.core.database import engine, Base, init_db, close_db


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Startup/shutdown events."""
    # Create tables on startup (Alembic handles migrations in production)
    await init_db()
    yield
    await close_db()


limiter = Limiter(key_func=get_remote_address, default_limits=["60/minute"])
app = FastAPI(
    title="OVAV Product API",
    version="1.0.0",
    docs_url="/docs" if settings.ENV == "development" else None,
    redoc_url=None,
    lifespan=lifespan,
)

app.state.limiter = limiter
app.add_exception_handler(RateLimitExceeded, _rate_limit_exceeded_handler)

app.add_middleware(
    CORSMiddleware,
    allow_origins=settings.CORS_ORIGINS,
    allow_credentials=True,
    allow_methods=["GET", "POST", "PUT", "DELETE"],
    allow_headers=["Authorization", "Content-Type"],
)

# Routers
app.include_router(health.router, tags=["Health"])
app.include_router(auth.router, prefix="/auth", tags=["Auth"])
app.include_router(checkout.router, prefix="/checkout", tags=["Checkout"])
app.include_router(licenses.router, prefix="/licenses", tags=["Licenses"])

# API v1
app.include_router(users.router, prefix="/v1", tags=["Users"])
app.include_router(api_keys.router, prefix="/v1", tags=["API Keys"])
app.include_router(billing.router, prefix="/v1", tags=["Billing"])
app.include_router(download.router, prefix="/v1", tags=["Download"])


@app.get("/")
async def root():
    return {"product": "OVAV", "version": "1.0.0", "status": "operational"}
