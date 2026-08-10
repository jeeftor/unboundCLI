import '../styles/EntriesTable.css';
import { CircleAlert, Network } from 'lucide-react';
import { useRef, useState, type KeyboardEvent } from 'react';
import { createPortal } from 'react-dom';
import { getHostnameDecision, suppressionKey } from '../lib/hostnameDecision';
import {
  dnsResultClass,
  statusClassByCode
} from '../lib/services';
import type { Entry, ServiceKey } from '../types';
import { CloudflareDetails } from './CloudflarePanel';
import { CopyButton } from './CopyButton';

export function EntriesTable({
  entries,
  selectedHostname,
  mutationEnabled,
  enabledServices: _enabledServices,
  caddyServerIP,
  suppressed,
  onToggleSuppress,
  onSelect,
  onQuickSync,
  onOpenModify,
  onOpenVisualize,
  onRemove,
}: {
  entries: Entry[];
  selectedHostname: string;
  mutationEnabled: boolean;
  enabledServices: Partial<Record<ServiceKey, boolean>>;
  caddyServerIP: string;
  suppressed: Set<string>;
  onToggleSuppress: (key: string) => void;
  onSelect: (hostname: string) => void;
  onQuickSync: (hostname: string) => void;
  onOpenModify: (hostname: string) => void;
  onOpenVisualize: (hostname: string) => void;
  onRemove: (hostname: string, service?: string) => Promise<void>;
}) {
  return (
    <section id="entries-panel" className="panel entries-panel">
      <table>
        <thead>
          <tr>
            <th>Hostname</th><th>Status</th><th>Services</th><th>Caddy upstream</th><th>DNS</th><th>Cloudflare route</th><th>Actions</th>
          </tr>
        </thead>
        <tbody id="entries">
          {entries.map((entry) => (
            <EntryRow
              key={entry.hostname}
              entry={entry}
              selected={entry.hostname === selectedHostname}
              mutationEnabled={mutationEnabled}
              caddyServerIP={caddyServerIP}
              suppressed={suppressed}
              onToggleSuppress={onToggleSuppress}
              onSelect={onSelect}
              onQuickSync={onQuickSync}
              onOpenModify={onOpenModify}
              onOpenVisualize={onOpenVisualize}
              onRemove={onRemove}
            />
          ))}
        </tbody>
      </table>
    </section>
  );
}

function EntryRow({
  entry,
  selected,
  mutationEnabled,
  caddyServerIP,
  suppressed,
  onToggleSuppress,
  onSelect,
  onQuickSync,
  onOpenModify,
  onOpenVisualize,
  onRemove,
}: {
  entry: Entry;
  selected: boolean;
  mutationEnabled: boolean;
  caddyServerIP: string;
  suppressed: Set<string>;
  onToggleSuppress: (key: string) => void;
  onSelect: (hostname: string) => void;
  onQuickSync: (hostname: string) => void;
  onOpenModify: (hostname: string) => void;
  onOpenVisualize: (hostname: string) => void;
  onRemove: (hostname: string, service?: string) => Promise<void>;
}) {
  const isStale = entry.overall_status === 4;
  const decision = getHostnameDecision(entry, caddyServerIP);
  const dnsOK = dnsResultClass(entry.dns_resolved) === 'ok';
  const statusDetail = entry.overall_status === 3
    ? 'Not in DNS'
    : entry.overall_status === 4
      ? 'Needs cleanup'
      : entry.dns_resolved
        ? 'Resolved'
        : 'Not in DNS';
  const selectRow = () => onSelect(entry.hostname);
  const onRowKeyDown = (event: KeyboardEvent<HTMLTableRowElement>) => {
    if (event.key !== 'Enter' && event.key !== ' ') return;
    event.preventDefault();
    selectRow();
  };
  const primaryKey = suppressionKey(entry.hostname, decision.kind);
  const primarySuppressed = suppressed.has(primaryKey);
  return (
    <tr data-hostname={entry.hostname} className={selected ? 'selected-row' : ''} onClick={selectRow} onKeyDown={onRowKeyDown} tabIndex={0} aria-selected={selected}>
      <td data-label="Hostname">
        <strong>{entry.hostname}</strong>
        <span className="subtle">{entry.data_source || 'Caddy route'} <i /></span>
        {(decision.kind === 'collision' || decision.kind === 'mismatch') && !primarySuppressed && (
          <AlertBadge title={decision.title} summary={decision.summary} facts={decision.facts} actions={decision.actions} variant="collision" onMarkIntentional={() => onToggleSuppress(primaryKey)} />
        )}
        {decision.warnings
          .filter(w => !suppressed.has(suppressionKey(entry.hostname, w.kind)))
          .map((w) => (
            <AlertBadge key={w.kind} title={w.title} summary={w.summary} facts={w.facts} actions={w.actions} variant="mismatch" onMarkIntentional={() => onToggleSuppress(suppressionKey(entry.hostname, w.kind))} />
          ))}
      </td>
      <td data-label="Status"><StatusChip entry={entry} /><span className="status-subtext">{statusDetail}</span></td>
      <td data-label="Services"><ServiceBadges entry={entry} /></td>
      <td data-label="Caddy upstream"><span>{entry.caddy_upstream || '-'}{entry.caddy_upstream && <CopyButton value={entry.caddy_upstream} label="upstream" />}</span><span className="subtle">admin {entry.caddy_ip || '-'}</span><span className="protocol-pill">HTTP</span></td>
      <td data-label="DNS"><span className={`dns-result ${dnsResultClass(entry.dns_resolved)}`}>{entry.dns_resolved || 'FAIL'}</span><span className="status-subtext">{dnsOK ? 'A record' : 'NXDOMAIN'}</span></td>
      <td data-label="Cloudflare route"><CloudflareDetails status={entry.cloudflare_status} hostname={entry.hostname} /></td>
      <td data-label="Actions">
        <div className="row-actions">
          {isStale ? (
            <button className="row-remove-btn" type="button" disabled={!mutationEnabled} onClick={(e) => { e.stopPropagation(); void onRemove(entry.hostname); }}>
              Remove
            </button>
          ) : (
            <button className="row-sync-btn" type="button" onClick={(e) => { e.stopPropagation(); onQuickSync(entry.hostname); }}>
              Sync
            </button>
          )}
          <button className="row-modify-btn" type="button" onClick={(e) => { e.stopPropagation(); onOpenModify(entry.hostname); }}>
            Modify
          </button>
          <button className="row-visualize-btn" type="button" title="Visualize access flow" onClick={(e) => { e.stopPropagation(); onOpenVisualize(entry.hostname); }}>
            <Network size={12} /> Flow
          </button>
        </div>
      </td>
    </tr>
  );
}

export function StatusChip({ entry }: { entry: Entry }) {
  return <span className={`status-chip ${statusClassByCode(entry.overall_status)}`}>{entry.status_label || 'Unknown'}</span>;
}

function ServiceBadges({ entry }: { entry: Entry }) {
  return (
    <div className="service-stack">
      <ServiceBadge name="Unbound" status={entry.unbound_status} />
      <ServiceBadge name="AdGuard" status={entry.adguard_status} />
      <ServiceBadge name="DHCP" status={entry.dhcp_status} />
    </div>
  );
}

function ServiceBadge({ name, status }: { name: string; status: { configured: boolean; in_sync: boolean; ip: string } }) {
  let tone = 'missing';
  let label = 'Missing';
  if (status.configured && status.in_sync) {
    tone = 'ok';
    label = status.ip || 'In sync';
  } else if (status.configured) {
    tone = 'bad';
    label = status.ip || 'Mismatch';
  }
  return <span className={`service-badge ${tone}`}><strong>{name}</strong>{label}</span>;
}

function AlertBadge({ title, summary, facts, actions, variant, onMarkIntentional }: {
  title: string;
  summary: string;
  facts: string[];
  actions: { label: string; description: string }[];
  variant: 'collision' | 'mismatch';
  onMarkIntentional?: () => void;
}) {
  const [pos, setPos] = useState<{ top: number; left: number } | null>(null);
  const ref = useRef<HTMLSpanElement>(null);

  const handleMouseEnter = () => {
    if (ref.current) {
      const r = ref.current.getBoundingClientRect();
      setPos({ top: r.bottom + 6, left: r.left });
    }
  };

  return (
    <span
      ref={ref}
      className={`decision-row-alert alert-badge-wrap${variant === 'mismatch' ? ' mismatch' : ''}`}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={() => setPos(null)}
    >
      <CircleAlert size={11} /> {title}
      {pos && createPortal(
        <div
          className={`alert-tooltip${variant === 'mismatch' ? ' mismatch' : ''}`}
          style={{ position: 'fixed', top: pos.top, left: pos.left, zIndex: 9999 }}
        >
          <strong>{title}</strong>
          <span>{summary}</span>
          {facts.length > 0 && <ul>{facts.map((f) => <li key={f}>{f}</li>)}</ul>}
          {actions.length > 0 && (
            <div className="alert-tooltip-actions">
              {actions.map((a) => (
                a.label === 'Mark intentional' && onMarkIntentional
                  ? <span key={a.label}>
                      <em>{a.label}:</em> {a.description}{' '}
                      <button
                        type="button"
                        className="suppress-inline-btn"
                        onClick={(e) => { e.stopPropagation(); setPos(null); onMarkIntentional(); }}
                      >
                        Suppress warning
                      </button>
                    </span>
                  : <span key={a.label}><em>{a.label}:</em> {a.description}</span>
              ))}
            </div>
          )}
        </div>,
        document.body
      )}
    </span>
  );
}
