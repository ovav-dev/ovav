import { useState, useEffect, useCallback, useRef, useMemo } from 'react';
import ToastContainer from './components/Toast';
import Login from './components/Login';
import CommandPalette, { type CommandItem } from './components/CommandPalette';
import BrandLogin from './pages/BrandLogin';
import DashboardSection from './sections/DashboardSection';
import SecuritySection from './sections/SecuritySection';
import ValidatorsSection from './sections/ValidatorsSection';
import MemorySection from './sections/MemorySection';
import AgentsSection from './sections/AgentsSection';
import EconomySection from './sections/EconomySection';
import ProfilesSection from './sections/ProfilesSection';
import SystemSection from './sections/SystemSection';
import GitSection from './sections/GitSection';
import OperationsSection from './sections/OperationsSection';
import VaultSection from './sections/VaultSection';
import { useToast } from './hooks/useApi';
import { api } from './services/api';
import type { StatusResponse } from './types';
import './App.css';

interface NavSection {
  id: string;
  icon: string;
  label: string;
  children?: { id: string; label: string }[];
}

const NAV: NavSection[] = [
  { id: 'brand-login', icon: '\uD83D\uDD11', label: 'Brand Login' },
  { id: 'dashboard', icon: '\u25C8', label: 'Dashboard' },
  { id: 'security', icon: '\uD83D\uDEE1', label: 'Security', children: [
    { id: 'security-auth-analytics', label: 'Auth Analytics' },
    { id: 'security-audit', label: 'Audit Log' },
    { id: 'security-alarms', label: 'Alarms' },
    { id: 'security-integrity', label: 'Integrity Scan' },
  ]},
  { id: 'validators', icon: '\u2713', label: 'Validators' },
  { id: 'memory', icon: '\uD83E\uDDE0', label: 'Memory & State', children: [
    { id: 'memory-ledger', label: 'Context Ledger' },
    { id: 'memory-beliefs', label: 'Beliefs' },
    { id: 'memory-capsules', label: 'Capsules' },
  ]},
  { id: 'agents', icon: '\uD83E\uDD16', label: 'Agents & Topology', children: [
    { id: 'agents-list', label: 'Agent List' },
    { id: 'agents-topology', label: 'Topology' },
    { id: 'agents-permissions', label: 'Permissions' },
  ]},
  { id: 'economy', icon: '\uD83D\uDCB0', label: 'Economy' },
  { id: 'profiles', icon: '\uD83D\uDCE6', label: 'Profiles' },
  { id: 'operations', icon: '\uD83D\uDD27', label: 'Operations' },
  { id: 'vault', icon: '\uD83D\uDD10', label: 'Vault' },
  { id: 'system', icon: '\u2699', label: 'System', children: [
    { id: 'system-health', label: 'Health' },
    { id: 'system-config', label: 'Config' },
    { id: 'system-registry', label: 'Registries' },
    { id: 'system-sbom', label: 'SBOM' },
    { id: 'system-kc', label: 'Knowledge Compiler' },
  ]},
  { id: 'git', icon: '\uD83D\uDD00', label: 'Git', children: [
    { id: 'git-branches', label: 'Branches' },
    { id: 'git-commits', label: 'Commits' },
    { id: 'git-worktrees', label: 'Worktrees' },
  ]},
];

// Role-based section access
const ROLE_SECTIONS: Record<string, string[]> = {
  admin:    ['brand-login', 'dashboard', 'security', 'validators', 'memory', 'agents', 'economy', 'profiles', 'operations', 'vault', 'system', 'git'],
  operator: ['brand-login', 'dashboard', 'security', 'validators', 'memory', 'agents', 'economy', 'vault', 'git'],
  viewer:   ['brand-login', 'dashboard', 'validators', 'economy'],
};

const KEY_MAP: Record<string, string> = {
  '1': 'dashboard', '2': 'security', '3': 'validators', '4': 'memory',
  '5': 'agents', '6': 'economy', '7': 'operations', '8': 'vault', '9': 'system', '0': 'git',
};

export default function App() {
  const [section, setSection] = useState('dashboard');
  const [subsection, setSubsection] = useState<string | null>(null);
  const [expanded, setExpanded] = useState<Record<string, boolean>>({});
  const [status, setStatus] = useState<StatusResponse | null>(null);
  const [authenticated, setAuthenticated] = useState(false);
  const [userRole, setUserRole] = useState<string>('viewer');
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [isBrandLogin, setIsBrandLogin] = useState(() =>
    window.location.hash === '#/brand-login'
  );
  const { toasts, toast } = useToast();
  const clockRef = useRef<HTMLSpanElement>(null);

  // Hash-based routing for standalone pages
  useEffect(() => {
    const onHashChange = () => setIsBrandLogin(window.location.hash === '#/brand-login');
    window.addEventListener('hashchange', onHashChange);
    return () => window.removeEventListener('hashchange', onHashChange);
  }, []);

  // Check for existing auth token
  useEffect(() => {
    const token = localStorage.getItem('ovav_cpanel_token');
    const role = localStorage.getItem('ovav_cpanel_role') || 'viewer';
    if (token) {
      setAuthenticated(true);
      setUserRole(role);
    }
  }, []);

  const handleLogin = (_token: string, role: string, _email: string) => {
    setAuthenticated(true);
    setUserRole(role);
  };
  // SSE disabled — blocks single-threaded server preventing all other API calls

  useEffect(() => {
    const tick = () => { if (clockRef.current) clockRef.current.textContent = new Date().toLocaleTimeString(); };
    tick(); const t = setInterval(tick, 1000); return () => clearInterval(t);
  }, []);

  useEffect(() => {
    const fetch = async () => { try { setStatus(await api.getStatus()); } catch { /* */ } };
    fetch(); const t = setInterval(fetch, 15000); return () => clearInterval(t);
  }, []);

  // Cmd+K / Ctrl+K → open command palette
  useEffect(() => {
    const h = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        setPaletteOpen(v => !v);
      }
    };
    window.addEventListener('keydown', h);
    return () => window.removeEventListener('keydown', h);
  }, []);

  useEffect(() => {
    const h = (e: KeyboardEvent) => {
      if (e.target instanceof HTMLInputElement || e.target instanceof HTMLSelectElement) return;
      const v = KEY_MAP[e.key];
      if (v) navigate(v);
      if (e.key === 'r' || e.key === 'R') window.dispatchEvent(new Event('cpanel-refresh'));
    };
    window.addEventListener('keydown', h);
    return () => window.removeEventListener('keydown', h);
  }, []);

  const navigate = useCallback((id: string) => {
    const parent = NAV.find(s => s.id === id || s.children?.some(c => c.id === id));
    if (parent?.children) {
      setSection(parent.id);
      setSubsection(id);
      setExpanded(prev => ({ ...prev, [parent.id]: true }));
    } else {
      setSection(id);
      setSubsection(null);
    }
  }, []);

  const toggleSection = (id: string) => {
    setExpanded(prev => ({ ...prev, [id]: !prev[id] }));
    if (section !== id) { setSection(id); setSubsection(null); }
  };

  const paletteItems: CommandItem[] = useMemo(() => {
    const navItems: CommandItem[] = NAV
      .filter(item => (ROLE_SECTIONS[userRole] || []).includes(item.id))
      .flatMap(item => [
        {
          id: `nav-${item.id}`,
          label: item.label,
          category: 'navigation' as const,
          icon: item.icon,
          description: 'Navigate to section',
          action: () => navigate(item.id),
        },
        ...(item.children || []).map(child => ({
          id: `nav-${child.id}`,
          label: child.label,
          category: 'navigation' as const,
          icon: item.icon,
          description: `Navigate to ${item.label}`,
          action: () => navigate(child.id),
        })),
      ]);

    const actionItems: CommandItem[] = [
      {
        id: 'action-refresh',
        label: 'Refresh Status',
        category: 'action' as const,
        icon: '\uD83D\uDD04',
        description: 'Re-fetch system status',
        action: async () => {
          try { setStatus(await api.getStatus()); toast('Status refreshed', 'info'); } catch { toast('Refresh failed', 'err'); }
        },
      },
      {
        id: 'action-copy-ip',
        label: 'Copy Node IP',
        category: 'action' as const,
        icon: '\uD83D\uDCCB',
        description: 'Copy public IP to clipboard',
        action: async () => {
          const ip = window.location.hostname;
          try { await navigator.clipboard.writeText(ip); toast(`Copied: ${ip}`, 'ok'); } catch { toast('Clipboard unavailable', 'err'); }
        },
      },
      {
        id: 'action-expand-all',
        label: 'Expand All Sections',
        category: 'action' as const,
        icon: '\u2B06',
        description: 'Expand all sidebar sections',
        action: () => {
          const all = NAV.reduce((acc, s) => ({ ...acc, [s.id]: true }), {});
          setExpanded(all);
        },
      },
      {
        id: 'action-collapse-all',
        label: 'Collapse All Sections',
        category: 'action' as const,
        icon: '\u2B07',
        description: 'Collapse all sidebar sections',
        action: () => setExpanded({}),
      },
      {
        id: 'action-open-vault',
        label: 'Open Vault',
        category: 'action' as const,
        icon: '\uD83D\uDD10',
        description: 'Go to Vault section',
        action: () => navigate('vault'),
      },
    ];

    return [...navItems, ...actionItems];
  }, [userRole, navigate, status, toast]);

  const alarms = status?.session?.canary_alarms ?? 0;
  const model = status?.economy?.model ?? '?';
  const sectionLabel = NAV.find(s => s.id === section)?.label || 'Dashboard';
  const subsectionLabel = subsection
    ? NAV.find(s => s.children?.some(c => c.id === subsection))?.children?.find(c => c.id === subsection)?.label
    : null;

  const renderContent = () => {
    const key = subsection || section;
    switch (key) {
      case 'brand-login': return <BrandLogin />;
      case 'dashboard': return <DashboardSection toast={toast} />;
      case 'security': case 'security-audit': return <SecuritySection toast={toast} view="audit" />;
      case 'security-alarms': return <SecuritySection toast={toast} view="alarms" />;
      case 'security-auth-analytics': return <SecuritySection toast={toast} view="auth-analytics" />;
      case 'security-integrity': return <SecuritySection toast={toast} view="integrity" />;
      case 'validators': return <ValidatorsSection toast={toast} />;
      case 'memory': case 'memory-ledger': return <MemorySection toast={toast} view="ledger" />;
      case 'memory-beliefs': return <MemorySection toast={toast} view="beliefs" />;
      case 'memory-capsules': return <MemorySection toast={toast} view="capsules" />;
      case 'agents': case 'agents-list': return <AgentsSection toast={toast} view="list" />;
      case 'agents-topology': return <AgentsSection toast={toast} view="topology" />;
      case 'agents-permissions': return <AgentsSection toast={toast} view="permissions" />;
      case 'economy': return <EconomySection toast={toast} />;
      case 'profiles': return <ProfilesSection toast={toast} />;
      case 'operations': return <OperationsSection toast={toast} />;
      case 'system': case 'system-health': return <SystemSection toast={toast} view="health" />;
      case 'system-config': return <SystemSection toast={toast} view="config" />;
      case 'system-registry': return <SystemSection toast={toast} view="registry" />;
      case 'system-sbom': return <SystemSection toast={toast} view="sbom" />;
      case 'system-kc': return <SystemSection toast={toast} view="kc" />;
      case 'git': case 'git-branches': return <GitSection toast={toast} view="branches" />;
      case 'git-commits': return <GitSection toast={toast} view="commits" />;
      case 'git-worktrees': return <GitSection toast={toast} view="worktrees" />;
      case 'vault': return <VaultSection />;
      default: return <DashboardSection toast={toast} />;
    }
  };

  return (
    <>
      {isBrandLogin ? (
        <BrandLogin />
      ) : !authenticated ? (
        <Login onLogin={handleLogin} />
      ) : (
      <>
      <ToastContainer toasts={toasts} />
      <CommandPalette
        items={paletteItems}
        isOpen={paletteOpen}
        onClose={() => setPaletteOpen(false)}
      />
      <nav className="sidebar">
        <div className="sidebar-logo">
          <div className="sidebar-logo-brand">
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 128 128" width="24" height="24">
              <defs>
                <linearGradient id="sg" x1="0%" y1="0%" x2="100%" y2="100%">
                  <stop offset="0%" stopColor="#7c3aed"/>
                  <stop offset="100%" stopColor="#8b5cf6"/>
                </linearGradient>
              </defs>
              <rect width="128" height="128" rx="24" fill="url(#sg)"/>
              <text x="64" y="82" fontFamily="system-ui,sans-serif" fontSize="56" fontWeight="700" fill="white" textAnchor="middle" letterSpacing="-2">OV</text>
              <circle cx="96" cy="40" r="8" fill="#f59e0b"/>
            </svg>
            <h1><span>OV</span>AV</h1>
          </div>
          <div className="ver">Systems · Control Plane</div>
        </div>
        <div className="sidebar-nav">
          {NAV.filter(item => (ROLE_SECTIONS[userRole] || []).includes(item.id)).map(item => (
            <div key={item.id}>
              <div
                className={`nav-item nav-section${section === item.id && !subsection ? ' active' : ''}`}
                onClick={() => toggleSection(item.id)}
              >
                <span className="nav-icon">{item.icon}</span>
                {item.label}
                {item.children && (
                  <span className={`nav-arrow ${expanded[item.id] ? 'open' : ''}`}>{'\u25BE'}</span>
                )}
              </div>
              {item.children && expanded[item.id] && (
                <div className="nav-sub">
                  {item.children.map(child => (
                    <div
                      key={child.id}
                      className={`nav-item nav-sub-item${subsection === child.id ? ' active' : ''}`}
                      onClick={() => navigate(child.id)}
                    >
                      {child.label}
                    </div>
                  ))}
                </div>
              )}
            </div>
          ))}
        </div>
        <div className="sidebar-footer">
          <div style={{ marginBottom: 6 }}>
            <span className={`tag ${userRole === 'admin' ? 'tag-ok' : userRole === 'operator' ? 'tag-warn' : 'tag-fail'}`} style={{ fontSize: 10 }}>
              {userRole}
            </span>
          </div>
          <div className="flex-row" style={{ gap: 4 }}>
            <span className="live-dot">{'\u25CF'}</span> LIVE
          </div>
          <div style={{ fontSize: 10, marginTop: 4 }}><span ref={clockRef}>--:--:--</span></div>
        </div>
      </nav>

      <main className="main">
        <div className="header-bar">
          <div>
            <h2>{sectionLabel}</h2>
            {subsectionLabel && <div className="header-sub">{subsectionLabel}</div>}
          </div>
          <div className="header-stats">
            <span className={`header-stat ${alarms === 0 ? 'ok' : 'fail'}`}>
              <span className={`status-dot ${alarms === 0 ? 'ok' : 'fail'}`} />{alarms} alarms
            </span>
            <span className="header-stat"><span className="status-dot ok" />{model}</span>
            <button className="cmd-k-btn" onClick={() => setPaletteOpen(true)} title="Command Palette (Cmd+K)">
              {'\u2318'}K
            </button>
          </div>
        </div>
        <div className="view-container" key={subsection || section}>
          {renderContent()}
        </div>
      </main>
      </>
      )}
    </>
  );
}
