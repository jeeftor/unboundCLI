import { Gauge, FileCode2, ShieldCheck, Stethoscope } from 'lucide-react';
import type { ComponentType } from 'react';

export type TabId = 'dashboard' | 'caddyfile' | 'auth' | 'diagnostics';

type TabDef = {
  id: TabId;
  label: string;
  icon: ComponentType<{ size?: number }>;
};

const TABS: TabDef[] = [
  { id: 'dashboard', label: 'Dashboard', icon: Gauge },
  { id: 'caddyfile', label: 'Caddyfile', icon: FileCode2 },
  { id: 'auth', label: 'Auth Flows', icon: ShieldCheck },
  { id: 'diagnostics', label: 'Diagnostics', icon: Stethoscope },
];

const VALID_IDS = new Set(TABS.map(t => t.id));

// eslint-disable-next-line react-refresh/only-export-components
export function tabFromHash(): TabId {
  const hash = window.location.hash.replace(/^#\/?/, '');
  if (VALID_IDS.has(hash as TabId)) return hash as TabId;
  return 'dashboard';
}

// eslint-disable-next-line react-refresh/only-export-components
export function setTabHash(tab: TabId) {
  const target = `#/${tab}`;
  if (window.location.hash !== target) {
    window.location.hash = target;
  }
}

export function TabBar({ active, onChange }: { active: TabId; onChange: (tab: TabId) => void }) {
  return (
    <nav className="tabbar" aria-label="Primary tabs">
      {TABS.map(tab => {
        const Icon = tab.icon;
        const isActive = tab.id === active;
        return (
          <button
            key={tab.id}
            type="button"
            className={`tabbar-item ${isActive ? 'active' : ''}`}
            aria-current={isActive ? 'page' : undefined}
            onClick={() => onChange(tab.id)}
          >
            <Icon size={15} />
            <span>{tab.label}</span>
          </button>
        );
      })}
    </nav>
  );
}
