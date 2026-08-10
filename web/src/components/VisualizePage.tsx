import '../styles/VisualizeModal.css';
import { useEffect, useState } from 'react';
import { ArrowLeft, Loader2 } from 'lucide-react';
import { useStore } from '../store';
import { api } from '../api/client';
import type { Entry } from '../types';
import { VisualizeContent } from './VisualizeModal';

export function VisualizePage({ hostname }: { hostname: string }) {
  const [entry, setEntry] = useState<Entry | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Auth data from store
  const auth = useStore((s) => s.authHosts.get(hostname) ?? null);
  const authLoading = useStore((s) => s.authLoading);

  useEffect(() => {
    void useStore.getState().refreshAuth();
    void (async () => {
      try {
        const resp = await api.entries();
        const found = resp.entries.find((e) => e.hostname === hostname);
        if (!found) {
          setError(`Host "${hostname}" not found`);
          return;
        }
        setEntry(found);
      } catch (e) {
        setError(String(e));
      }
    })();
  }, [hostname]);

  if (error) {
    return (
      <div className="visualize-page-error">
        <p>{error}</p>
        <a href="/">← Back to dashboard</a>
      </div>
    );
  }

  if (!entry) {
    return (
      <div className="visualize-page-loading">
        <Loader2 size={24} className="visualize-loading-spin" />
        <p>Loading {hostname}…</p>
      </div>
    );
  }

  return (
    <div className="visualize-page">
      <div className="visualize-page-header">
        <a href="/" className="visualize-back-link">
          <ArrowLeft size={16} /> Dashboard
        </a>
      </div>
      <div className="visualize-page-content">
        <VisualizeContent entry={entry} auth={auth} authLoading={authLoading} />
      </div>
    </div>
  );
}
