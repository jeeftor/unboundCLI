import { useCallback, useEffect, useRef, useState } from 'react';
import { api } from '../api/client';
import { SERVICE_META } from '../store';
import type { ConfigResponse, EntriesResponse, Entry, ProgressEvent } from '../types';

const CONFIG_CACHE_KEY = 'caddy-dns-sync:config';

function loadCachedConfig(): ConfigResponse | null {
  try {
    const raw = sessionStorage.getItem(CONFIG_CACHE_KEY);
    return raw ? JSON.parse(raw) as ConfigResponse : null;
  } catch {
    return null;
  }
}

function saveCachedConfig(cfg: ConfigResponse) {
  try {
    sessionStorage.setItem(CONFIG_CACHE_KEY, JSON.stringify(cfg));
  } catch {
    // sessionStorage might be full or unavailable — non-fatal.
  }
}

export function useRuntimeData(onDataChanged?: () => void) {
  const [config, setConfig] = useState<ConfigResponse | null>(() => loadCachedConfig());
  const [entries, setEntries] = useState<Entry[]>([]);
  const [report, setReport] = useState<EntriesResponse['report']>({});
  const [loading, setLoading] = useState(true);
  const [message, setMessage] = useState('Loading service status...');
  const [messageKind, setMessageKind] = useState<'info' | 'error' | 'ok'>('info');
  // Per-service progress: service → { status, count, error }
  const [progress, setProgress] = useState<Record<string, ProgressEvent>>({});
  const sequenceRef = useRef(0);
  const eventSourceRef = useRef<EventSource | null>(null);
  const configFetchedRef = useRef(false);

  const applyConfig = useCallback((nextConfig: ConfigResponse) => {
    setConfig(nextConfig);
    saveCachedConfig(nextConfig);
  }, []);

  const shouldHoldLoadingForE2E = useCallback(() => {
    if (window.UNBOUNDCLI_TEST_HOOKS !== true) return false;
    const script = new URLSearchParams(window.location.search).get('e2e') || '';
    return script.split(',').includes('holdloading');
  }, []);

  const refreshEntries = useCallback(() => {
    // Close any existing SSE connection.
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }

    const requestID = sequenceRef.current + 1;
    sequenceRef.current = requestID;
    // eslint-disable-next-line @eslint-react/set-state-in-effect
    setLoading(true);
    // eslint-disable-next-line @eslint-react/set-state-in-effect
    setProgress({});
    // eslint-disable-next-line @eslint-react/set-state-in-effect
    setMessage('Loading service status...');
    // eslint-disable-next-line @eslint-react/set-state-in-effect
    setMessageKind('info');

    // Fetch config in parallel via regular API (fast, cached).
    // Only fetch if we don't already have it from cache.
    const configPromise = (configFetchedRef.current
      ? Promise.resolve(null)
      : api.config().then(cfg => {
          if (requestID !== sequenceRef.current) return null;
          setConfig(cfg);
          saveCachedConfig(cfg);
          configFetchedRef.current = true;
          return cfg;
        }).catch(() => null)
    );

    // Stream entries via SSE.
    const es = new EventSource('/api/entries/stream');
    eventSourceRef.current = es;

    es.addEventListener('progress', (e: MessageEvent) => {
      if (requestID !== sequenceRef.current) return;
      try {
        const ev = JSON.parse(e.data) as ProgressEvent;
        // eslint-disable-next-line @eslint-react/set-state-in-effect
        setProgress(prev => ({ ...prev, [ev.service]: ev }));
        // Update message to show what's happening.
        const meta = SERVICE_META[ev.service];
        if (meta) {
          if (ev.status === 'loaded') {
            // eslint-disable-next-line @eslint-react/set-state-in-effect
            setMessage(`Loaded ${meta.label} (${ev.count} entries)`);
          } else if (ev.status === 'failed') {
            // eslint-disable-next-line @eslint-react/set-state-in-effect
            setMessage(`${meta.label} failed: ${ev.error || 'unknown'}`);
          }
        }
      } catch {
        // Ignore malformed progress events.
      }
    });

    es.addEventListener('done', (e: MessageEvent) => {
      if (requestID !== sequenceRef.current) return;
      try {
        const data = JSON.parse(e.data) as EntriesResponse;
        // eslint-disable-next-line @eslint-react/set-state-in-effect
        setEntries(data.entries || []);
        // eslint-disable-next-line @eslint-react/set-state-in-effect
        setReport(data.report || {});
        onDataChanged?.();
        // eslint-disable-next-line @eslint-react/set-state-in-effect
        setMessage((data.entries || []).length ? 'Loaded service status.' : 'No entries found.');
        // eslint-disable-next-line @eslint-react/set-state-in-effect
        setMessageKind('info');
      } catch {
        // eslint-disable-next-line @eslint-react/set-state-in-effect
        setMessage('Failed to parse entries response');
        // eslint-disable-next-line @eslint-react/set-state-in-effect
        setMessageKind('error');
      } finally {
        if (requestID === sequenceRef.current && !shouldHoldLoadingForE2E()) setLoading(false);
        es.close();
        eventSourceRef.current = null;
      }
    });

    es.addEventListener('error', (e: MessageEvent) => {
      // Server-sent error event (has data) vs native ES error (no data).
      if (e.data) {
        try {
          const data = JSON.parse(e.data);
          if (requestID === sequenceRef.current) {
            setMessage(data.error || 'Stream error');
            setMessageKind('error');
          }
        } catch {
          // Ignore malformed error events.
        }
        return;
      }
      // Native ES error — connection failed or server closed.
      // If we haven't received "done" yet, this is a real error.
      if (requestID === sequenceRef.current && es.readyState === EventSource.CLOSED) {
        // If we still have entries from a previous load, don't error out.
        setMessage('Connection lost — retry');
        setMessageKind('error');
        if (requestID === sequenceRef.current && !shouldHoldLoadingForE2E()) setLoading(false);
      }
      es.close();
      eventSourceRef.current = null;
    });

    // Wait for config promise (non-blocking — entries come via SSE).
    void configPromise;
  }, [onDataChanged, shouldHoldLoadingForE2E]);

  useEffect(() => {
    void refreshEntries();
    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
    };
  }, [refreshEntries]);

  return {
    config,
    entries,
    report,
    loading,
    message,
    messageKind,
    progress,
    applyConfig,
    refreshEntries
  };
}
