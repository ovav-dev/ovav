"""
OVAV API v1 — API Keys endpoints
"""
from fastapi import APIRouter, Depends, HTTPException, status, Query
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials
from sqlalchemy.ext.asyncio import AsyncSession
from pydantic import BaseModel, Field
from typing import Optional, List
from uuid import UUID, uuid4
from datetime import datetime, timedelta
import hashlib
import secrets

from app.core.database import get_db
from app.core.security import decode_token
from app.models.user import User

router = APIRouter(prefix="/api-keys", tags=["API Keys"])
security = HTTPBearer()


# Schemas
class ApiKeyResponse(BaseModel):
    id: UUID
    name: str
    key_prefix: str
    expires_at: Optional[str] = None
    created_at: str
    last_used_at: Optional[str] = None

    class Config:
        from_attributes = True


class CreateApiKeyRequest(BaseModel):
    name: str = Field(..., min_length=1, max_length=100)
    expires_in_days: Optional[int] = Field(None, ge=1, le=365)


class ApiKeyCreatedResponse(BaseModel):
    id: UUID
    name: str
    key: str  # Full key shown only once
    key_prefix: str
    expires_at: Optional[str] = None
    created_at: str


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


def generate_key() -> tuple[str, str]:
    """Generate a new API key. Returns (full_key, hashed_key)."""
    prefix = secrets.token_hex(4)
    random_part = secrets.token_hex(24)
    full_key = f"ovav_{prefix}_{random_part}"
    hashed = hashlib.sha256(full_key.encode()).hexdigest()
    return full_key, hashed


# Endpoints
@router.get("", response_model=List[ApiKeyResponse])
async def list_api_keys(
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db)
):
    """
    GET /api-keys - Listar todas las API keys del usuario
    """
    result = await db.execute(
        """
        SELECT id, name, key_prefix, expires_at, created_at, last_used_at
        FROM api_keys
        WHERE user_id = :user_id
        ORDER BY created_at DESC
        """,
        {"user_id": current_user.id}
    )
    keys = result.fetchall()
    
    return [
        ApiKeyResponse(
            id=key.id,
            name=key.name,
            key_prefix=key.key_prefix,
            expires_at=key.expires_at.isoformat() if key.expires_at else None,
            created_at=key.created_at.isoformat() if key.created_at else "",
            last_used_at=key.last_used_at.isoformat() if key.last_used_at else None
        )
        for key in keys
    ]


@router.post("", response_model=ApiKeyCreatedResponse, status_code=status.HTTP_201_CREATED)
async def create_api_key(
    request: CreateApiKeyRequest,
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db)
):
    """
    POST /api-keys - Crear nueva API key
    """
    full_key, hashed_key = generate_key()
    
    expires_at = None
    if request.expires_in_days:
        expires_at = datetime.utcnow() + timedelta(days=request.expires_in_days)
    
    key_prefix = full_key.split("_")[1] + "_" + full_key.split("_")[2][:4]
    
    # Insert into database
    await db.execute(
        """
        INSERT INTO api_keys (id, user_id, name, key_hash, key_prefix, expires_at, created_at)
        VALUES (:id, :user_id, :name, :key_hash, :key_prefix, :expires_at, :created_at)
        """,
        {
            "id": str(uuid4()),
            "user_id": str(current_user.id),
            "name": request.name,
            "key_hash": hashed_key,
            "key_prefix": key_prefix,
            "expires_at": expires_at,
            "created_at": datetime.utcnow()
        }
    )
    await db.commit()
    
    # Get the created key
    result = await db.execute(
        "SELECT * FROM api_keys WHERE user_id = :user_id AND name = :name ORDER BY created_at DESC LIMIT 1",
        {"user_id": str(current_user.id), "name": request.name}
    )
    created = result.fetchone()
    
    return ApiKeyCreatedResponse(
        id=created.id,
        name=created.name,
        key=full_key,  # Full key shown only once!
        key_prefix=created.key_prefix,
        expires_at=created.expires_at.isoformat() if created.expires_at else None,
        created_at=created.created_at.isoformat() if created.created_at else ""
    )


@router.get("/{key_id}", response_model=ApiKeyResponse)
async def get_api_key(
    key_id: UUID,
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db)
):
    """
    GET /api-keys/{key_id} - Obtener detalle de API key
    """
    result = await db.execute(
        "SELECT * FROM api_keys WHERE id = :id AND user_id = :user_id",
        {"id": str(key_id), "user_id": str(current_user.id)}
    )
    key = result.fetchone()
    
    if not key:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="API key no encontrada"
        )
    
    return ApiKeyResponse(
        id=key.id,
        name=key.name,
        key_prefix=key.key_prefix,
        expires_at=key.expires_at.isoformat() if key.expires_at else None,
        created_at=key.created_at.isoformat() if key.created_at else "",
        last_used_at=key.last_used_at.isoformat() if key.last_used_at else None
    )


@router.delete("/{key_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_api_key(
    key_id: UUID,
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db)
):
    """
    DELETE /api-keys/{key_id} - Revocar API key
    """
    result = await db.execute(
        "DELETE FROM api_keys WHERE id = :id AND user_id = :user_id RETURNING id",
        {"id": str(key_id), "user_id": str(current_user.id)}
    )
    deleted = result.fetchone()
    
    if not deleted:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="API key no encontrada"
        )
    
    await db.commit()
    return None


@router.post("/{key_id}/rotate", response_model=ApiKeyCreatedResponse)
async def rotate_api_key(
    key_id: UUID,
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db)
):
    """
    POST /api-keys/{key_id}/rotate - Rotar API key
    """
    # Get existing key
    result = await db.execute(
        "SELECT * FROM api_keys WHERE id = :id AND user_id = :user_id",
        {"id": str(key_id), "user_id": str(current_user.id)}
    )
    existing = result.fetchone()
    
    if not existing:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="API key no encontrada"
        )
    
    # Delete old key
    await db.execute(
        "DELETE FROM api_keys WHERE id = :id",
        {"id": str(key_id)}
    )
    await db.commit()
    
    # Create new key with same name and expiry
    full_key, hashed_key = generate_key()
    key_prefix = full_key.split("_")[1] + "_" + full_key.split("_")[2][:4]
    
    await db.execute(
        """
        INSERT INTO api_keys (id, user_id, name, key_hash, key_prefix, expires_at, created_at)
        VALUES (:id, :user_id, :name, :key_hash, :key_prefix, :expires_at, :created_at)
        """,
        {
            "id": str(uuid4()),
            "user_id": str(current_user.id),
            "name": existing.name,
            "key_hash": hashed_key,
            "key_prefix": key_prefix,
            "expires_at": existing.expires_at,
            "created_at": datetime.utcnow()
        }
    )
    await db.commit()
    
    return ApiKeyCreatedResponse(
        id=uuid4(),  # New ID
        name=existing.name,
        key=full_key,
        key_prefix=key_prefix,
        expires_at=existing.expires_at.isoformat() if existing.expires_at else None,
        created_at=datetime.utcnow().isoformat()
    )
