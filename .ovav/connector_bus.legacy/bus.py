#!/usr/bin/env python3
"""
OVAV Connector Bus Engine v2.0
===============================
Lee .ovav/connector_bus/connectors/*.yaml (un archivo por tipo de slot)
y resuelve automáticamente todas las conexiones entre componentes
y superficies del sistema.

Principio: UN archivo por tipo para integrar/desconectar.
Este engine se encarga de:
  1. Cargar dinámicamente TODOS los componentes desde connectors/
  2. Resolver dependencias entre componentes
  3. Verificar health checks
  4. Proveer validadores → validate_all
  5. Proveer triggers → auto_triggers.yaml
  6. Proveer surface maps → surface_validator_map.yaml
  7. Hot reload sin reiniciar OVAV

Patrones profesionales aplicados:
  - VS Code: manifest por extensión (connectors/*.yaml)
  - Kubernetes: desired state reconciliation
  - Spring Boot: dependency declaration + auto-wiring

Uso:
  from .ovav.connector_bus.bus import ConnectorBus
  bus = ConnectorBus()
  validators = bus.get_validators()
  stats = bus.stats()

Ubicación: .ovav/connector_bus/bus.py
"""

from __future__ import annotations

import importlib
import sys
import time
from collections.abc import Callable
from pathlib import Path
from typing import Any

import yaml

REPO_ROOT = Path(__file__).resolve().parents[2]
if str(REPO_ROOT) not in sys.path:
    sys.path.insert(0, str(REPO_ROOT))

BUS_DIR = REPO_ROOT / ".ovav" / "connector_bus"
CONNECTORS_DIR = BUS_DIR / "connectors"
SLOTS_PATH = BUS_DIR / "slots.yaml"
REGISTRY_PATH = BUS_DIR / "registry.yaml"  # legacy — deprecated


class ConnectorBus:
    """Motor de conexiones OVAV v2. Lee connectors/*.yaml por tipo."""

    def __init__(self, connectors_dir: Path | None = None):
        self._connectors_dir = connectors_dir or CONNECTORS_DIR
        self._slots = self._load_slots()
        self._connectors: dict[str, dict[str, Any]] = {}
        self._cache: dict[str, Any] = {}
        self._load_time: float = 0.0
        self._load_all()

    # ── Carga ──────────────────────────────────────────────────────────

    def _load_slots(self) -> dict[str, Any]:
        if SLOTS_PATH.exists():
            return yaml.safe_load(SLOTS_PATH.read_text()) or {}
        return {}

    def _load_all(self) -> None:
        """Cargar todos los connectors/*.yaml."""
        start = time.time()
        self._connectors.clear()
        if self._connectors_dir.exists():
            for yaml_file in sorted(self._connectors_dir.glob("*.yaml")):
                try:
                    data = yaml.safe_load(yaml_file.read_text()) or {}
                    slot_type = data.get("slot_type", yaml_file.stem)
                    if "components" in data:
                        # Merge components from this file
                        existing = self._connectors.get(slot_type, {})
                        existing.update(data["components"])
                        self._connectors[slot_type] = existing
                except Exception as exc:
                    print(f"WARN: cannot load {yaml_file.name}: {exc}")
        self._load_time = time.time() - start
        self._cache.clear()

    def reload(self) -> None:
        """Hot reload: recargar connectors sin reiniciar."""
        self._load_all()
        print(f"  ↻ ConnectorBus reloaded ({len(self._connectors)} slot types, {self._load_time:.2f}s)")

    def get_load_time(self) -> float:
        return self._load_time

    # ── Validadores ────────────────────────────────────────────────────

    def get_validators(self) -> list[Callable[[Path], list[str]]]:
        """Retorna lista de funciones validate(root) para validate_all."""
        if "validators" in self._cache:
            return self._cache["validators"]

        validators: list[Callable[[Path], list[str]]] = []
        comps = self._connectors.get("validator", {})

        for name, cfg in sorted(comps.items()):
            # Circuit breaker check
            if cfg.get("breaker", {}).get("tripped"):
                print(f"WARN: validator '{name}' circuit breaker tripped — skipping")
                continue

            fn = self._import_function(cfg["module"], cfg.get("function", "validate"))
            if fn:
                # Preserve metadata for debugging
                fn.__ovav_name__ = name
                fn.__ovav_module__ = cfg["module"]
                validators.append(fn)
            else:
                print(f"WARN: validator '{name}' not found: {cfg['module']}.{cfg.get('function', 'validate')}")

        self._cache["validators"] = validators
        return validators

    def get_validator_entries(self) -> dict[str, Any]:
        """Retorna el diccionario crudo de entries de validadores (para debugging)."""
        return self._connectors.get("validator", {})

    # ── Triggers ───────────────────────────────────────────────────────

    def get_triggers(self) -> dict[str, list[dict[str, Any]]]:
        """Retorna {evento: [{type, name, module, function, tier}, ...]}."""
        if "triggers" in self._cache:
            return self._cache["triggers"]

        triggers: dict[str, list[dict[str, Any]]] = {}
        triggerable_types = ["validator", "harness", "config_watcher"]

        for slot_type in triggerable_types:
            for name, cfg in self._connectors.get(slot_type, {}).items():
                for trigger in cfg.get("triggers", []):
                    triggers.setdefault(trigger, []).append({
                        "type": slot_type,
                        "name": name,
                        "module": cfg.get("module", ""),
                        "function": cfg.get("function", "validate"),
                        "tier": cfg.get("tier", 10),
                        "manual_command": cfg.get("manual_command"),
                    })

        self._cache["triggers"] = triggers
        return triggers

    # ── Surface Map ────────────────────────────────────────────────────

    def get_surface_map(self) -> dict[str, list[dict[str, Any]]]:
        """Retorna {superficie: [{validator, tier, module}, ...]}."""
        if "surface_map" in self._cache:
            return self._cache["surface_map"]

        surface_map: dict[str, list[dict[str, Any]]] = {}

        for name, cfg in self._connectors.get("validator", {}).items():
            surface = cfg.get("surface")
            if surface:
                surface_map.setdefault(surface, []).append({
                    "validator": name,
                    "tier": cfg.get("tier", 10),
                    "module": cfg["module"],
                })

        self._cache["surface_map"] = surface_map
        return surface_map

    # ── Tools ──────────────────────────────────────────────────────────

    def get_tools(self) -> dict[str, dict[str, Any]]:
        if "tools" in self._cache:
            return self._cache["tools"]

        tools: dict[str, dict[str, Any]] = {}
        for name, cfg in self._connectors.get("tool", {}).items():
            alias = cfg.get("alias", name)
            tools[alias] = {
                "name": name,
                "command": cfg["command"],
                "category": cfg.get("category", "uncategorized"),
                "risk": cfg.get("risk", "low"),
                "labels": cfg.get("labels", []),
            }

        self._cache["tools"] = tools
        return tools

    # ── Skills ─────────────────────────────────────────────────────────

    def get_skills(self) -> dict[str, dict[str, Any]]:
        if "skills" in self._cache:
            return self._cache["skills"]
        self._cache["skills"] = self._connectors.get("skill", {})
        return self._cache["skills"]

    # ── Personnel ──────────────────────────────────────────────────────

    def get_personnel(self) -> dict[str, dict[str, Any]]:
        if "personnel" in self._cache:
            return self._cache["personnel"]
        self._cache["personnel"] = self._connectors.get("personnel", {})
        return self._cache["personnel"]

    def get_active_leads(self) -> list[str]:
        """Retorna nombres de leads activos."""
        personnel = self.get_personnel()
        return [
            name for name, cfg in personnel.items()
            if cfg.get("type") == "lead" and cfg.get("active")
        ]

    # ── Clients ────────────────────────────────────────────────────────

    def get_clients(self) -> dict[str, dict[str, Any]]:
        if "clients" in self._cache:
            return self._cache["clients"]
        self._cache["clients"] = self._connectors.get("client", {})
        return self._cache["clients"]

    # ── Dependency Check ───────────────────────────────────────────────

    def check_dependencies(self) -> list[str]:
        """Verificar que todos los componentes tengan sus dependencias resueltas.
        Retorna lista de issues encontrados."""
        issues: list[str] = []

        for slot_type, comps in self._connectors.items():
            for name, cfg in comps.items():
                deps = cfg.get("depends_on", [])
                for dep in deps:
                    dep_type, dep_name = dep.split(":", 1) if ":" in dep else (None, dep)
                    if dep_type and dep_name:
                        found = dep_name in self._connectors.get(dep_type, {})
                        if not found:
                            issues.append(f"{slot_type}.{name}: missing dependency {dep}")
                    elif not self._import_function(dep, "validate"):
                        issues.append(f"{slot_type}.{name}: dependency not importable: {dep}")

        return issues

    def check_health(self) -> dict[str, Any]:
        """Verificar health checks de componentes registrados.
        Retorna {componente: {status, message}}."""
        results: dict[str, Any] = {}

        for slot_type, comps in self._connectors.items():
            for name, cfg in comps.items():
                health = cfg.get("health", {})
                if not health:
                    continue

                check_type = health.get("check", "")
                key = f"{slot_type}.{name}"

                if check_type == "file_exists":
                    target = REPO_ROOT / health.get("target", "")
                    if target.exists():
                        results[key] = {"status": "healthy", "message": f"{target} exists"}
                    else:
                        results[key] = {"status": "degraded", "message": f"{target} not found"}

                elif check_type == "module_loads":
                    fn = self._import_function(cfg.get("module", ""), cfg.get("function", "validate"))
                    if fn:
                        results[key] = {"status": "healthy", "message": "module loaded"}
                    else:
                        results[key] = {"status": "degraded", "message": "module failed to load"}

        return results

    # ── Stats ──────────────────────────────────────────────────────────

    def stats(self) -> dict[str, int]:
        """Estadísticas del bus."""
        return {
            "slot_types": len(self._connectors),
            "validators": len(self._connectors.get("validator", {})),
            "harnesses": len(self._connectors.get("harness", {})),
            "tools": len(self._connectors.get("tool", {})),
            "adapters": len(self._connectors.get("adapter", {})),
            "plugins": len(self._connectors.get("plugin", {})),
            "skills": len(self._connectors.get("skill", {})),
            "clients": len(self._connectors.get("client", {})),
            "personnel": len(self._connectors.get("personnel", {})),
            "watchers": len(self._connectors.get("config_watcher", {})),
            "triggers_total": sum(len(v) for v in self.get_triggers().values()),
            "surfaces_total": len(self.get_surface_map()),
            "load_time_ms": int(self._load_time * 1000),
        }

    # ── Internals ──────────────────────────────────────────────────────

    def _import_function(self, module_path: str, function_name: str) -> Callable | None:
        try:
            mod = importlib.import_module(module_path)
            fn = getattr(mod, function_name, None)
            if fn is None:
                return None
            # Si la funcion es 'main', envolverla para compatibilidad con validate_all.
            # main() tipicamente imprime output y llama sys.exit(); capturamos stdout
            # y convertimos el resultado a list[str] para la interfaz estandar.
            if function_name == "main":
                return self._wrap_main(fn, module_path)
            return fn
        except Exception:
            return None

    def _wrap_main(self, main_fn: Callable, module_path: str) -> Callable[[Path], list[str]]:
        """Envuelve una funcion main() para que se comporte como validate(root) -> list[str].
        
        Captura stdout, previene sys.exit(), y convierte la salida a lineas.
        """
        import io as _io

        def wrapped_main(root: Path | None = None) -> list[str]:
            old_stdout = sys.stdout
            old_exit = sys.exit
            captured = _io.StringIO()
            try:
                sys.stdout = captured
                # Prevenir que main() llame a sys.exit()
                sys.exit = lambda code=0: None
                main_fn()
                output = captured.getvalue()
            except Exception as e:
                output = captured.getvalue()
                output += f"\nERROR: {module_path}.main() crashed: {e}"
            finally:
                sys.stdout = old_stdout
                sys.exit = old_exit

            lines = [l.strip() for l in output.split("\n") if l.strip()]
            if not lines:
                lines = [f"PASS {module_path}"]
            return lines

        return wrapped_main


# ── Singleton ────────────────────────────────────────────────────────
_bus_instance: ConnectorBus | None = None


def get_bus() -> ConnectorBus:
    global _bus_instance
    if _bus_instance is None:
        _bus_instance = ConnectorBus()
    return _bus_instance


# ── CLI ──────────────────────────────────────────────────────────────
if __name__ == "__main__":
    bus = ConnectorBus()
    s = bus.stats()
    print("OVAV Connector Bus v2.0")
    print(f"  Load time:    {s['load_time_ms']}ms")
    print(f"  Slot types:   {s['slot_types']}")
    print(f"  Validators:   {s['validators']}")
    print(f"  Harnesses:    {s['harnesses']}")
    print(f"  Tools:        {s['tools']}")
    print(f"  Adapters:     {s['adapters']}")
    print(f"  Plugins:      {s['plugins']}")
    print(f"  Skills:       {s['skills']}")
    print(f"  Clients:      {s['clients']}")
    print(f"  Personnel:    {s['personnel']}")
    print(f"  Watchers:     {s['watchers']}")
    print(f"  Triggers:     {s['triggers_total']} across {len(bus.get_triggers())} events")
    print(f"  Surfaces:     {s['surfaces_total']}")
    print(f"  Active leads: {', '.join(bus.get_active_leads())}")

    deps = bus.check_dependencies()
    if deps:
        print(f"\n  Dependency issues: {len(deps)}")
        for d in deps[:5]:
            print(f"    - {d}")

    health = bus.check_health()
    degraded = {k: v for k, v in health.items() if v["status"] != "healthy"}
    if degraded:
        print(f"\n  Health issues: {len(degraded)}")
        for k, v in degraded.items():
            print(f"    - {k}: {v['message']}")
    else:
        print(f"\n  Health: all {len(health)} checks healthy")
