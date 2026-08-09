/* ═══════════ OVAV Hooks ═══════════ */
import { useState, useEffect, useCallback, useRef } from 'react';
import type { StatusResponse, TaskStatus, ToastType } from '../types';
import { api } from '../services/api';
import { sse } from '../services/sse';

/** Auto-refreshing status hook. Refreshes every `interval` ms. */
export function useStatus(interval = 15000) {
  const [status, setStatus] = useState<StatusResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const refresh = useCallback(async () => {
    try {
      const data = await api.getStatus();
      setStatus(data);
      setError(null);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
    const timer = setInterval(refresh, interval);
    return () => clearInterval(timer);
  }, [refresh, interval]);

  return { status, error, loading, refresh };
}

/** Validator task polling hook. */
export function useValidatorPoll() {
  const [running, setRunning] = useState(false);
  const [taskId, setTaskId] = useState<string | null>(null);
  const [taskResult, setTaskResult] = useState<TaskStatus | null>(null);
  const [output, setOutput] = useState<string | null>(null);
  const pollRef = useRef<ReturnType<typeof setInterval>>();

  const poll = useCallback((id: string) => {
    setTaskId(id);
    setRunning(true);
    setOutput(null);

    pollRef.current = setInterval(async () => {
      try {
        const data = await api.getValidatorStatus(id);
        setTaskResult(data);
        if (data.status === 'complete') {
          setOutput(data.result?.output ?? 'No output');
          setRunning(false);
          clearInterval(pollRef.current);
        } else if (data.status === 'error') {
          setOutput(`Error: ${data.result?.error ?? 'unknown'}`);
          setRunning(false);
          clearInterval(pollRef.current);
        }
      } catch {
        // Polling error, continue
      }
    }, 2000);
  }, []);

  const start = useCallback(async () => {
    try {
      const { task_id } = await api.runValidators();
      poll(task_id);
    } catch {
      setRunning(false);
    }
  }, [poll]);

  useEffect(() => () => clearInterval(pollRef.current), []);

  return { running, taskId, taskResult, output, start };
}

/** SSE connection hook. Connects on mount, disconnects on unmount. */
export function useSSE() {
  useEffect(() => {
    sse.connect();
    return () => sse.disconnect();
  }, []);
}

/** Toast notification state. */
export interface Toast {
  id: number;
  message: string;
  type: ToastType;
}

let toastId = 0;

export function useToast() {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const add = useCallback((message: string, type: ToastType = 'info') => {
    const id = ++toastId;
    setToasts((prev) => [...prev, { id, message, type }]);
    setTimeout(() => setToasts((prev) => prev.filter((t) => t.id !== id)), 3000);
  }, []);

  return { toasts, toast: add };
}
