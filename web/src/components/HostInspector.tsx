import { WifiOff } from 'lucide-react';
import {
  cloudflareStateText,
  dnsResultClass,
  serviceStateText,
  statusClassByCode
} from '../lib/services';
import type { Entry } from '../types';

export function HostInspector({ entry, mutationEnabled, onPreview, onSync }: { entry?: Entry; mutationEnabled: boolean; onPreview: (hostname: string) => Promise<boolean>; onSync: (hostname: string) => Promise<void> }) {
  if (!entry) {
    return (
      <section id="host-inspector" className="panel inspector" aria-live="polite">
        <div className="panel-title"><strong>Selected host</strong><span>Select a row to inspect service state.</span></div>
        <div className="empty-state"><WifiOff size={18} /> No hostname selected.</div>
      </section>
    );
  }
  return (
    <section id="host-inspector" className="panel inspector" aria-live="polite">
      <div className="host-title">
        <strong>{entry.hostname}</strong>
        <div><StatusChip entry={entry} /><span className={`dns-result ${dnsResultClass(entry.dns_resolved)}`}>{entry.dns_resolved || 'FAIL'}</span></div>
      </div>
      <div className="inspector-grid">
        <InspectorLine label="Caddy upstream" value={entry.caddy_upstream || '-'} />
        <InspectorLine label="Source" value={entry.data_source || '-'} />
        <InspectorLine label="Unbound" value={serviceStateText(entry.unbound_status)} tone={entry.unbound_status?.configured ? 'ok' : 'bad'} />
        <InspectorLine label="AdGuard" value={serviceStateText(entry.adguard_status)} tone={entry.adguard_status?.configured ? 'ok' : 'bad'} />
        <InspectorLine label="DHCP lease" value={serviceStateText(entry.dhcp_status)} />
        <InspectorLine label="Cloudflare route" value={cloudflareStateText(entry.cloudflare_status)} tone={entry.cloudflare_status?.configured ? 'violet' : 'bad'} />
      </div>
      <div className="inspector-actions">
        <button type="button" id="inspector-preview" onClick={() => void onPreview(entry.hostname)}>Preview selected</button>
        <button type="button" id="inspector-sync" disabled={!mutationEnabled} onClick={() => void onSync(entry.hostname)}>Sync selected</button>
      </div>
    </section>
  );
}

function StatusChip({ entry }: { entry: Entry }) {
  return <span className={`status-chip ${statusClassByCode(entry.overall_status)}`}>{entry.status_label || 'Unknown'}</span>;
}

function InspectorLine({ label, value, tone = '' }: { label: string; value: string; tone?: string }) {
  return <div className={`inspector-line ${tone}`}><span>{label}</span><strong>{value}</strong></div>;
}
