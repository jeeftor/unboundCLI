import { memo } from 'react';
import { HelpCircle } from 'lucide-react';
import type { AuthMeta } from '../lib/authMeta';

export const AuthBadge = memo(function AuthBadge({ value, info, showIcon = true }: { value: string; info: Record<string, AuthMeta>; showIcon?: boolean }) {
  const meta = info[value] ?? { label: value, desc: '', icon: HelpCircle, tone: 'gray' };
  const cls = value === 'none' ? 'auth-badge none' : `auth-badge ${value.replace(/_/g, '-')}`;
  const Icon = meta.icon;
  return (
    <span className={cls} title={meta.desc}>
      {showIcon && <Icon size={12} className="auth-badge-icon" />}
      {meta.label}
      <HelpCircle size={11} className="auth-badge-help" />
    </span>
  );
});
