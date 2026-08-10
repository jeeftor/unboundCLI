import { lazy, Suspense, useEffect, useRef } from 'react';
import { AppShell } from './components/Dashboard';
import { LoadingSpinner } from './components/LoadingSpinner';
import { tabFromHash } from './components/TabBar';
import { useStore, refreshEntries, syncFormsFromConfig, previewSync, dryRunSync, syncNow, saveConfig, testConfig } from './store';

const VisualizePage = lazy(() => import('./components/VisualizePage').then(m => ({ default: m.VisualizePage })));

export function App() {
  const config = useStore((s) => s.config);
  const configOpen = useStore((s) => s.configOpen);
  const loading = useStore((s) => s.loading);
  const syncLoading = useStore((s) => s.syncLoading);
  const plan = useStore((s) => s.plan);
  const mutationEnabled = useStore((s) => s.mutationEnabled);
  const mobile = useStore((s) => s.mobile);
  const tableScrolls = useStore((s) => s.tableScrolls);
  const entries = useStore((s) => s.entries);
  const selectedHostname = useStore((s) => s.selectedHostname);

  const setConfigOpen = useStore((s) => s.setConfigOpen);
  const setMobile = useStore((s) => s.setMobile);
  const setTableScrolls = useStore((s) => s.setTableScrolls);
  const e2eRanRef = useRef(false);

  useEffect(() => {
    const onHashChange = () => useStore.getState().setView(tabFromHash());
    window.addEventListener('hashchange', onHashChange);
    return () => window.removeEventListener('hashchange', onHashChange);
  }, []);

  // Initial data load.
  useEffect(() => {
    void refreshEntries(() => useStore.getState().clearPlan());
    void useStore.getState().refreshAuth();
  }, []);

  // Sync forms when config changes.
  useEffect(() => {
    syncFormsFromConfig(config);
  }, [config]);

  // Clear plan when selected hostname changes.
  useEffect(() => {
    useStore.getState().clearPlan();
  }, [selectedHostname]);

  // Responsive detection.
  useEffect(() => {
    const updateResponsive = () => {
      setMobile(window.innerWidth <= 760);
      const panel = document.getElementById('entries-panel');
      setTableScrolls(Boolean(panel && panel.scrollWidth > panel.clientWidth));
    };
    updateResponsive();
    window.addEventListener('resize', updateResponsive);
    return () => window.removeEventListener('resize', updateResponsive);
  }, [entries, setMobile, setTableScrolls]);

  // Close config modal on Escape.
  useEffect(() => {
    if (!configOpen) return undefined;
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setConfigOpen(false);
    };
    window.addEventListener('keydown', closeOnEscape);
    return () => window.removeEventListener('keydown', closeOnEscape);
  }, [configOpen, setConfigOpen]);

  // E2E test hooks.
  useEffect(() => {
    if (!config || e2eRanRef.current || window.UNBOUNDCLI_TEST_HOOKS !== true) return;
    e2eRanRef.current = true;
    const script = new URLSearchParams(window.location.search).get('e2e');
    if (!script) return;
    const run = async () => {
      for (const action of script.split(',')) {
        const [name, ...parts] = action.split(':');
        const value = parts.join(':');
        const store = useStore.getState();
        if (name === 'filter') store.setStatusFilter(value);
        if (name === 'search') store.setSearch(value);
        if (name === 'preview') {
          store.setSyncService(value);
          await previewSync(value);
        }
        if (name === 'rowpreview') {
          const [hostname, service = 'unbound'] = parts;
          store.setSelectedHostname(hostname);
          await previewSync(service, hostname);
        }
        if (name === 'dryrun') await dryRunSync();
        if (name === 'sync') await syncNow();
        if (name === 'toggleconfig') setConfigOpen(value !== 'closed');
        if (name === 'setconfig' && value === 'unbound') {
          const nextForms = {
            ...useStore.getState().forms,
            unbound: { ...useStore.getState().forms.unbound, base_url: 'https://saved.example.test', api_key: 'saved-key' },
          };
          useStore.getState().setForms(nextForms);
          await saveConfig('unbound');
        }
        if (name === 'testconfig') await testConfig(value as never);
      }
      document.getElementById('app')?.setAttribute('data-e2e', 'done');
    };
    void run();
  }, [config, setConfigOpen]);

  // ── Standalone visualize page (/visualize/hostname) ──
  const visualizeMatch = typeof window !== 'undefined'
    ? window.location.pathname.match(/^\/visualize\/(.+)$/)
    : null;

  if (visualizeMatch) {
    return (
      <div id="app" data-visualize-page="true">
        <Suspense fallback={<LoadingSpinner />}>
          <VisualizePage hostname={decodeURIComponent(visualizeMatch[1])} />
        </Suspense>
      </div>
    );
  }

  return (
    <div
      id="app"
      data-loading={String(loading)}
      data-preview-loading={String(syncLoading)}
      data-dry-run-enabled={String(plan.actions.length > 0)}
      data-sync-enabled={String(plan.planID !== '' && plan.actionIDs.length > 0)}
      data-unbound-enabled={String(config?.enabled?.unbound !== false)}
      data-adguard-enabled={String(config?.enabled?.adguard !== false)}
      data-mutation-enabled={String(mutationEnabled)}
      data-mobile={String(mobile)}
      data-table-scrolls={String(tableScrolls)}
    >
      <AppShell />
    </div>
  );
}
