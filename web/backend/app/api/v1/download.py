"""
OVAV API v1 — Download endpoints
"""
from fastapi import APIRouter, Depends, HTTPException, status
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials
from sqlalchemy.ext.asyncio import AsyncSession
from pydantic import BaseModel
from typing import Optional, List
import platform

from app.core.database import get_db
from app.core.security import decode_token
from app.models.user import User

router = APIRouter(prefix="/download", tags=["Download"])
security = HTTPBearer()

# Current CLI version
CLI_VERSION = "3.0.0"
API_BASE = "https://api.ovav.dev/v1"


# Schemas
class PlatformDownload(BaseModel):
    platform: str
    arch: str
    url: str
    checksum: str
    size_mb: int


class VersionInfo(BaseModel):
    version: str
    release_date: str
    release_notes_url: str
    minimum_os_version: dict
    downloads: List[PlatformDownload]


class DownloadUrls(BaseModel):
    version: str
    platforms: List[PlatformDownload]


# Dependencies
async def get_current_user(
    credentials: HTTPAuthorizationCredentials = Depends(security),
    db: AsyncSession = Depends(get_db)
) -> User:
    """Get current authenticated user (optional for download)."""
    if not credentials:
        return None
    
    token = credentials.credentials
    payload = decode_token(token)
    
    if not payload:
        return None
    
    user_id = payload.get("sub")
    if not user_id:
        return None
    
    result = await db.execute(
        "SELECT * FROM users WHERE id = :id",
        {"id": user_id}
    )
    return result.fetchone()


def get_detected_platform() -> tuple[str, str]:
    """Detect current platform and architecture."""
    system = platform.system().lower()
    arch = platform.machine().lower()
    
    # Map to our platform names
    platform_map = {
        "darwin": ("macos", "universal"),
        "linux": ("linux", "x86_64"),
        "windows": ("windows", "x86_64")
    }
    
    if system in platform_map:
        return platform_map[system]
    
    # ARM detection
    if arch in ["arm64", "aarch64"]:
        if system == "darwin":
            return ("macos", "arm64")
        return ("linux", "arm64")
    
    return (system, arch)


# Endpoints
@router.get("/cli", response_model=DownloadUrls)
async def get_cli_downloads(
    credentials: Optional[HTTPAuthorizationCredentials] = Depends(
        HTTPBearer(auto_error=False)
    )
):
    """
    GET /download/cli - Obtener URLs de descarga para todas las plataformas
    """
    platforms = [
        PlatformDownload(
            platform="macos",
            arch="universal",
            url=f"{API_BASE}/download/cli/macos/universal",
            checksum="sha256:placeholder",
            size_mb=45
        ),
        PlatformDownload(
            platform="macos",
            arch="arm64",
            url=f"{API_BASE}/download/cli/macos/arm64",
            checksum="sha256:placeholder",
            size_mb=38
        ),
        PlatformDownload(
            platform="linux",
            arch="x86_64",
            url=f"{API_BASE}/download/cli/linux/x86_64",
            checksum="sha256:placeholder",
            size_mb=42
        ),
        PlatformDownload(
            platform="linux",
            arch="arm64",
            url=f"{API_BASE}/download/cli/linux/arm64",
            checksum="sha256:placeholder",
            size_mb=40
        ),
        PlatformDownload(
            platform="windows",
            arch="x86_64",
            url=f"{API_BASE}/download/cli/windows/x86_64.exe",
            checksum="sha256:placeholder",
            size_mb=44
        ),
    ]
    
    return DownloadUrls(
        version=CLI_VERSION,
        platforms=platforms
    )


@router.get("/cli/{platform}", response_model=PlatformDownload)
async def get_platform_download(
    platform: str,
    arch: str = "x86_64"
):
    """
    GET /download/cli/{platform} - Obtener URL de descarga para plataforma específica
    """
    valid_platforms = ["macos", "linux", "windows"]
    
    if platform.lower() not in valid_platforms:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Plataforma inválida. Options: {valid_platforms}"
        )
    
    return PlatformDownload(
        platform=platform.lower(),
        arch=arch,
        url=f"{API_BASE}/download/cli/{platform.lower()}/{arch}",
        checksum="sha256:placeholder",
        size_mb=42
    )


@router.get("/version", response_model=VersionInfo)
async def get_version_info():
    """
    GET /download/version - Obtener información de versión actual
    """
    return VersionInfo(
        version=CLI_VERSION,
        release_date="2026-08-07",
        release_notes_url=f"https://docs.ovav.dev/changelog/{CLI_VERSION}",
        minimum_os_version={
            "macos": "12.0",
            "linux": "Ubuntu 20.04 / Debian 11",
            "windows": "Windows 10"
        },
        downloads=[
            PlatformDownload(
                platform="macos",
                arch="universal",
                url=f"{API_BASE}/download/cli/macos/universal",
                checksum="sha256:placeholder",
                size_mb=45
            ),
            PlatformDownload(
                platform="windows",
                arch="x86_64",
                url=f"{API_BASE}/download/cli/windows/x86_64.exe",
                checksum="sha256:placeholder",
                size_mb=44
            ),
            PlatformDownload(
                platform="linux",
                arch="x86_64",
                url=f"{API_BASE}/download/cli/linux/x86_64",
                checksum="sha256:placeholder",
                size_mb=42
            ),
        ]
    )


@router.get("/detect")
async def detect_platform():
    """
    GET /download/detect - Detectar plataforma del cliente
    """
    sys, arch = get_detected_platform()
    return {
        "platform": sys,
        "arch": arch,
        "recommended_download": f"{API_BASE}/download/cli/{sys}/{arch}",
        "version": CLI_VERSION
    }
