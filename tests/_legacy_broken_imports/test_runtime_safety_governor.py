"""OVAV Runtime Safety Governor — validación acotada (10 tests).

No usa validate_all.py. Cada test verifica un guard específico.
Ejecutar: python3 -m pytest tests/test_runtime_safety_governor.py -v
"""

from __future__ import annotations

import os
import time

import pytest

from tools.agent_runtime.runtime_safety_governor import (
    ConcurrencyGuard,
    ExecutionLedger,
    FanoutGuard,
    LoopGuard,
    PIDLock,
    ResourcePreflight,
    RuntimeSafetyGovernor,
    _kill_process_tree,
    activate_repair_mode,
    deactivate_repair_mode,
    in_repair_mode,
)

# ══════════════════════════════════════════════════════════════════════════════
# TEST 1: PID Lock — adquisición y liberación
# ══════════════════════════════════════════════════════════════════════════════

def test_pid_lock_acquire_release():
    """PID lock se adquiere con el PID actual y se libera correctamente."""
    lock = PIDLock("test_family_1")
    assert lock.acquire() is True
    assert lock.is_acquired is True

    # Verificar que el archivo contiene nuestro PID
    assert lock._path.exists()
    assert int(lock._path.read_text().strip()) == os.getpid()

    lock.release()
    assert lock.is_acquired is False
    assert not lock._path.exists()


# ══════════════════════════════════════════════════════════════════════════════
# TEST 2: PID Lock — bloqueo por lock activo
# ══════════════════════════════════════════════════════════════════════════════

def test_pid_lock_blocks_concurrent():
    """Dos locks de la misma familia: el segundo es bloqueado."""
    lock1 = PIDLock("test_family_2")
    lock2 = PIDLock("test_family_2")

    assert lock1.acquire() is True
    assert lock2.acquire() is False  # Mismo PID, mismo archivo → bloqueado

    lock1.release()
    assert lock2.acquire() is True  # Ahora sí
    lock2.release()


# ══════════════════════════════════════════════════════════════════════════════
# TEST 3: Concurrency Guard — límite global
# ══════════════════════════════════════════════════════════════════════════════

def test_concurrency_guard_global_limit():
    """No se pueden exceder max_concurrency global."""
    guard = ConcurrencyGuard(max_global=2, max_per_family=2)

    lock1 = PIDLock("family_a")
    lock2 = PIDLock("family_b")
    lock1.acquire()
    lock2.acquire()

    # 2 locks activos → límite global alcanzado
    allowed, reason = guard.allow("family_c")
    assert allowed is False
    assert "max_global_concurrency" in reason

    lock1.release()
    lock2.release()


# ══════════════════════════════════════════════════════════════════════════════
# TEST 4: Concurrency Guard — límite por familia
# ══════════════════════════════════════════════════════════════════════════════

def test_concurrency_guard_family_limit():
    """No se pueden exceder max_concurrency por familia."""
    guard = ConcurrencyGuard(max_global=10, max_per_family=1)

    lock1 = PIDLock("validator")
    lock1.acquire()

    allowed, reason = guard.allow("validator")
    assert allowed is False
    assert "max_family_concurrency" in reason
    assert "validator" in reason

    lock1.release()


# ══════════════════════════════════════════════════════════════════════════════
# TEST 5: Execution Ledger — registrar y consultar
# ══════════════════════════════════════════════════════════════════════════════

def test_execution_ledger_record_and_query():
    """El ledger registra entradas y permite consultar por familia."""
    ledger = ExecutionLedger()

    started = ledger.record_start("validator", "test_check", ttl=30)
    assert started
    time.sleep(0.01)
    ledger.record_end(started, "completed", duration_ms=10.0)

    recent = ledger.recent_by_family("validator", minutes=5)
    assert len(recent) >= 1
    assert recent[-1]["task_name"] == "test_check"
    assert recent[-1]["status"] == "completed"


# ══════════════════════════════════════════════════════════════════════════════
# TEST 6: Fanout Guard — bloquea después de N ejecuciones
# ══════════════════════════════════════════════════════════════════════════════

def test_fanout_guard_blocks_after_limit():
    """El fanout guard bloquea cuando se excede el límite por hora."""
    ledger = ExecutionLedger()
    guard = FanoutGuard(ledger, limits={"snapshot": 3})

    # Registrar 3 snapshots completados
    for i in range(3):
        started = ledger.record_start("snapshot", f"snap_{i}", ttl=1)
        ledger.record_end(started, "completed", duration_ms=1.0)

    # La 4ta debe ser bloqueada
    allowed, reason = guard.allow("snapshot")
    assert allowed is False
    assert "fanout_guard" in reason
    assert "/3" in reason  # Rate limit denominator present


# ══════════════════════════════════════════════════════════════════════════════
# TEST 7: Loop Guard — detecta repetición anormal
# ══════════════════════════════════════════════════════════════════════════════

def test_loop_guard_detects_repetition():
    """El loop guard detecta >3 ejecuciones de la misma tarea en 5min."""
    ledger = ExecutionLedger()
    guard = LoopGuard(ledger, max_repeats=3, window_minutes=5)

    # Registrar 3 ejecuciones de la misma tarea
    for i in range(3):
        started = ledger.record_start("health_check", "system_health", ttl=10)
        ledger.record_end(started, "completed", duration_ms=1.0)

    # La 4ta debe ser bloqueada
    allowed, reason = guard.allow("health_check", "system_health")
    assert allowed is False
    assert "loop_guard" in reason


# ══════════════════════════════════════════════════════════════════════════════
# TEST 8: Resource Preflight — bloquea con RAM baja simulada
# ══════════════════════════════════════════════════════════════════════════════

def test_resource_preflight_blocks_low_ram():
    """Resource preflight bloquea cuando RAM está bajo el mínimo."""
    preflight = ResourcePreflight(min_ram_mb=999999)  # Imposible de alcanzar

    ok, reason, metrics = preflight.check()
    assert ok is False
    assert "RAM baja" in reason
    assert metrics["ram_mb"] < 999999


# ══════════════════════════════════════════════════════════════════════════════
# TEST 9: Safe Repair Mode — activación y desactivación
# ══════════════════════════════════════════════════════════════════════════════

def test_repair_mode_activate_deactivate():
    """El repair mode se activa, detecta y desactiva correctamente."""
    # Asegurar que empieza desactivado
    deactivate_repair_mode()
    assert in_repair_mode() is False

    # Activar
    assert activate_repair_mode() is True
    assert in_repair_mode() is True

    # Verificar que el governor permite todo en repair mode
    gov = RuntimeSafetyGovernor()
    with gov.guard("snapshot", "test_repair", ttl=1) as g:
        assert g.allowed is True  # Repair mode → todo permitido

    # Desactivar
    assert deactivate_repair_mode() is True
    assert in_repair_mode() is False


# ══════════════════════════════════════════════════════════════════════════════
# TEST 10: Process Tree Kill — función centralizada
# ══════════════════════════════════════════════════════════════════════════════

def test_process_tree_kill_importable():
    """La función _kill_process_tree es importable y ejecutable."""
    # Verificar que está disponible y no lanza excepción con PID inválido
    killed = _kill_process_tree(99999)  # PID que no existe
    assert killed >= 0  # Debe retornar 0 (no encontró el proceso)


# ══════════════════════════════════════════════════════════════════════════════
# FIXTURE: limpiar locks después de cada test
# ══════════════════════════════════════════════════════════════════════════════

@pytest.fixture(autouse=True)
def _cleanup_locks():
    """Limpia locks y desactiva repair mode después de cada test."""
    yield
    # Clean up test locks
    from tools.agent_runtime.runtime_safety_governor import LOCKS_DIR
    for lock_file in LOCKS_DIR.glob("test_*.pid"):
        try:
            pid = int(lock_file.read_text().strip())
            if pid == os.getpid():
                lock_file.unlink()
        except (ValueError, OSError):
            lock_file.unlink(missing_ok=True)
    # Desactivar repair mode
    deactivate_repair_mode()
