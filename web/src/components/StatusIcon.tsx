import { memo } from 'react';
import { STATUS_INFO } from '../lib/authMeta';
import type { AuthStatus } from '../types';

export const StatusIcon = memo(function StatusIcon({ status }: { status: AuthStatus }) {
  const meta = STATUS_INFO[status] ?? STATUS_INFO.unknown;
  const Icon = meta.icon;
  return <span title={meta.desc}><Icon size={16} className={`auth-status-icon ${status}`} /></span>;
});
