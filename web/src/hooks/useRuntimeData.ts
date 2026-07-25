import { useCallback, useEffect, useRef, useState } from 'react';
import { api } from '../api/client';
import type { ConfigResponse, EntriesResponse, Entry } from '../types';

export function useRuntimeData(onDataChanged?: () => void) {
  const [config, setConfig] = useState<ConfigResponse | null>(null);
  const [entries, setEntries] = useState<Entry[]>([]);
  const [report, setReport] = useState<EntriesResponse['report']>({});
  const [loading, setLoading] = useState(true);
  const [message, setMessage] = useState('Loading service status...');
  const [messageKind, setMessageKind] = useState<'info' | 'error' | 'ok'>('info');
  const sequence = useRef(0);
  const abortRef = useRef<AbortController | null>(null);

  const applyConfig = useCallback((nextConfig: ConfigResponse) => {
    setConfig(nextConfig);
  }, []);

  const shouldHoldLoadingForE2E = useCallback(() => {
    if (window.UNBOUNDCLI_TEST_HOOKS !== true) return false;
    const script = new URLSearchParams(window.location.search).get('e2e') || '';
    return script.split(',').includes('holdloading');
  }, []);

  const refreshEntries = useCallback(async () => {
    // Abort any in-flight request
    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;

    const requestID = sequence.current + 1;
    sequence.current = requestID;
    setLoading(true);
    setMessage('Loading service status...');
    setMessageKind('info');
    try {
      const [nextConfig, data] = await Promise.all([
        api.config(controller.signal),
        api.entries(controller.signal),
      ]);
      if (requestID !== sequence.current) return;
      setConfig(nextConfig);
      setEntries(data.entries || []);
      setReport(data.report || {});
      onDataChanged?.();
      setMessage((data.entries || []).length ? 'Loaded service status.' : 'No entries found.');
      setMessageKind('info');
    } catch (err) {
      if (controller.signal.aborted) return;
      if (requestID !== sequence.current) return;
      const text = err instanceof Error ? err.message : String(err);
      setMessage(text);
      setMessageKind('error');
    } finally {
      if (requestID === sequence.current && !shouldHoldLoadingForE2E()) setLoading(false);
    }
  }, [onDataChanged, shouldHoldLoadingForE2E]);

  useEffect(() => {
    void refreshEntries();
    return () => {
      abortRef.current?.abort();
    };
  }, [refreshEntries]);

  return {
    config,
    entries,
    report,
    loading,
    message,
    messageKind,
    applyConfig,
    refreshEntries
  };
}
