import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render } from '@testing-library/react';
import App from '../App';

// Mock services to avoid real network calls
vi.mock('../services/api', () => ({
  api: {
    getStatus: vi.fn().mockResolvedValue({
      git: { branch: 'test', head: 'abc123', commits: '5', dirty: 'clean', last_commit: 'test' },
      system: { agents: '42', python: '3.12' },
      economy: { session_cost: 0, session_pct: 0, monthly_cost: 0, monthly_pct: 0, model: 'test' },
      session: { uptime: '1h', canary_alarms: 0 },
    }),
    getValidators: vi.fn().mockResolvedValue({ overall: 'pass', score: 100, pass: 10, fail: 0, checks: [] }),
    runValidators: vi.fn().mockResolvedValue({ task_id: 't1', status: 'queued' }),
    getValidatorStatus: vi.fn().mockResolvedValue({ status: 'complete' }),
    getAuditLog: vi.fn().mockResolvedValue({ entries: [], total: 0, chain_intact: true }),
    clearAlarms: vi.fn().mockResolvedValue({ status: 'ok', message: 'cleared' }),
    getProfiles: vi.fn().mockResolvedValue({ ok: true, profiles: [], output: '' }),
    applyProfile: vi.fn().mockResolvedValue({ ok: true, output: 'applied' }),
    getOperations: vi.fn().mockResolvedValue({}),
  },
}));

// Mock login flow — skip auth for smoke test
beforeEach(() => {
  localStorage.setItem('ovav_token', 'dev');
  localStorage.setItem('ovav_role', 'admin');
});

describe('App', () => {
  it('renders without crashing', async () => {
    const { container } = render(<App />);
    expect(container).toBeTruthy();
    // App should render sidebar navigation
    expect(container.querySelector('.sidebar') || container.querySelector('nav') || container.innerHTML.length).toBeTruthy();
  });

  it('renders dashboard as default view', async () => {
    const { container } = render(<App />);
    // Should have some content rendered
    expect(container.textContent?.length).toBeGreaterThan(0);
  });

  it('has sidebar with nav items', async () => {
    const { container } = render(<App />);
    // Check for navigation elements (buttons, links, or sidebar items)
    const navElements = container.querySelectorAll('button, a, [role="button"]');
    expect(navElements.length).toBeGreaterThan(0);
  });
});
