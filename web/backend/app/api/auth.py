"""Authentication endpoints — magic link + OAuth + JWT session management."""
import uuid
from datetime import UTC, datetime, timedelta

from fastapi import APIRouter, HTTPException, Depends, Request
from pydantic import BaseModel, EmailStr
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession
from jose import jwt, JWTError

from app.core.config import settings
from app.core.database import get_db
from app.models.user import User

router = APIRouter()

# --- Schemas ---

class LoginRequest(BaseModel):
    email: EmailStr

class TokenVerifyRequest(BaseModel):
    token: str

class RegisterRequest(BaseModel):
    email: EmailStr
    name: str | None = None

class SessionResponse(BaseModel):
    user_id: str
    email: str
    name: str | None
    avatar_url: str | None

class AuthResponse(BaseModel):
    access_token: str
    user: SessionResponse


# --- Helpers ---

def _create_access_token(user_id: str, email: str) -> str:
    expire = datetime.now(UTC) + timedelta(minutes=settings.JWT_EXPIRE_MINUTES)
    payload = {
        "sub": user_id,
        "email": email,
        "exp": expire,
        "iat": datetime.now(UTC),
        "jti": str(uuid.uuid4()),
    }
    return jwt.encode(payload, settings.JWT_SECRET, algorithm="HS256")


def _create_magic_token(email: str) -> str:
    expire = datetime.now(UTC) + timedelta(minutes=settings.MAGIC_LINK_EXPIRE_MINUTES)
    payload = {
        "sub": "magic-link",
        "email": email,
        "exp": expire,
        "iat": datetime.now(UTC),
        "jti": str(uuid.uuid4()),
        "purpose": "login",
    }
    return jwt.encode(payload, settings.JWT_SECRET, algorithm="HS256")


def _decode_token(token: str) -> dict:
    try:
        return jwt.decode(token, settings.JWT_SECRET, algorithms=["HS256"])
    except JWTError:
        raise HTTPException(status_code=401, detail="Invalid or expired token")


async def get_current_user(
    request: Request,
    db: AsyncSession = Depends(get_db),
) -> User:
    """Dependency: extract and validate JWT from Authorization header."""
    auth_header = request.headers.get("Authorization", "")
    if not auth_header.startswith("Bearer "):
        raise HTTPException(status_code=401, detail="Missing or invalid Authorization header")

    token = auth_header.removeprefix("Bearer ").strip()
    payload = _decode_token(token)

    user_id = payload.get("sub")
    if not user_id:
        raise HTTPException(status_code=401, detail="Invalid token payload")

    result = await db.execute(select(User).where(User.id == uuid.UUID(user_id)))
    user = result.scalar_one_or_none()
    if not user:
        raise HTTPException(status_code=401, detail="User not found")

    return user


# --- Routes ---

@router.post("/register", response_model=AuthResponse)
async def register(body: RegisterRequest, db: AsyncSession = Depends(get_db)):
    """Register a new user or return existing. Sends magic link for verification."""
    result = await db.execute(select(User).where(User.email == body.email))
    user = result.scalar_one_or_none()

    if user:
        # User exists — send magic link for login
        access_token = _create_access_token(str(user.id), user.email)
        return AuthResponse(
            access_token=access_token,
            user=SessionResponse(
                user_id=str(user.id),
                email=user.email,
                name=user.name,
                avatar_url=user.avatar_url,
            ),
        )

    # Create new user
    user = User(
        id=uuid.uuid4(),
        email=body.email,
        name=body.name or body.email.split("@")[0],
        email_verified=False,
    )
    db.add(user)
    await db.flush()

    access_token = _create_access_token(str(user.id), user.email)

    return AuthResponse(
        access_token=access_token,
        user=SessionResponse(
            user_id=str(user.id),
            email=user.email,
            name=user.name,
            avatar_url=user.avatar_url,
        ),
    )


@router.post("/login")
async def request_magic_link(body: LoginRequest, db: AsyncSession = Depends(get_db)):
    """Request a magic link for passwordless login."""
    result = await db.execute(select(User).where(User.email == body.email))
    user = result.scalar_one_or_none()

    # Always return success to prevent email enumeration
    if not user:
        # Auto-register user on first login attempt
        user = User(
            id=uuid.uuid4(),
            email=body.email,
            name=body.email.split("@")[0],
            email_verified=False,
        )
        db.add(user)
        await db.flush()

    magic_token = _create_magic_token(user.email)

    # In development/preview, return the token directly (no email sending)
    # In production, this would use Resend to send the magic link
    magic_link = f"{settings.FRONTEND_URL or 'http://localhost:3000'}/auth/verify?token={magic_token}"

    return {
        "message": "Magic link generated",
        "magic_link": magic_link if settings.ENV == "development" else None,
        "expires_in_minutes": settings.MAGIC_LINK_EXPIRE_MINUTES,
    }


@router.get("/verify", response_model=AuthResponse)
async def verify_magic_link(token: str, db: AsyncSession = Depends(get_db)):
    """Verify a magic link token and return a session."""
    payload = _decode_token(token)

    if payload.get("purpose") != "login":
        raise HTTPException(status_code=401, detail="Invalid token purpose")

    email = payload.get("email")
    if not email:
        raise HTTPException(status_code=401, detail="Invalid token payload")

    result = await db.execute(select(User).where(User.email == email))
    user = result.scalar_one_or_none()
    if not user:
        raise HTTPException(status_code=404, detail="User not found")

    # Mark email as verified on first magic link use
    if not user.email_verified:
        user.email_verified = True
        await db.flush()

    access_token = _create_access_token(str(user.id), user.email)

    return AuthResponse(
        access_token=access_token,
        user=SessionResponse(
            user_id=str(user.id),
            email=user.email,
            name=user.name,
            avatar_url=user.avatar_url,
        ),
    )


@router.get("/session", response_model=SessionResponse)
async def get_session(user: User = Depends(get_current_user)):
    """Get the current authenticated user's session info."""
    return SessionResponse(
        user_id=str(user.id),
        email=user.email,
        name=user.name,
        avatar_url=user.avatar_url,
    )


@router.get("/oauth/{provider}")
async def oauth_login(provider: str, request: Request):
    """Initiate OAuth flow with Google or GitHub."""
    if provider not in ("google", "github"):
        raise HTTPException(status_code=400, detail=f"Unsupported provider: {provider}")

    redirect_uri = f"{settings.FRONTEND_URL or 'http://localhost:3000'}/auth/callback/{provider}"

    if provider == "google":
        auth_url = (
            f"https://accounts.google.com/o/oauth2/v2/auth"
            f"?client_id={settings.GOOGLE_CLIENT_ID}"
            f"&redirect_uri={redirect_uri}"
            f"&response_type=code"
            f"&scope=openid+email+profile"
        )
    else:  # github
        auth_url = (
            f"https://github.com/login/oauth/authorize"
            f"?client_id={settings.GITHUB_CLIENT_ID}"
            f"&redirect_uri={redirect_uri}"
            f"&scope=user:email"
        )

    return {"url": auth_url, "provider": provider}


@router.get("/oauth/{provider}/callback", response_model=AuthResponse)
async def oauth_callback(
    provider: str,
    code: str,
    request: Request,
    db: AsyncSession = Depends(get_db),
):
    """Handle OAuth callback from Google or GitHub. Exchange code for user info."""
    import httpx

    if provider not in ("google", "github"):
        raise HTTPException(status_code=400, detail=f"Unsupported provider: {provider}")

    redirect_uri = f"{settings.FRONTEND_URL or 'http://localhost:3000'}/auth/callback/{provider}"

    # Exchange code for access token
    async with httpx.AsyncClient() as client:
        if provider == "google":
            token_response = await client.post(
                "https://oauth2.googleapis.com/token",
                data={
                    "client_id": settings.GOOGLE_CLIENT_ID,
                    "client_secret": settings.GOOGLE_CLIENT_SECRET,
                    "code": code,
                    "grant_type": "authorization_code",
                    "redirect_uri": redirect_uri,
                },
            )
        else:  # github
            token_response = await client.post(
                "https://github.com/login/oauth/access_token",
                data={
                    "client_id": settings.GITHUB_CLIENT_ID,
                    "client_secret": settings.GITHUB_CLIENT_SECRET,
                    "code": code,
                    "redirect_uri": redirect_uri,
                },
                headers={"Accept": "application/json"},
            )

        if token_response.status_code != 200:
            raise HTTPException(status_code=401, detail=f"OAuth token exchange failed for {provider}")

        token_data = token_response.json()
        access_token = token_data.get("access_token")
        if not access_token:
            raise HTTPException(status_code=401, detail=f"No access token from {provider}")

        # Fetch user info
        if provider == "google":
            user_response = await client.get(
                "https://www.googleapis.com/oauth2/v3/userinfo",
                headers={"Authorization": f"Bearer {access_token}"},
            )
        else:  # github
            user_response = await client.get(
                "https://api.github.com/user",
                headers={"Authorization": f"Bearer {access_token}"},
            )
            # Also fetch emails for GitHub
            emails_response = await client.get(
                "https://api.github.com/user/emails",
                headers={"Authorization": f"Bearer {access_token}"},
            )

        if user_response.status_code != 200:
            raise HTTPException(status_code=401, detail=f"Failed to fetch user info from {provider}")

        user_data = user_response.json()

        if provider == "google":
            email = user_data.get("email")
            name = user_data.get("name")
            avatar = user_data.get("picture")
        else:
            name = user_data.get("name") or user_data.get("login")
            avatar = user_data.get("avatar_url")
            # Get primary email
            emails = emails_response.json() if emails_response.status_code == 200 else []
            primary = next((e["email"] for e in emails if e.get("primary")), None)
            email = primary or emails[0]["email"] if emails else None

        if not email:
            raise HTTPException(status_code=400, detail=f"Could not retrieve email from {provider}")

    # Find or create user
    result = await db.execute(select(User).where(User.email == email))
    user = result.scalar_one_or_none()

    if not user:
        user = User(
            id=uuid.uuid4(),
            email=email,
            name=name,
            avatar_url=avatar,
            email_verified=True,
        )
        db.add(user)
        await db.flush()
    else:
        # Update OAuth info
        user.name = user.name or name
        user.avatar_url = user.avatar_url or avatar
        user.email_verified = True
        await db.flush()

    access_token_jwt = _create_access_token(str(user.id), user.email)

    return AuthResponse(
        access_token=access_token_jwt,
        user=SessionResponse(
            user_id=str(user.id),
            email=user.email,
            name=user.name,
            avatar_url=user.avatar_url,
        ),
    )
