import type { ReactNode } from 'react';

export function InlineProgress({ loading, title, detail }: { loading: boolean; title: string; detail: string }) {
  return (
    <div id="sync-progress" className="inline-progress" role="status" aria-live="polite" aria-label={title} hidden={!loading}>
      <div className="loading-copy compact">
        <span id="sync-progress-title">{title}</span>
        <strong>Working</strong>
      </div>
      <div className="progress-track"><span /></div>
      <small id="sync-progress-detail">{detail}</small>
    </div>
  );
}

export function Field({ label, children }: { label: string; children: ReactNode }) {
  return <label>{label}{children}</label>;
}
