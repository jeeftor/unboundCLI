import { STATUS_INFO } from '../lib/authMeta';
import type { AuthStatus } from '../types';

export function StatusIcon({ status }: { status: AuthStatus }) {
  const meta = STATUS_INFO[status] ?? STATUS_INFO.unknown;
  const Icon = meta.icon;
  return <span title={meta.desc}><Icon size={16} className={`auth-status-icon ${status}`} /></span>;
}
