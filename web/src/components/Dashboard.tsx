import '../styles/Dashboard.css';
import {
  GitBranch,
  Loader2,
  RefreshCw,
  X,
} from 'lucide-react';
import { useCallback, useEffect, useState } from 'react';
import { api } from '../api/client';
import { AuthFlowsTab } from './AuthFlowsTab';
import { CaddyEditor } from './CaddyEditor';
import { CFRepairBanner } from './CloudflarePanel';
import { ConfigModal } from './ConfigModal';
import { DiagnosticsTab } from './DiagnosticsTab';
import { EntriesTable } from './EntriesTable';
import { EntriesToolbar } from './EntriesToolbar';
import { LogBar } from './LogBar';
import { MetricGrid } from './MetricCards';
import { OperationsHeader } from './OperationsHeader';
import { SyncModal } from './SyncModal';
import { Topbar } from './Topbar';
import type { TabId } from './TabBar';
import type { ConfigForms } from '../hooks/useConfigForms';
import {
  useStore,
  selectFilteredEntries,
  selectSummary,
  selectSelectedEntry,
  selectCanSyncNow,
  selectPlannedActions,
  selectEnabledServices,
  refreshEntries,
  previewSync,
  dryRunSync,
  syncNow,
  removeEntry,
  syncAll,
  saveConfig,
  saveCaddyEditor,
  testConfig,
} from '../store';

export function AppShell() {
  // ── Selectors (each subscribes to a slice of state) ──
  const view = useStore((s) => s.view);
  const config = useStore((s) => s.config);
  const loading = useStore((s) => s.loading);
  const message = useStore((s) => s.message);
  const messageKind = useStore((s) => s.messageKind);
  const progress = useStore((s) => s.progress);
  const mutationEnabled = useStore((s) => s.mutationEnabled);
  const syncService = useStore((s) => s.syncService);
  const syncLoading = useStore((s) => s.syncLoading);
  const syncProgress = useStore((s) => s.syncProgress);
  const syncLog = useStore((s) => s.syncLog);
  const configOpen = useStore((s) => s.configOpen);
  const forms = useStore((s) => s.forms);
  const savedForms = useStore((s) => s.savedForms);
  const configStatus = useStore((s) => s.configStatus);
  const configStatusKind = useStore((s) => s.configStatusKind);
  const testResults = useStore((s) => s.testResults);
  const statusFilter = useStore((s) => s.statusFilter);
  const serviceFilter = useStore((s) => s.serviceFilter);
  const search = useStore((s) => s.search);
  const selectedHostname = useStore((s) => s.selectedHostname);
  const suppressed = useStore((s) => s.suppressed);

  // ── Derived selectors ──
  const entries = useStore(selectFilteredEntries);
  const summary = useStore(selectSummary);
  const selectedEntry = useStore(selectSelectedEntry);
  const canSyncNow = useStore(selectCanSyncNow);
  const plannedActions = useStore(selectPlannedActions);
  const enabledServices = useStore(selectEnabledServices);

  // ── Actions from store ──
  const setView = useStore((s) => s.setView);
  const setConfigOpen = useStore((s) => s.setConfigOpen);
  const setStatusFilter = useStore((s) => s.setStatusFilter);
  const setServiceFilter = useStore((s) => s.setServiceFilter);
  const setSearch = useStore((s) => s.setSearch);
  const setSelectedHostname = useStore((s) => s.setSelectedHostname);
  const toggleSuppress = useStore((s) => s.toggleSuppress);
  const setSyncService = useStore((s) => s.setSyncService);
  const setForms = useStore((s) => s.setForms);
  const setModalOpen = useStore((s) => s.setModalOpen);
  const setModalHostname = useStore((s) => s.setModalHostname);
  const setModalAutoSync = useStore((s) => s.setModalAutoSync);
  const modalOpen = useStore((s) => s.modalOpen);
  const modalHostname = useStore((s) => s.modalHostname);
  const modalAutoSync = useStore((s) => s.modalAutoSync);
  const logBarOpen = useStore((s) => s.logBarOpen);
  const setLogBarOpen = useStore((s) => s.setLogBarOpen);

  // ── Local UI state (not shared) ──
  const [remoteStatus, setRemoteStatus] = useState<{
    remote_ahead: number; local_ahead: number; branch: string; remote: string; fetch_error?: string;
  } | null>(null);
  const [remoteChecking, setRemoteChecking] = useState(false);
  const [pulling, setPulling] = useState(false);
  const [pullOutput, setPullOutput] = useState('');

  // ── Git remote polling ──
  const checkRemote = useCallback(async () => {
    setRemoteChecking(true);
    try { setRemoteStatus(await api.caddyGitStatus()); } catch { /* non-fatal */ } finally { setRemoteChecking(false); }
  }, []);

  const handlePull = useCallback(async () => {
    setPulling(true); setPullOutput('');
    try {
      const res = await api.caddyGitPull();
      setPullOutput(res.output || 'Already up to date.');
      await checkRemote();
      await refreshEntries();
    } catch (err) { setPullOutput(`Error: ${String(err)}`); }
    finally { setPulling(false); }
  }, [checkRemote]);

  useEffect(() => {
    void checkRemote();
    const id = setInterval(() => { void checkRemote(); }, 60_000);
    return () => clearInterval(id);
  }, [checkRemote]);

  // ── Modal helpers ──
  const modalEntry = useStore((s) => s.entries.find((e) => e.hostname === modalHostname));

  const openModify = (hostname: string) => {
    setModalHostname(hostname);
    setSelectedHostname(hostname);
    setModalAutoSync(false);
    setModalOpen(true);
  };

  const openQuickSync = (hostname: string) => {
    setModalHostname(hostname);
    setSelectedHostname(hostname);
    setModalAutoSync(true);
    setModalOpen(true);
  };

  const showRemoteBanner = remoteStatus && (remoteStatus.remote_ahead > 0 || remoteStatus.fetch_error);

  return (
    <div className="console-layout">
      <div className="console-main">
        <Topbar
          config={config}
          loading={loading}
          syncLoading={syncLoading}
          view={view as TabId}
          setView={(v) => setView(v)}
          onRefresh={() => void refreshEntries()}
          onOpenConfig={() => setConfigOpen(true)}
        />

        {showRemoteBanner && (
          <div className={`git-remote-banner global ${remoteStatus!.fetch_error ? 'error' : 'warn'}`}>
            <GitBranch size={14} />
            {remoteStatus!.fetch_error ? (
              <span>Caddy repo: could not reach remote — {remoteStatus!.fetch_error}</span>
            ) : (
              <span>
                Caddy repo: remote <strong>{remoteStatus!.remote}/{remoteStatus!.branch}</strong> is{' '}
                <strong>{remoteStatus!.remote_ahead}</strong> commit{remoteStatus!.remote_ahead !== 1 ? 's' : ''} ahead — pull before deploying
              </span>
            )}
            <div className="git-remote-banner-actions">
              {remoteStatus!.remote_ahead > 0 && mutationEnabled && (
                <button type="button" className="btn-sm btn-warn" onClick={() => void handlePull()} disabled={pulling}>
                  {pulling ? <><Loader2 size={12} className="spin" /> Pulling…</> : '⬇ Pull'}
                </button>
              )}
              <button type="button" className="btn-sm" onClick={() => void checkRemote()} disabled={remoteChecking}>
                {remoteChecking ? <Loader2 size={12} className="spin" /> : <RefreshCw size={12} />}
              </button>
            </div>
          </div>
        )}

        {pullOutput && (
          <div className="git-pull-output global">
            <pre>{pullOutput}</pre>
            <button type="button" className="git-pull-dismiss" onClick={() => setPullOutput('')}><X size={13} /></button>
          </div>
        )}

        <div className="console-scroll">
          {view === 'caddyfile' ? (
            <main className="dashboard-shell caddy-editor-shell">
              <CaddyEditor
                mutationEnabled={mutationEnabled}
                remoteStatus={remoteStatus}
                remoteChecking={remoteChecking}
                pulling={pulling}
                onPull={handlePull}
                onCheckRemote={checkRemote}
              />
            </main>
          ) : view === 'auth' ? (
            <AuthFlowsTab />
          ) : view === 'diagnostics' ? (
            <DiagnosticsTab />
          ) : (
            <main className="dashboard-shell">
              <OperationsHeader
                loading={loading}
                message={message}
                messageKind={messageKind}
                progress={progress}
                summary={summary}
              />
              <MetricGrid summary={summary} statusFilter={statusFilter} setStatusFilter={setStatusFilter} />
              <EntriesToolbar
                entriesCount={entries.length}
                statusFilter={statusFilter}
                setStatusFilter={setStatusFilter}
                serviceFilter={serviceFilter}
                setServiceFilter={setServiceFilter}
                search={search}
                setSearch={setSearch}
                enabledServices={enabledServices}
                mutationEnabled={mutationEnabled}
                onSyncAll={() => syncAll()}
              />
              <CFRepairBanner
                entries={entries}
                mutationEnabled={mutationEnabled}
                cfEnabled={enabledServices.cloudflare === true}
                onRepaired={() => void refreshEntries()}
              />
              <EntriesTable
                entries={entries}
                selectedHostname={selectedHostname}
                mutationEnabled={mutationEnabled}
                enabledServices={enabledServices}
                caddyServerIP={config?.caddy?.server_ip || ''}
                suppressed={suppressed}
                onToggleSuppress={toggleSuppress}
                onSelect={setSelectedHostname}
                onQuickSync={openQuickSync}
                onOpenModify={openModify}
                onRemove={(hostname, service) => removeEntry(hostname, service as 'all' | 'unbound' | 'adguard')}
              />
            </main>
          )}
        </div>
        <LogBar open={logBarOpen} onToggle={() => setLogBarOpen(!logBarOpen)} />
      </div>
      <SyncModal
        open={modalOpen}
        autoSync={modalAutoSync}
        onClose={() => setModalOpen(false)}
        hostname={modalHostname}
        entry={modalEntry}
        enabledServices={enabledServices}
        caddyServerIP={config?.caddy?.server_ip || ''}
        suppressed={suppressed}
        onToggleSuppress={toggleSuppress}
        syncService={syncService}
        setSyncService={setSyncService}
        syncLoading={syncLoading}
        syncProgress={syncProgress}
        syncLog={syncLog}
        plannedActions={plannedActions}
        canSyncNow={canSyncNow}
        mutationEnabled={mutationEnabled}
        onPreviewFor={(service, hostname) => previewSync(service, hostname)}
        onDryRun={() => dryRunSync()}
        onSync={() => syncNow()}
        onRefresh={() => void refreshEntries()}
        onRemoveEntry={(hostname, service) => removeEntry(hostname, service as 'all' | 'unbound' | 'adguard')}
      />
      <ConfigModal
        open={configOpen}
        onClose={() => setConfigOpen(false)}
        config={config}
        forms={forms}
        setForms={(updater) => {
          if (typeof updater === 'function') {
            setForms((updater as (prev: ConfigForms) => ConfigForms)(useStore.getState().forms));
          } else {
            setForms(updater);
          }
        }}
        savedForms={savedForms}
        mutationEnabled={mutationEnabled}
        status={configStatus}
        statusKind={configStatusKind}
        testResults={testResults}
        onSave={(service) => saveConfig(service)}
        onSaveCaddyEditor={() => saveCaddyEditor()}
        onTest={(service) => testConfig(service)}
      />
    </div>
  );
}
