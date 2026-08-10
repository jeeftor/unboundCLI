import { CheckCircle2, ChevronDown, HelpCircle } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import type { AuthMeta } from '../lib/authMeta';

export type AuthField = 'wan_auth' | 'lan_auth' | 'api_auth';

export type PendingChange = {
  hostname: string;
  field: AuthField;
  oldValue: string;
  newValue: string;
};

// eslint-disable-next-line react-refresh/only-export-components
export function changesKey(hostname: string, field: AuthField) {
  return `${hostname}::${field}`;
}

export function EditableAuthBadge({
  hostname,
  field,
  value,
  info,
  disabled,
  pendingValue,
  onChange,
}: {
  hostname: string;
  field: AuthField;
  value: string;
  info: Record<string, AuthMeta>;
  disabled?: boolean;
  pendingValue?: string;
  onChange: (hostname: string, field: AuthField, newValue: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLSpanElement>(null);
  const displayValue = pendingValue ?? value;
  const isPending = pendingValue !== undefined && pendingValue !== value;
  const meta = info[displayValue] ?? { label: displayValue, desc: '', icon: HelpCircle, tone: 'gray' };
  const Icon = meta.icon;
  const cls = displayValue === 'none' ? 'auth-badge none' : `auth-badge ${displayValue.replace(/_/g, '-')}`;

  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [open]);

  if (disabled) {
    return <span className="auth-na" title="This host is not exposed to the internet (no Cloudflare tunnel). WAN auth doesn't apply.">Not exposed</span>;
  }

  return (
    <span className="auth-badge-editable" ref={ref}>
      <button
        type="button"
        className={`${cls} ${isPending ? 'pending' : ''}`}
        onClick={(e) => {
          e.stopPropagation();
          setOpen(o => !o);
        }}
        title={meta.desc + (isPending ? ' (pending change)' : '')}
      >
        <Icon size={12} className="auth-badge-icon" />
        {meta.label}
        {isPending && <span className="auth-badge-pending-dot" />}
        <ChevronDown size={10} className="auth-badge-chevron" />
      </button>
      {open && (
        <div className="auth-dropdown" onClick={e => e.stopPropagation()}>
          {Object.entries(info).map(([key, m]) => {
            const ItemIcon = m.icon;
            const isSelected = key === displayValue;
            return (
              <button
                key={key}
                type="button"
                className={`auth-dropdown-item ${isSelected ? 'selected' : ''}`}
                onClick={() => {
                  onChange(hostname, field, key);
                  setOpen(false);
                }}
                title={m.desc}
              >
                <ItemIcon size={13} className="auth-dropdown-item-icon" />
                <span className="auth-dropdown-item-label">{m.label}</span>
                {isSelected && <CheckCircle2 size={12} className="auth-dropdown-item-check" />}
              </button>
            );
          })}
        </div>
      )}
    </span>
  );
}
