"""Database engine and session management."""
import os
from sqlalchemy.ext.asyncio import AsyncSession, create_async_engine, async_sessionmaker
from sqlalchemy.orm import DeclarativeBase

from app.core.config import settings


def get_database_url() -> str:
    """Get database URL with fallback for local development."""
    env = os.getenv("ENV", settings.ENV)
    
    if env == "development" or env == "test":
        # Use SQLite for local development
        return "sqlite+aiosqlite:///./ovav_dev.db"
    
    return settings.DATABASE_URL


engine = create_async_engine(
    get_database_url(),
    echo=settings.DEBUG
)
AsyncSessionLocal = async_sessionmaker(engine, class_=AsyncSession, expire_on_commit=False)


class Base(DeclarativeBase):
    pass


async def get_db() -> AsyncSession:
    async with AsyncSessionLocal() as session:
        try:
            yield session
            await session.commit()
        except Exception:
            await session.rollback()
            raise


async def init_db():
    """Initialize database tables."""
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)


async def close_db():
    """Close database connection."""
    await engine.dispose()
