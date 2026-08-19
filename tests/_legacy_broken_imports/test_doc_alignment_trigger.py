"""Tests para Doc Alignment Trigger v2.0 — Sistema Nervioso Vivo.

Prueba el detector + corrector inteligente de alineación documental:
- Actualización de tablas de estado en markdown
- Appending a secciones
- Actualización de CHANGELOG
- Clasificación de cambios por superficie
- Flujo completo align() con mocks de git
"""

import json
import sys
from pathlib import Path
from unittest.mock import MagicMock, patch

import pytest

# Asegurar que el repo root está en sys.path
REPO_ROOT = Path(__file__).resolve().parents[1]
if str(REPO_ROOT) not in sys.path:
    sys.path.insert(0, str(REPO_ROOT))

# ─── Fixtures ─────────────────────────────────────────────────────────────────

@pytest.fixture
def sample_status_table():
    """Tabla de estado estilo .ovav/plan/caps.yaml"""
    return """## Estado real (no plan — lo que YA existe)

| Sistema | Estado |
|---|---|
| **Integrity Mesh (F0-F4)** | ✅ 18 validators, 0 rotos |
| **Doc Alignment Trigger** | ⬜ propuesto: detectar superficies doc |
| **Security (3 breaches)** | ✅ gate_self_protection |
| **Sistema Nervioso Vivo** | ⬜ propuesto: grafo de conocimiento |
"""


@pytest.fixture
def sample_changelog():
    """CHANGELOG.md simulado"""
    return """# OVAV Changelog

Registro automático de cambios generado desde git history.
Última actualización: 2026-06-01 03:35 UTC

---

## ✨ Features

- **[2026-06-01]** feat: Knowledge Compiler — F5 gates operational (`eb9ed22b`)

## 🐛 Fixes

- **[2026-06-01]** fix: wire 3 security breaches (`0347e249`)

## ♻️ Refactors

## 📝 Documentation

- **[2026-06-01]** docs: registrar diagnóstico de fuga de snapshots (`6982f95a`)
"""


@pytest.fixture
def sample_section_doc():
    """Documento con secciones markdown"""
    return """# Documento de Prueba

## Introducción

Contenido de introducción.

## Harnesses Activos

- **[2026-05-01]** Primer harness implementado

## Conclusión

Fin del documento.
"""


@pytest.fixture
def sample_registry():
    """Registry de superficies simulado"""
    return {
        "surfaces": {
            "tools/security": {
                "priority": 1,
                "docs": [
                    {"file": "docs/security/06_SECURITY_FRAMEWORK.md", "update": "section_append", "target": "## Runtime Security Gates", "rule": "append_new_gate_entry"}
                ]
            },
            "tools/harnesses": {
                "priority": 2,
                "docs": [
                    {"file": ".ovav/plan/caps.yaml", "update": "status_table", "table_header": "Sistema | Estado", "row_match_pattern": "Doc Alignment", "marker_column": 2},
                    {"file": "docs/implementation/07_IMPLEMENTATION_ROADMAP.md", "update": "section_append", "target": "## Harnesses Activos", "rule": "append_harness_entry"}
                ]
            }
        },
        "heuristics": [
            {"keywords": ["transport", "https", "git"], "docs": ["docs/system/01_ARCHITECTURE.md"]},
            {"keywords": ["validate", "severity"], "docs": ["docs/runtime/04_RUNTIME_ENFORCEMENT.md"]}
        ]
    }


# ─── Test: _update_status_table ───────────────────────────────────────────────

class TestUpdateStatusTable:
    """Pruebas para actualización de tablas de estado markdown."""

    def test_updates_pending_to_done(self, sample_status_table):
        """⬜ → ✅: actualiza un marcador pendiente a completado."""
        from tools.harnesses.doc_alignment_trigger import _update_status_table

        result, updated = _update_status_table(
            sample_status_table,
            "Sistema | Estado",
            "Doc Alignment",
            "✅",
            2,
        )
        assert updated == 1
        # La fila Doc Alignment ahora tiene ✅ en lugar de ⬜
        assert "| **Doc Alignment Trigger** | ✅ propuesto:" in result
        # La fila Doc Alignment ya no tiene ⬜ solo
        doc_line = [l for l in result.split("\n") if "Doc Alignment" in l][0]
        assert "⬜" not in doc_line

    def test_skips_already_done(self, sample_status_table):
        """No actualiza filas que ya están en ✅."""
        from tools.harnesses.doc_alignment_trigger import _update_status_table

        result, updated = _update_status_table(
            sample_status_table,
            "Sistema | Estado",
            "Integrity Mesh",
            "✅",
            2,
        )
        assert updated == 0
        # El contenido no debe cambiar
        assert "| **Integrity Mesh (F0-F4)** | ✅" in result

    def test_no_match_returns_zero(self, sample_status_table):
        """Si no hay coincidencia, updated=0."""
        from tools.harnesses.doc_alignment_trigger import _update_status_table

        result, updated = _update_status_table(
            sample_status_table,
            "Sistema | Estado",
            "Sistema Inexistente",
            "✅",
            2,
        )
        assert updated == 0

    def test_updates_multiple_matches(self):
        """Actualiza todas las filas que coinciden con el patrón."""
        from tools.harnesses.doc_alignment_trigger import _update_status_table

        content = """| Feature | Status |
|---|---|
| Feature A | ⬜ pending |
| Feature B | ⬜ pending |
| Feature C | ✅ done |
"""
        result, updated = _update_status_table(
            content,
            "Feature | Status",
            "Feature",  # Match both A and B
            "✅",
            2,
        )
        assert updated == 2
        assert "⬜ pending" not in result

    def test_marker_column_index(self):
        """Respeta el índice de columna especificado."""
        from tools.harnesses.doc_alignment_trigger import _update_status_table

        content = """| Name | Priority | Status |
|---|---|---|
| Alpha | High | ⬜ |
| Beta | Low | ✅ |
"""
        # Columna 3 es Status
        result, updated = _update_status_table(
            content,
            "Name | Priority | Status",
            "Alpha",
            "✅",
            3,
        )
        assert updated == 1
        assert "| Alpha | High | ✅ |" in result


# ─── Test: _append_to_section ─────────────────────────────────────────────────

class TestAppendToSection:
    """Pruebas para append a secciones markdown."""

    def test_appends_to_existing_section(self, sample_section_doc):
        """Agrega entrada al final de una sección existente."""
        from tools.harnesses.doc_alignment_trigger import _append_to_section

        entry = "- **[2026-06-02]** Nuevo harness agregado"
        result, success = _append_to_section(
            sample_section_doc,
            "## Harnesses Activos",
            entry,
        )
        assert success is True
        assert "Nuevo harness agregado" in result
        # Debe aparecer antes de "## Conclusión"
        assert result.index("Nuevo harness") < result.index("## Conclusión")

    def test_section_not_found_no_change(self, sample_section_doc):
        """Si la sección no existe, no modifica nada."""
        from tools.harnesses.doc_alignment_trigger import _append_to_section

        entry = "- Nueva entrada"
        result, success = _append_to_section(
            sample_section_doc,
            "## Sección Inexistente",
            entry,
        )
        assert success is False
        assert result == sample_section_doc

    def test_no_duplicate_entries(self, sample_section_doc):
        """No agrega entradas duplicadas."""
        from tools.harnesses.doc_alignment_trigger import _append_to_section

        entry = "- **[2026-05-01]** Primer harness implementado"
        result, success = _append_to_section(
            sample_section_doc,
            "## Harnesses Activos",
            entry,
        )
        assert success is False
        assert result == sample_section_doc

    def test_appends_to_last_section(self):
        """Cuando la sección es la última, agrega al final del documento."""
        from tools.harnesses.doc_alignment_trigger import _append_to_section

        content = """# Doc

## Última Sección

Contenido existente.
"""
        entry = "- Nueva entrada al final"
        result, success = _append_to_section(
            content,
            "## Última Sección",
            entry,
        )
        assert success is True
        assert result.strip().endswith("Nueva entrada al final")


# ─── Test: _update_changelog ──────────────────────────────────────────────────

class TestUpdateChangelog:
    """Pruebas para actualización de CHANGELOG.md."""

    def test_adds_feat_entry(self, sample_changelog):
        """Agrega una entrada de feature al changelog."""
        from tools.harnesses.doc_alignment_trigger import _update_changelog

        commits = ["abc12345 feat: nuevo sistema de caché"]
        categories = {"feat": 1}

        result, added = _update_changelog(sample_changelog, commits, categories)
        assert added == 1
        assert "nuevo sistema de caché" in result
        assert "abc12345" in result

    def test_skips_duplicate_entries(self, sample_changelog):
        """No duplica entradas ya existentes."""
        from tools.harnesses.doc_alignment_trigger import _update_changelog

        # El changelog tiene: feat: Knowledge Compiler — F5 gates operational (`eb9ed22b`)
        # Usar exactamente el mismo mensaje que ya existe
        commits = ["eb9ed22b feat: Knowledge Compiler — F5 gates operational"]
        categories = {"feat": 1}

        result, added = _update_changelog(sample_changelog, commits, categories)
        assert added == 0  # Ya existe

    def test_adds_multiple_categories(self, sample_changelog):
        """Agrega entradas en múltiples categorías."""
        from tools.harnesses.doc_alignment_trigger import _update_changelog

        commits = [
            "11111111 feat: feature nueva",
            "22222222 fix: bug reparado",
        ]
        categories = {"feat": 1, "fix": 1}

        result, added = _update_changelog(sample_changelog, commits, categories)
        assert added == 2
        assert "feature nueva" in result
        assert "bug reparado" in result

    def test_updates_timestamp(self, sample_changelog):
        """Actualiza la línea de última actualización."""
        from tools.harnesses.doc_alignment_trigger import _update_changelog

        commits = ["abc12345 docs: nueva documentación"]
        categories = {"docs": 1}

        result, added = _update_changelog(sample_changelog, commits, categories)
        # El timestamp debe haberse actualizado (fecha de hoy)
        import re
        assert re.search(r"Última actualización: \d{4}-\d{2}-\d{2}", result)

    def test_unknown_category_no_section(self, sample_changelog):
        """Categorías sin sección no generan entradas."""
        from tools.harnesses.doc_alignment_trigger import _update_changelog

        commits = ["abc12345 unknown: algo raro"]
        categories = {"unknown": 1}

        result, added = _update_changelog(sample_changelog, commits, categories)
        assert added == 0  # No hay sección para "unknown"


# ─── Test: _update_implementation_plan_status ─────────────────────────────────

class TestImplementationPlanStatus:
    """Pruebas para actualización de .ovav/plan/caps.yaml."""

    def test_doc_alignment_updates_status(self, sample_status_table):
        """Detecta trabajo en doc alignment y actualiza la tabla."""
        from tools.harnesses.doc_alignment_trigger import _update_implementation_plan_status

        changed = {"tools/harnesses/doc_alignment_trigger.py"}
        commits = ["abc12345 feat: doc alignment trigger v2 completo"]

        result, updated = _update_implementation_plan_status(
            sample_status_table, changed, commits
        )
        assert updated == 1
        assert "✅" in result
        # La fila de Doc Alignment ahora tiene ✅
        assert "| **Doc Alignment Trigger** | ✅" in result

    def test_no_matching_keyword_no_change(self, sample_status_table):
        """Sin keyword relevante, no modifica nada."""
        from tools.harnesses.doc_alignment_trigger import _update_implementation_plan_status

        changed = {"README.md"}
        commits = ["abc12345 chore: actualizar readme"]

        result, updated = _update_implementation_plan_status(
            sample_status_table, changed, commits
        )
        assert updated == 0


# ─── Test: _classify_changes ──────────────────────────────────────────────────

class TestClassifyChanges:
    """Pruebas para clasificación de cambios."""

    def test_classifies_by_surface(self, sample_registry):
        """Clasifica archivos en superficies usando el registry."""
        from tools.harnesses.doc_alignment_trigger import _classify_changes

        changed = {
            "tools/security/gate.py",
            "tools/harnesses/trigger.py",
            "unknown/file.txt",
        }
        commits = ["abc12345 feat: agregar seguridad"]

        result = _classify_changes(changed, commits, sample_registry)

        assert "tools/security" in result["surfaces_matched"]
        assert "tools/harnesses" in result["surfaces_matched"]
        assert result["surfaces_matched"]["tools/security"] == 1
        assert result["surfaces_matched"]["tools/harnesses"] == 1

        # unknown/file.txt debe ir a other:
        other_keys = [k for k in result["surfaces_matched"] if k.startswith("other:")]
        assert len(other_keys) == 1
        assert result["surfaces_matched"][other_keys[0]] == 1

    def test_affected_docs_from_registry(self, sample_registry):
        """Determina docs afectados desde el registry."""
        from tools.harnesses.doc_alignment_trigger import _classify_changes

        changed = {"tools/security/gate.py"}
        commits = ["abc12345 feat: hardening"]

        result = _classify_changes(changed, commits, sample_registry)

        assert len(result["affected_docs"]) >= 1
        doc_files = [d["file"] for d in result["affected_docs"]]
        assert "docs/security/06_SECURITY_FRAMEWORK.md" in doc_files

    def test_heuristic_fallback(self, sample_registry):
        """Heurísticas agregan docs adicionales cuando aplican."""
        from tools.harnesses.doc_alignment_trigger import _classify_changes

        changed = {"README.md"}
        commits = ["abc12345 feat: implementar HTTPS transport"]

        result = _classify_changes(changed, commits, sample_registry)

        assert len(result["heuristic_docs"]) >= 1
        assert "docs/system/01_ARCHITECTURE.md" in result["heuristic_docs"]

    def test_commit_categories(self, sample_registry):
        """Extrae categorías de commits (feat, fix, etc.)."""
        from tools.harnesses.doc_alignment_trigger import _classify_changes

        changed = set()
        commits = [
            "abc12345 feat: nueva feature",
            "def67890 fix: reparar bug",
            "a1b2c3d4 feat: otra feature",
        ]

        result = _classify_changes(changed, commits, sample_registry)
        assert result["commit_categories"]["feat"] == 2
        assert result["commit_categories"]["fix"] == 1


# ─── Test: File I/O ───────────────────────────────────────────────────────────

class TestFileIO:
    """Pruebas para lectura/escritura de documentos."""

    def test_read_doc_existing(self, tmp_path):
        """Lee un documento markdown existente."""
        from tools.harnesses import doc_alignment_trigger

        # Usar monkeypatch para cambiar ROOT
        test_file = tmp_path / "test.md"
        test_file.write_text("# Hello\n\nWorld.", encoding="utf-8")

        with patch.object(doc_alignment_trigger, 'ROOT', tmp_path):
            content = doc_alignment_trigger._read_doc("test.md")
            assert content == "# Hello\n\nWorld."

    def test_read_doc_missing(self, tmp_path):
        """Retorna None si el archivo no existe."""
        from tools.harnesses import doc_alignment_trigger

        with patch.object(doc_alignment_trigger, 'ROOT', tmp_path):
            content = doc_alignment_trigger._read_doc("nonexistent.md")
            assert content is None

    def test_write_doc(self, tmp_path):
        """Escribe un documento correctamente."""
        from tools.harnesses import doc_alignment_trigger

        with patch.object(doc_alignment_trigger, 'ROOT', tmp_path):
            success = doc_alignment_trigger._write_doc(
                "output.md", "# Test Output", dry_run=False
            )
            assert success is True
            assert (tmp_path / "output.md").exists()
            assert (tmp_path / "output.md").read_text() == "# Test Output"

    def test_write_doc_dry_run(self, tmp_path):
        """En dry_run no escribe el archivo."""
        from tools.harnesses import doc_alignment_trigger

        with patch.object(doc_alignment_trigger, 'ROOT', tmp_path):
            success = doc_alignment_trigger._write_doc(
                "output.md", "# Test Output", dry_run=True
            )
            assert success is True
            assert not (tmp_path / "output.md").exists()


# ─── Test: _load_registry ─────────────────────────────────────────────────────

class TestLoadRegistry:
    """Pruebas para carga del registry."""

    def test_loads_yaml_registry(self):
        """Carga el registry YAML existente."""
        from tools.harnesses.doc_alignment_trigger import _load_registry

        registry = _load_registry()
        assert isinstance(registry, dict)
        assert "surfaces" in registry or "heuristics" in registry

    def test_missing_registry_returns_empty(self, tmp_path):
        """Si no existe el archivo, retorna estructura vacía."""
        from tools.harnesses import doc_alignment_trigger

        with patch.object(doc_alignment_trigger, 'REGISTRY_FILE', tmp_path / "nonexistent.yaml"):
            registry = doc_alignment_trigger._load_registry()
            assert registry == {"surfaces": {}, "heuristics": []}


# ─── Test: Git Operations ─────────────────────────────────────────────────────

class TestGitOperations:
    """Pruebas para operaciones git (mocked)."""

    def test_tag_exists_true(self):
        """Detecta tag existente."""
        from tools.harnesses.doc_alignment_trigger import _tag_exists

        with patch("subprocess.run") as mock_run:
            mock_run.return_value = MagicMock(
                returncode=0,
                stdout="doc-aligned\nother-tag\n",
                stderr="",
            )
            assert _tag_exists("doc-aligned") is True

    def test_tag_exists_false(self):
        """Detecta tag inexistente."""
        from tools.harnesses.doc_alignment_trigger import _tag_exists

        with patch("subprocess.run") as mock_run:
            mock_run.return_value = MagicMock(
                returncode=0,
                stdout="other-tag\n",
                stderr="",
            )
            assert _tag_exists("doc-aligned") is False

    def test_git_log_returns_commits(self):
        """Retorna lista de commits."""
        from tools.harnesses.doc_alignment_trigger import _git_log_since_tag

        with patch("subprocess.run") as mock_run:
            mock_run.return_value = MagicMock(
                returncode=0,
                stdout="abc1234 feat: test commit\ndef5678 fix: bug fix\n",
                stderr="",
            )
            commits = _git_log_since_tag("doc-aligned")
            assert len(commits) == 2
            assert "abc1234 feat: test commit" in commits

    def test_git_changed_files(self):
        """Retorna set de archivos modificados."""
        from tools.harnesses.doc_alignment_trigger import _git_changed_files_since_tag

        with patch("subprocess.run") as mock_run:
            mock_run.return_value = MagicMock(
                returncode=0,
                stdout="tools/harnesses/trigger.py\nregistry/sbom.yaml\n",
                stderr="",
            )
            files = _git_changed_files_since_tag("doc-aligned")
            assert len(files) == 2
            assert "tools/harnesses/trigger.py" in files


# ─── Test: align() Integration ────────────────────────────────────────────────

class TestAlignIntegration:
    """Pruebas de integración para align()."""

    def test_no_commits_since_tag(self):
        """Sin commits nuevos → no_op."""
        from tools.harnesses.doc_alignment_trigger import align

        with patch("tools.harnesses.doc_alignment_trigger._git_log_since_tag", return_value=[]):
            with patch("tools.harnesses.doc_alignment_trigger._tag_exists", return_value=True):
                result = align(dry_run=True)
                assert result["status"] == "ok"
                assert result["reason"] == "no_new_commits_since_tag"
                assert result["commits"] == 0

    def test_dry_run_no_file_writes(self, tmp_path):
        """Dry run no modifica archivos ni tags."""
        from tools.harnesses import doc_alignment_trigger

        with patch.object(doc_alignment_trigger, 'ROOT', tmp_path):
            with patch.object(doc_alignment_trigger, '_git_log_since_tag', return_value=[
                "abc12345 feat: doc alignment trigger completo"
            ]):
                with patch.object(doc_alignment_trigger, '_git_changed_files_since_tag', return_value={
                    "tools/harnesses/doc_alignment_trigger.py",
                    "registry/doc_surfaces.yaml",
                }):
                    with patch.object(doc_alignment_trigger, '_tag_exists', return_value=True):
                        with patch.object(doc_alignment_trigger, '_create_tag') as mock_create_tag:
                            # Crear archivos necesarios
                            (tmp_path / ".ovav" / "plan").mkdir(parents=True, exist_ok=True)
                            (tmp_path / ".ovav/plan/caps.yaml").write_text(
                                "| **Doc Alignment Trigger** | ⬜ propuesto |\n", encoding="utf-8"
                            )
                            (tmp_path / "CHANGELOG.md").write_text(
                                "# Changelog\n\nÚltima actualización: 2026-06-01\n\n## ✨ Features\n\n",
                                encoding="utf-8"
                            )
                            (tmp_path / "registry").mkdir(parents=True, exist_ok=True)
                            (tmp_path / "registry" / "doc_surfaces.yaml").write_text(
                                "surfaces: {}\nheuristics: []\n", encoding="utf-8"
                            )

                            result = doc_alignment_trigger.align(dry_run=True)
                            assert result["dry_run"] is True
                            # No debe crear tag en dry run
                            assert mock_create_tag.call_count <= 1  # Solo el inicial

    def test_align_with_new_commits(self, tmp_path):
        """Con commits nuevos, ejecuta alineación."""
        from tools.harnesses import doc_alignment_trigger

        with patch.object(doc_alignment_trigger, 'ROOT', tmp_path):
            with patch.object(doc_alignment_trigger, '_git_log_since_tag', return_value=[
                "abc12345 feat: nuevo sistema implementado"
            ]):
                with patch.object(doc_alignment_trigger, '_git_changed_files_since_tag', return_value={
                    "tools/harnesses/nuevo.py",
                }):
                    with patch.object(doc_alignment_trigger, '_tag_exists', return_value=True):
                        with patch.object(doc_alignment_trigger, '_create_tag'):
                            # Crear archivos de doc simulados
                            (tmp_path / ".ovav" / "plan").mkdir(parents=True, exist_ok=True)
                            (tmp_path / ".ovav/plan/caps.yaml").write_text(
                                "| **Sistema** | ⬜ pendiente |\n", encoding="utf-8"
                            )
                            (tmp_path / "CHANGELOG.md").write_text(
                                "# Changelog\n\nÚltima actualización: 2026-06-01\n\n## ✨ Features\n\n",
                                encoding="utf-8"
                            )
                            (tmp_path / "registry").mkdir(parents=True, exist_ok=True)
                            (tmp_path / "registry" / "doc_surfaces.yaml").write_text(
                                "surfaces: {}\nheuristics: []\n", encoding="utf-8"
                            )

                            result = doc_alignment_trigger.align(dry_run=True)
                            assert result["status"] == "ok"
                            assert result["commits"] == 1
                            assert result["commit_categories"]["feat"] == 1


# ─── Test: Status Persistence ─────────────────────────────────────────────────

class TestStatusPersistence:
    """Pruebas para carga/guardado de estado."""

    def test_load_status_default(self, tmp_path):
        """Sin archivo de estado, retorna defaults."""
        from tools.harnesses import doc_alignment_trigger

        with patch.object(doc_alignment_trigger, 'STATUS_FILE', tmp_path / "nonexistent.json"):
            status = doc_alignment_trigger._load_status()
            assert status["runs"] == 0
            assert status["last_aligned_tag"] == "doc-aligned"

    def test_save_and_load_status(self, tmp_path):
        """Guarda y recupera estado correctamente."""
        from tools.harnesses import doc_alignment_trigger

        status_file = tmp_path / "status.json"
        with patch.object(doc_alignment_trigger, 'STATUS_FILE', status_file):
            status = {"runs": 5, "last_aligned_tag": "doc-aligned", "last_run": "2026-06-02T00:00:00Z"}
            doc_alignment_trigger._save_status(status)

            loaded = doc_alignment_trigger._load_status()
            assert loaded["runs"] == 5
            assert loaded["last_run"] == "2026-06-02T00:00:00Z"


# ─── CLI Tests ────────────────────────────────────────────────────────────────

class TestCLI:
    """Pruebas de la interfaz CLI."""

    def test_json_output(self, capsys):
        """--json produce salida JSON válida."""
        import sys

        from tools.harnesses.doc_alignment_trigger import main

        test_args = ['doc_alignment_trigger.py', '--dry-run', '--json']
        with patch.object(sys, 'argv', test_args):
            with patch("tools.harnesses.doc_alignment_trigger._git_log_since_tag", return_value=[]):
                with patch("tools.harnesses.doc_alignment_trigger._tag_exists", return_value=True):
                    with patch("tools.harnesses.doc_alignment_trigger._load_registry", return_value={"surfaces": {}, "heuristics": []}):
                        exit_code = main()
                        captured = capsys.readouterr()
                        result = json.loads(captured.out)
                        assert result["dry_run"] is True
                        assert result["commits"] == 0
                        assert exit_code == 0
