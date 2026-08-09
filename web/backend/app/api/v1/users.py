"""
OVAV API v1 — Users endpoints
"""
from fastapi import APIRouter, Depends, HTTPException, status
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials
from sqlalchemy.ext.asyncio import AsyncSession
from pydantic import BaseModel, EmailStr, Field
from typing import Optional
from uuid import UUID

from app.core.database import get_db
from app.core.security import decode_token
from app.models.user import User

router = APIRouter(prefix="/users", tags=["Users"])
security = HTTPBearer()


# Schemas
class UserProfile(BaseModel):
    id: UUID
    email: str
    name: Optional[str] = None
    avatar_url: Optional[str] = None
    email_verified: bool = False
    created_at: str
    
    class Config:
        from_attributes = True


class UpdateUserRequest(BaseModel):
    name: Optional[str] = Field(None, max_length=100)
    avatar_url: Optional[str] = None


class UserResponse(BaseModel):
    user: UserProfile
    tier: str = "core"


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
@router.get("/me", response_model=UserProfile)
async def get_me(
    current_user: User = Depends(get_current_user)
):
    """
    GET /users/me - Obtener perfil del usuario actual
    """
    return UserProfile(
        id=current_user.id,
        email=current_user.email,
        name=current_user.name,
        avatar_url=current_user.avatar_url,
        email_verified=current_user.email_verified,
        created_at=current_user.created_at.isoformat() if current_user.created_at else ""
    )


@router.patch("/me", response_model=UserProfile)
async def update_me(
    update_data: UpdateUserRequest,
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db)
):
    """
    PATCH /users/me - Actualizar perfil del usuario
    """
    update_dict = {k: v for k, v in update_data.model_dump().items() if v is not None}
    
    if update_dict:
        from sqlalchemy import update
        stmt = (
            update(User)
            .where(User.id == current_user.id)
            .values(**update_dict)
            .returning(User)
        )
        result = await db.execute(stmt)
        await db.commit()
        updated_user = result.scalar_one()
    else:
        updated_user = current_user
    
    return UserProfile(
        id=updated_user.id,
        email=updated_user.email,
        name=updated_user.name,
        avatar_url=updated_user.avatar_url,
        email_verified=updated_user.email_verified,
        created_at=updated_user.created_at.isoformat() if updated_user.created_at else ""
    )


@router.delete("/me", status_code=status.HTTP_204_NO_CONTENT)
async def delete_me(
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db)
):
    """
    DELETE /users/me - Eliminar cuenta de usuario
    """
    from sqlalchemy import delete
    stmt = delete(User).where(User.id == current_user.id)
    await db.execute(stmt)
    await db.commit()
    return None
