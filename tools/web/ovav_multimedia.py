#!/usr/bin/env python3
"""OVAV Multimedia Tools — imágenes, video, transcripción (I-013, I-014).

Proporciona capacidades multimedia a OVAV:
  - image: analizar imágenes en todos los formatos
  - youtube: extraer transcripción y analizar contenido de YouTube

Dependencias opcionales:
  - yt-dlp: para extracción de YouTube (pip install yt-dlp)
  - Pillow: para metadatos de imágenes (pip install Pillow)

Uso:
  python3 tools/web/ovav_multimedia.py image <path>
  python3 tools/web/ovav_multimedia.py youtube <url>
"""

from __future__ import annotations

import base64
import json
import subprocess
import sys
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

REPO_ROOT = Path(__file__).resolve().parents[2]


def analyze_image(image_path: str) -> dict[str, Any]:
    """Analiza una imagen y retorna metadatos + base64 para el modelo.

    Soporta: PNG, JPEG, WebP, GIF, SVG, BMP, TIFF.

    Returns:
        dict con metadatos y datos base64 listos para enviar al modelo.
    """
    path = Path(image_path)
    if not path.exists():
        return {"error": f"File not found: {image_path}"}

    ext = path.suffix.lower()
    supported = {".png", ".jpg", ".jpeg", ".webp", ".gif", ".svg", ".bmp", ".tiff", ".tif"}

    if ext not in supported:
        return {"error": f"Unsupported format: {ext}. Supported: {sorted(supported)}"}

    result: dict[str, Any] = {
        "file": str(path),
        "format": ext.lstrip("."),
        "size_bytes": path.stat().st_size,
        "timestamp": datetime.now(UTC).isoformat(),
    }

    # Leer y codificar en base64
    try:
        with open(path, "rb") as f:
            image_data = f.read()
        result["base64"] = base64.b64encode(image_data).decode("utf-8")
        result["data_uri"] = f"data:image/{ext.lstrip('.')};base64,{result['base64'][:50]}..."
    except Exception as e:
        result["read_error"] = str(e)

    # Metadatos adicionales con Pillow
    try:
        from PIL import Image
        with Image.open(path) as img:
            result["dimensions"] = f"{img.width}x{img.height}"
            result["mode"] = img.mode
            if hasattr(img, "info"):
                result["metadata"] = {k: str(v)[:200] for k, v in img.info.items() if k != "icc_profile"}
    except ImportError:
        result["pillow_note"] = "Install Pillow for richer metadata: pip install Pillow"
    except Exception:
        pass

    return result


def extract_youtube_transcript(url: str) -> dict[str, Any]:
    """Extrae transcripción y metadatos de un video de YouTube.

    Usa yt-dlp para extraer subtítulos y metadatos.

    Returns:
        dict con título, descripción, transcripción, duración.
    """
    result: dict[str, Any] = {
        "url": url,
        "timestamp": datetime.now(UTC).isoformat(),
    }

    # Verificar que yt-dlp esté disponible
    ytdlp = _find_ytdlp()
    if not ytdlp:
        return {
            "error": "yt-dlp not found",
            "install_hint": "pip install yt-dlp",
            "url": url,
        }

    try:
        # Extraer metadatos
        meta_cmd = [ytdlp, "--dump-json", "--no-playlist", "--skip-download", url]
        meta_result = subprocess.run(meta_cmd, capture_output=True, text=True, timeout=30)

        if meta_result.returncode == 0 and meta_result.stdout.strip():
            meta = json.loads(meta_result.stdout)
            result["title"] = meta.get("title", "")
            result["duration_seconds"] = meta.get("duration", 0)
            result["description"] = (meta.get("description", "") or "")[:1000]
            result["channel"] = meta.get("uploader", "")
            result["view_count"] = meta.get("view_count", 0)
            result["upload_date"] = meta.get("upload_date", "")

        # Extraer subtítulos
        sub_cmd = [
            ytdlp, "--write-auto-sub", "--sub-lang", "en,es",
            "--skip-download", "--convert-subs", "srt",
            "-o", f"{REPO_ROOT}/tmp/yt_%(id)s", url,
        ]
        sub_result = subprocess.run(sub_cmd, capture_output=True, text=True, timeout=60)

        # Buscar el archivo de subtítulos generado
        tmp_dir = REPO_ROOT / "tmp"
        if tmp_dir.exists():
            for srt_file in tmp_dir.glob("yt_*.srt"):
                content = srt_file.read_text(errors="replace")
                result["transcript"] = _clean_srt(content)[:5000]
                result["transcript_length"] = len(result.get("transcript", ""))
                srt_file.unlink()  # Limpiar
                break

        if "transcript" not in result:
            # Intentar subtítulos manuales si no hay auto
            sub_cmd_manual = [
                ytdlp, "--write-sub", "--sub-lang", "en,es",
                "--skip-download", "-o", f"{REPO_ROOT}/tmp/yt_%(id)s", url,
            ]
            subprocess.run(sub_cmd_manual, capture_output=True, text=True, timeout=60)
            for srt_file in tmp_dir.glob("yt_*.srt") if tmp_dir.exists() else []:
                content = srt_file.read_text(errors="replace")
                result["transcript"] = _clean_srt(content)[:5000]
                result["transcript_length"] = len(result.get("transcript", ""))
                result["transcript_type"] = "manual"
                srt_file.unlink()
                break

        if "transcript" not in result:
            result["transcript_note"] = "No subtitles available for this video"

    except subprocess.TimeoutExpired:
        result["error"] = "YouTube extraction timed out (30s)"
    except Exception as e:
        result["error"] = str(e)

    return result


def _find_ytdlp() -> str | None:
    """Find yt-dlp executable."""
    for cmd in ["yt-dlp", "yt-dlp.exe"]:
        result = subprocess.run(["which", cmd], capture_output=True, text=True)
        if result.returncode == 0 and result.stdout.strip():
            return result.stdout.strip()
    # Check pip-installed
    try:
        result = subprocess.run(
            [sys.executable, "-m", "yt_dlp", "--version"],
            capture_output=True, text=True,
        )
        if result.returncode == 0:
            return f"{sys.executable} -m yt_dlp"
    except Exception:
        pass
    return None


def _clean_srt(srt_content: str) -> str:
    """Limpia formato SRT: elimina números de línea y timestamps."""
    import re
    # Eliminar números de secuencia y timestamps
    cleaned = re.sub(r'\d+\n\d{2}:\d{2}:\d{2}[.,]\d{3} --> \d{2}:\d{2}:\d{2}[.,]\d{3}\n', '', srt_content)
    cleaned = re.sub(r'<[^>]+>', '', cleaned)  # Eliminar tags HTML
    cleaned = re.sub(r'\n\s*\n', '\n', cleaned)  # Colapsar líneas vacías
    return cleaned.strip()


# ── CLI ─────────────────────────────────────────────────────────────────────

def main() -> int:
    import argparse
    ap = argparse.ArgumentParser(description="OVAV Multimedia Tools")
    sub = ap.add_subparsers(dest="command")

    img = sub.add_parser("image", help="Analizar imagen")
    img.add_argument("path", help="Ruta de la imagen")

    yt = sub.add_parser("youtube", help="Extraer transcripción de YouTube")
    yt.add_argument("url", help="URL del video")

    sub.add_parser("formats", help="Mostrar formatos soportados")

    args = ap.parse_args()

    if args.command == "image":
        result = analyze_image(args.path)
        if "error" in result:
            print(f"❌ {result['error']}")
        else:
            print(f"🖼️ {result['file']}")
            print(f"   Formato: {result['format']} | Tamaño: {result['size_bytes']:,} bytes")
            if "dimensions" in result:
                print(f"   Dimensiones: {result['dimensions']} | Modo: {result['mode']}")
            print(f"   Base64: {len(result.get('base64', '')):,} chars (listo para modelo)")

    elif args.command == "youtube":
        result = extract_youtube_transcript(args.url)
        if "error" in result:
            print(f"❌ {result['error']}")
            if "install_hint" in result:
                print(f"   💡 {result['install_hint']}")
        else:
            print(f"🎬 {result.get('title', 'Unknown')}")
            print(f"   Canal: {result.get('channel', '?')} | Duración: {result.get('duration_seconds', 0)}s")
            if "transcript" in result:
                print(f"   Transcripción: {result['transcript_length']} chars")
                print(f"   Preview: {result['transcript'][:200]}...")

    elif args.command == "formats":
        print("Formatos de imagen soportados:")
        for fmt in ["PNG", "JPEG", "WebP", "GIF", "SVG", "BMP", "TIFF"]:
            print(f"  ✅ {fmt}")

    else:
        ap.print_help()

    return 0


if __name__ == "__main__":
    sys.exit(main())
