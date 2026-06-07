import { useCallback, useEffect, useRef, useState } from 'react';
import { AppShell } from './components/Dashboard';
import { useConfigForms } from './hooks/useConfigForms';
import { useEntryFilters } from './hooks/useEntryFilters';
import { useRuntimeData } from './hooks/useRuntimeData';
import { useSyncPlan } from './hooks/useSyncPlan';
import type { ServiceKey } from './types';

export function App() {
  const [configOpen, setConfigOpen] = useState(false);
  const [mobile, setMobile] = useState(false);
  const [tableScrolls, setTableScrolls] = useState(false);
  const e2eRan = useRef(false);
  const clearPlanRef = useRef<(() => void) | null>(null);

  const clearPlanForDataChange = useCallback(() => {
    clearPlanRef.current?.();
  }, []);

  const runtime = useRuntimeData(clearPlanForDataChange);
  const mutationEnabled = Boolean(runtime.config?.mutation_enabled && window.UNBOUNDCLI_WEB_CONFIG?.mutationEnabled);
  const filters = useEntryFilters(runtime.entries);
  const sync = useSyncPlan({
    config: runtime.config,
    mutationEnabled,
    refreshEntries: runtime.refreshEntries
  });
  const configForms = useConfigForms({
    config: runtime.config,
    mutationEnabled,
    applyConfig: runtime.applyConfig,
    onConfigChanged: sync.clearPlan
  });

  useEffect(() => {
    clearPlanRef.current = () => sync.clearPlan();
  }, [sync]);

  useEffect(() => {
    sync.clearPlan();
  }, [filters.selectedHostname, sync.clearPlan]);

  useEffect(() => {
    const updateResponsive = () => {
      setMobile(window.innerWidth <= 760);
      const panel = document.getElementById('entries-panel');
      setTableScrolls(Boolean(panel && panel.scrollWidth > panel.clientWidth));
    };
    updateResponsive();
    window.addEventListener('resize', updateResponsive);
    return () => window.removeEventListener('resize', updateResponsive);
  }, [runtime.entries, filters.filteredEntries]);

  useEffect(() => {
    if (!configOpen) return undefined;
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setConfigOpen(false);
    };
    window.addEventListener('keydown', closeOnEscape);
    return () => window.removeEventListener('keydown', closeOnEscape);
  }, [configOpen]);

  useEffect(() => {
    if (!runtime.config || e2eRan.current || window.UNBOUNDCLI_TEST_HOOKS !== true) return;
    e2eRan.current = true;
    const script = new URLSearchParams(window.location.search).get('e2e');
    if (!script) return;
    const run = async () => {
      for (const action of script.split(',')) {
        const [name, ...parts] = action.split(':');
        const value = parts.join(':');
        if (name === 'filter') filters.setStatusFilter(value);
        if (name === 'search') filters.setSearch(value);
        if (name === 'preview') {
          sync.setSyncService(value);
          await sync.previewSync(value);
        }
        if (name === 'rowpreview') {
          const [hostname, service = 'unbound'] = parts;
          filters.setSelectedHostname(hostname);
          await sync.previewSync(service, hostname);
        }
        if (name === 'dryrun') await sync.dryRunSync();
        if (name === 'sync') await sync.syncNow();
        if (name === 'toggleconfig') setConfigOpen(value !== 'closed');
        if (name === 'setconfig' && value === 'unbound') {
          const nextForms = {
            ...configForms.forms,
            unbound: { ...configForms.forms.unbound, base_url: 'https://saved.example.test', api_key: 'saved-key' }
          };
          configForms.setForms(nextForms);
          await configForms.saveConfig('unbound', nextForms);
        }
        if (name === 'testconfig') await configForms.testConfig(value as ServiceKey);
      }
      document.getElementById('app')?.setAttribute('data-e2e', 'done');
    };
    void run();
  }, [configForms, filters, runtime.config, sync]);

  return (
    <div
      id="app"
      data-loading={String(runtime.loading)}
      data-preview-loading={String(sync.syncLoading)}
      data-dry-run-enabled={String(sync.plannedActions.length > 0)}
      data-sync-enabled={String(sync.canSyncNow)}
      data-unbound-enabled={String(runtime.config?.enabled?.unbound !== false)}
      data-adguard-enabled={String(runtime.config?.enabled?.adguard !== false)}
      data-mutation-enabled={String(mutationEnabled)}
      data-mobile={String(mobile)}
      data-table-scrolls={String(tableScrolls)}
    >
      <AppShell
        config={runtime.config}
        loading={runtime.loading}
        message={runtime.message}
        messageKind={runtime.messageKind}
        report={runtime.report}
        summary={filters.summary}
        statusFilter={filters.statusFilter}
        setStatusFilter={filters.setStatusFilter}
        serviceFilter={filters.serviceFilter}
        setServiceFilter={filters.setServiceFilter}
        search={filters.search}
        setSearch={filters.setSearch}
        entries={filters.filteredEntries}
        selectedEntry={filters.selectedEntry}
        selectedHostname={filters.selectedHostname}
        setSelectedHostname={filters.setSelectedHostname}
        mutationEnabled={mutationEnabled}
        syncService={sync.syncService}
        setSyncService={sync.setSyncService}
        syncLoading={sync.syncLoading}
        syncProgress={sync.syncProgress}
        syncLog={sync.syncLog}
        plannedActions={sync.plannedActions}
        canSyncNow={sync.canSyncNow}
        onRefresh={() => void runtime.refreshEntries()}
        onPreview={sync.previewSync}
        onDryRun={sync.dryRunSync}
        onSync={sync.syncNow}
        configOpen={configOpen}
        setConfigOpen={setConfigOpen}
        forms={configForms.forms}
        setForms={configForms.setForms}
        configStatus={configForms.configStatus}
        configStatusKind={configForms.configStatusKind}
        testResults={configForms.testResults}
        onSaveConfig={configForms.saveConfig}
        onTestConfig={configForms.testConfig}
      />
    </div>
  );
}
