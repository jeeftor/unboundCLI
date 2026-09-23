import { useState } from 'react';
import { api } from '../api/client';
import type { OwnershipAdoptionPreview } from '../types';

export function OwnershipPanel({ mutationEnabled }: { mutationEnabled: boolean }) {
  const [provider, setProvider] = useState<'adguard' | 'cloudflare'>('adguard');
  const [preview, setPreview] = useState<OwnershipAdoptionPreview | null>(null);
  const [selected, setSelected] = useState<Set<string>>(() => new Set());
  const [message, setMessage] = useState('Preview provider records before explicitly adopting them.');
  const [busy, setBusy] = useState(false);
  const load = async () => {
    setBusy(true); setMessage('');
    try {
      const next = await api.ownershipPreview(provider);
      setPreview(next); setSelected(new Set(next.candidates.map((candidate) => candidate.id)));
      setMessage(next.candidates.length ? 'Select resources to adopt, then confirm.' : 'No unowned resources found.');
    } catch (error) { setMessage(error instanceof Error ? error.message : String(error)); }
    finally { setBusy(false); }
  };
  const adopt = async () => {
    if (!preview || selected.size === 0) return;
    setBusy(true); setMessage('');
    try {
      const result = await api.ownershipAdopt(preview.preview_id, [...selected]);
      setPreview(null); setSelected(new Set()); setMessage(`Adopted ${result.adopted} resource(s).`);
    } catch (error) { setMessage(error instanceof Error ? error.message : String(error)); }
    finally { setBusy(false); }
  };
  return <section className="ownership-panel">
    <header><strong>Resource ownership</strong><span>Only explicitly adopted or tool-created records may be changed.</span></header>
    <div className="ownership-controls"><select value={provider} onChange={(e) => { setProvider(e.target.value as typeof provider); setPreview(null); }}><option value="adguard">AdGuard rewrites</option><option value="cloudflare">Cloudflare ingress and DNS</option></select><button type="button" onClick={() => void load()} disabled={busy}>Preview adoption</button></div>
    <p role="status">{message}</p>
    {preview && <><div className="ownership-candidates">{preview.candidates.map((candidate) => <label key={candidate.id}><input type="checkbox" checked={selected.has(candidate.id)} onChange={(e) => setSelected((current) => { const next = new Set(current); if (e.target.checked) next.add(candidate.id); else next.delete(candidate.id); return next; })} /><span>{candidate.kind}</span><code>{candidate.id}</code><small>{candidate.current}</small></label>)}</div><button type="button" className="primary" onClick={() => void adopt()} disabled={!mutationEnabled || busy || selected.size === 0}>Adopt selected ({selected.size})</button></>}
  </section>;
}
