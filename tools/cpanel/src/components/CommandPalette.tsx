import { useState, useEffect, useRef, useCallback, useMemo } from 'react';

export interface CommandItem {
  id: string;
  label: string;
  category: 'navigation' | 'action' | 'recent';
  icon?: string;
  description?: string;
  action: () => void;
}

interface CommandPaletteProps {
  items: CommandItem[];
  isOpen: boolean;
  onClose: () => void;
}

function useRecentItems() {
  const STORAGE_KEY = 'ovav_cp_recent';
  const [recent, setRecent] = useState<CommandItem[]>(() => {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      return raw ? JSON.parse(raw) : [];
    } catch {
      return [];
    }
  });

  const addRecent = useCallback((item: CommandItem) => {
    setRecent(prev => {
      const filtered = prev.filter(i => i.id !== item.id);
      const next = [item, ...filtered].slice(0, 5);
      try { localStorage.setItem(STORAGE_KEY, JSON.stringify(next)); } catch { /**/ }
      return next;
    });
  }, []);

  return { recent, addRecent };
}

function fuzzyMatch(query: string, label: string): boolean {
  if (!query) return true;
  const q = query.toLowerCase();
  const l = label.toLowerCase();
  let qi = 0;
  for (let i = 0; i < l.length && qi < q.length; i++) {
    if (l[i] === q[qi]) qi++;
  }
  return qi === q.length;
}

export default function CommandPalette({ items, isOpen, onClose }: CommandPaletteProps) {
  const [query, setQuery] = useState('');
  const [activeIndex, setActiveIndex] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const { recent, addRecent } = useRecentItems();

  useEffect(() => {
    if (isOpen) {
      setQuery('');
      setActiveIndex(0);
      setTimeout(() => inputRef.current?.focus(), 20);
    }
  }, [isOpen]);

  const filtered = useMemo(() => {
    if (!query.trim()) {
      const withRecent = [
        ...recent,
        ...items.filter(i => i.category !== 'recent' && !recent.some(r => r.id === i.id)),
      ];
      return withRecent;
    }
    return items.filter(item => fuzzyMatch(query, item.label));
  }, [query, items, recent]);

  const grouped = useMemo(() => {
    const groups: { label: string; items: CommandItem[] }[] = [];
    const nav = filtered.filter(i => i.category === 'navigation');
    const act = filtered.filter(i => i.category === 'action');
    const rec = !query.trim() ? filtered.filter(i => i.category === 'recent') : [];
    if (nav.length) groups.push({ label: 'Navigation', items: nav });
    if (act.length) groups.push({ label: 'Actions', items: act });
    if (rec.length) groups.push({ label: 'Recent', items: rec });
    return groups;
  }, [filtered, query]);

  const flatItems = useMemo(() => grouped.flatMap(g => g.items), [grouped]);

  useEffect(() => { setActiveIndex(0); }, [query]);

  useEffect(() => {
    const el = listRef.current?.querySelector(`[data-index="${activeIndex}"]`) as HTMLElement | null;
    el?.scrollIntoView({ block: 'nearest' });
  }, [activeIndex]);

  const execute = useCallback((item: CommandItem) => {
    addRecent(item);
    item.action();
    onClose();
  }, [addRecent, onClose]);

  useEffect(() => {
    if (!isOpen) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') { onClose(); return; }
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        setActiveIndex(i => Math.min(i + 1, flatItems.length - 1));
      }
      if (e.key === 'ArrowUp') {
        e.preventDefault();
        setActiveIndex(i => Math.max(i - 1, 0));
      }
      if (e.key === 'Enter') {
        e.preventDefault();
        const item = flatItems[activeIndex];
        if (item) execute(item);
      }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [isOpen, onClose, flatItems, activeIndex, execute]);

  if (!isOpen) return null;

  let globalIndex = 0;

  return (
    <div className="cp-overlay" onClick={onClose}>
      <div className="cp-modal" onClick={e => e.stopPropagation()}>
        <div className="cp-search-row">
          <span className="cp-search-icon">{'\u2318'}</span>
          <input
            ref={inputRef}
            className="cp-input"
            placeholder="Search commands, navigation, actions..."
            value={query}
            onChange={e => setQuery(e.target.value)}
          />
          <kbd className="cp-esc-hint">ESC</kbd>
        </div>

        <div className="cp-divider" />

        <div className="cp-results" ref={listRef}>
          {grouped.length === 0 && (
            <div className="cp-empty">No results for &ldquo;{query}&rdquo;</div>
          )}
          {grouped.map(group => (
            <div key={group.label} className="cp-group">
              <div className="cp-group-label">{group.label}</div>
              {group.items.map(item => {
                const idx = globalIndex++;
                const isActive = idx === activeIndex;
                return (
                  <div
                    key={item.id}
                    data-index={idx}
                    className={`cp-item${isActive ? ' cp-item-active' : ''}`}
                    onClick={() => execute(item)}
                    onMouseEnter={() => setActiveIndex(idx)}
                  >
                    {item.icon && <span className="cp-item-icon">{item.icon}</span>}
                    <div className="cp-item-content">
                      <span className="cp-item-label">{item.label}</span>
                      {item.description && (
                        <span className="cp-item-desc">{item.description}</span>
                      )}
                    </div>
                    {isActive && <kbd className="cp-enter-hint">↵</kbd>}
                  </div>
                );
              })}
            </div>
          ))}
        </div>

        <div className="cp-footer">
          <span><kbd>↑↓</kbd> navigate</span>
          <span><kbd>↵</kbd> select</span>
          <span><kbd>esc</kbd> close</span>
        </div>
      </div>
    </div>
  );
}
