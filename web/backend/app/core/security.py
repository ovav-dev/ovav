"""
OVAV Security — JWT utilities
"""
from datetime import datetime, timedelta
from typing import Optional, Dict, Any
from jose import jwt, JWTError
from app.core.config import settings


ALGORITHM = "HS256"


def create_access_token(
    user_id: str,
    email: str,
    tier: str = "core",
    expires_delta: Optional[timedelta] = None
) -> str:
    """
    Create JWT access token.
    
    Args:
        user_id: User UUID
        email: User email
        tier: License tier (core, pro, enterprise)
        expires_delta: Token expiry (default: 15 minutes)
    
    Returns:
        Encoded JWT token
    """
    if expires_delta is None:
        expires_delta = timedelta(minutes=15)
    
    expire = datetime.utcnow() + expires_delta
    
    payload = {
        "sub": user_id,
        "email": email,
        "tier": tier,
        "type": "access",
        "exp": expire,
        "iat": datetime.utcnow()
    }
    
    return jwt.encode(payload, settings.SECRET_KEY, algorithm=ALGORITHM)


def create_refresh_token(user_id: str) -> str:
    """
    Create JWT refresh token.
    
    Args:
        user_id: User UUID
    
    Returns:
        Encoded JWT refresh token
    """
    expire = datetime.utcnow() + timedelta(days=7)
    
    payload = {
        "sub": user_id,
        "type": "refresh",
        "exp": expire,
        "iat": datetime.utcnow()
    }
    
    return jwt.encode(payload, settings.SECRET_KEY, algorithm=ALGORITHM)


def decode_token(token: str) -> Optional[Dict[str, Any]]:
    """
    Decode and validate JWT token.
    
    Args:
        token: JWT token string
    
    Returns:
        Token payload or None if invalid
    """
    try:
        payload = jwt.decode(token, settings.SECRET_KEY, algorithms=[ALGORITHM])
        return payload
    except JWTError:
        return None


def verify_token(token: str, token_type: str = "access") -> Optional[Dict[str, Any]]:
    """
    Verify token and check type.
    
    Args:
        token: JWT token string
        token_type: Expected token type (access, refresh)
    
    Returns:
        Token payload or None if invalid
    """
    payload = decode_token(token)
    
    if not payload:
        return None
    
    if payload.get("type") != token_type:
        return None
    
    return payload
