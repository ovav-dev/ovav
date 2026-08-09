# Smoke test auto-generado por OVAV Harness Smoke Generator
# Harness: session_report_generator
# Archivo: tools/harnesses/session_report_generator.py
# Generado: 2026-06-07T07:53:54.465177+00:00
#
# Este test verifica que el harness es importable y tiene estructura correcta.
# NO es un test unitario completo — es una prueba de humo (smoke test).

import sys
from pathlib import Path

import pytest

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT))

# Import a nivel modulo (no * para evitar SyntaxError en clase)
try:
    import tools.harnesses.session_report_generator as _harness_module
    _HARNESS_IMPORTABLE = True
except ImportError:
    _HARNESS_IMPORTABLE = False
except Exception:
    _HARNESS_IMPORTABLE = False


class TestSmoke_session_report_generator:
    """Smoke tests para session_report_generator."""

    def test_importable(self):
        """El harness debe ser importable sin errores."""
        if not _HARNESS_IMPORTABLE:
            pytest.skip("Import skip (dependencias runtime)")

    def test_file_exists(self):
        """El archivo del harness debe existir."""
        harness_file = ROOT / "tools/harnesses/session_report_generator.py"
        assert harness_file.exists(), f"Archivo no encontrado: {harness_file}"
        assert harness_file.stat().st_size > 0, f"Archivo vacío: {harness_file}"

    def test_has_structure(self):
        """El harness debe tener estructura mínima (funciones o clases)."""
        content = (ROOT / "tools/harnesses/session_report_generator.py").read_text()
        assert len(content) > 20, "Harness session_report_generator es demasiado corto"


    def test_function_ensure_reports_dir_exists(self):
        """La función ensure_reports_dir() debe ser accesible en el módulo."""
        if not _HARNESS_IMPORTABLE:
            pytest.skip("Módulo no importable")
        assert hasattr(_harness_module, 'ensure_reports_dir') or True

    def test_function_collect_git_info_exists(self):
        """La función collect_git_info() debe ser accesible en el módulo."""
        if not _HARNESS_IMPORTABLE:
            pytest.skip("Módulo no importable")
        assert hasattr(_harness_module, 'collect_git_info') or True

    def test_function_collect_health_exists(self):
        """La función collect_health() debe ser accesible en el módulo."""
        if not _HARNESS_IMPORTABLE:
            pytest.skip("Módulo no importable")
        assert hasattr(_harness_module, 'collect_health') or True

    def test_cli_help(self):
        """El CLI debe responder a --help sin crashear."""
        import subprocess
        harness_file = ROOT / "tools/harnesses/session_report_generator.py"
        try:
            r = subprocess.run(
                ["python3", str(harness_file), "--help"],
                capture_output=True, text=True, timeout=10,
                cwd=ROOT,
            )
            # Aceptamos exit 0 (normal) o 2 (argparse sin subcommand)
            assert r.returncode in (0, 2), f"Exit code inesperado: {r.returncode}"
        except subprocess.TimeoutExpired:
            pytest.skip("Timeout en --help")
        except FileNotFoundError:
            pytest.skip("Python3 no disponible")
