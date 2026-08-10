import '../styles/Dashboard.css';
import {
  GitBranch,
  RefreshCw,
  X,
} from 'lucide-react';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { api } from '../api/client';
import { isIssue } from '../lib/hostnameDecision';
import type { ServiceKey } from '../types';
import { AuthFlowsTab } from './AuthFlowsTab';
import { CaddyEditor } from './CaddyEditor';
import { CFRepairBanner } from './CloudflarePanel';
import { ConfigModal } from './ConfigModal';
import { LoadingSpinner } from './LoadingSpinner';
import { DiagnosticsTab } from './DiagnosticsTab';
import { EntriesTable } from './EntriesTable';
import { EntriesToolbar } from './EntriesToolbar';
import { LogBar } from './LogBar';
import { MetricGrid } from './MetricCards';
import { OperationsHeader } from './OperationsHeader';
import { SyncModal } from './SyncModal';
import { Topbar } from './Topbar';
import { VisualizeModal } from './VisualizeModal';
import type { TabId } from './TabBar';
import {
  useStore,
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
  const plan = useStore((s) => s.plan);
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

  // ── Derived (useMemo, not store selectors, to avoid infinite re-render) ──
  const allEntries = useStore((s) => s.entries);
  const caddyServerIP = config?.caddy?.server_ip ?? '';
  const entries = useMemo(
    () => allEntries.filter((entry) => {
      if (statusFilter === 'synced' && entry.overall_status !== 0 && entry.overall_status !== 1) return false;
      if (statusFilter === 'out_of_sync' && entry.overall_status !== 2) return false;
      if (statusFilter === 'caddy_only' && entry.overall_status !== 3) return false;
      if (statusFilter === 'stale' && entry.overall_status !== 4) return false;
      if (statusFilter === 'cloudflare' && !entry.cloudflare_status?.configured) return false;
      if (statusFilter === 'issues' && !isIssue(entry, caddyServerIP, suppressed)) return false;
      if (serviceFilter === 'unbound' && !entry.unbound_status?.configured) return false;
      if (serviceFilter === 'adguard' && !entry.adguard_status?.configured) return false;
      if (serviceFilter === 'dhcp' && !entry.dhcp_status?.configured) return false;
      if (serviceFilter === 'cloudflare' && !entry.cloudflare_status?.configured) return false;
      return !search.trim() || entry.hostname.toLowerCase().includes(search.trim().toLowerCase());
    }),
    [allEntries, statusFilter, serviceFilter, search, suppressed, caddyServerIP]
  );
  const summary = useMemo(() => ({
    entries: allEntries.length,
    inSync: allEntries.filter((e) => e.overall_status === 0 || e.overall_status === 1).length,
    out: allEntries.filter((e) => e.overall_status === 2).length,
    caddyOnly: allEntries.filter((e) => e.overall_status === 3).length,
    stale: allEntries.filter((e) => e.overall_status === 4).length,
    cloudflare: allEntries.filter((e) => e.cloudflare_status?.configured).length,
    issues: allEntries.filter((e) => isIssue(e, caddyServerIP, suppressed)).length,
  }), [allEntries, caddyServerIP, suppressed]);
  const canSyncNow = mutationEnabled && plan.planID !== '' && plan.actionIDs.length > 0;
  const plannedActions = plan.actions;
  const enabledServices = config?.enabled ?? ({} as Partial<Record<ServiceKey, boolean>>);

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
  const visualizeOpen = useStore((s) => s.visualizeOpen);
  const visualizeHostname = useStore((s) => s.visualizeHostname);
  const setVisualizeOpen = useStore((s) => s.setVisualizeOpen);
  const setVisualizeHostname = useStore((s) => s.setVisualizeHostname);
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
      void refreshEntries();
    } catch (err) { setPullOutput(`Error: ${String(err)}`); }
    finally { setPulling(false); }
  }, [checkRemote]);

  useEffect(() => {
    void checkRemote();
    const id = setInterval(() => { void checkRemote(); }, 60_000);
    return () => clearInterval(id);
  }, [checkRemote]);

  // ── Modal helpers ──
  const allEntriesForModal = useStore((s) => s.entries);
  const modalEntry = useMemo(
    () => allEntriesForModal.find((e) => e.hostname === modalHostname),
    [allEntriesForModal, modalHostname]
  );

  // ── Deep link: open visualize modal via #visualize=hostname ──
  useEffect(() => {
    const hash = window.location.hash;
    const match = hash.match(/^#visualize=(.+)$/);
    if (!match) return;
    const hostname = decodeURIComponent(match[1]);
    // Wait until entries are loaded
    if (allEntriesForModal.length === 0) return;
    const entry = allEntriesForModal.find((e) => e.hostname === hostname);
    if (entry) {
      setVisualizeHostname(hostname);
      setSelectedHostname(hostname);
      setVisualizeOpen(true);
      // Clear hash so refresh doesn't re-open
      history.replaceState(null, '', window.location.pathname);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps, @eslint-react/exhaustive-deps
  }, [allEntriesForModal, setVisualizeHostname, setVisualizeOpen]);

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

  const openVisualize = (hostname: string) => {
    setVisualizeHostname(hostname);
    setSelectedHostname(hostname);
    setVisualizeOpen(true);
  };

  const visualizeEntry = useMemo(
    () => allEntriesForModal.find((e) => e.hostname === visualizeHostname),
    [allEntriesForModal, visualizeHostname]
  );

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
          <div className={`git-remote-banner global ${remoteStatus.fetch_error ? 'error' : 'warn'}`}>
            <GitBranch size={14} />
            {remoteStatus.fetch_error ? (
              <span>Caddy repo: could not reach remote — {remoteStatus.fetch_error}</span>
            ) : (
              <span>
                Caddy repo: remote <strong>{remoteStatus.remote}/{remoteStatus.branch}</strong> is{' '}
                <strong>{remoteStatus.remote_ahead}</strong> commit{remoteStatus.remote_ahead !== 1 ? 's' : ''} ahead — pull before deploying
              </span>
            )}
            <div className="git-remote-banner-actions">
              {remoteStatus.remote_ahead > 0 && mutationEnabled && (
                <button type="button" className="btn-sm btn-warn" onClick={() => void handlePull()} disabled={pulling}>
                  {pulling ? <><LoadingSpinner size={12} /> Pulling…</> : '⬇ Pull'}
                </button>
              )}
              <button type="button" className="btn-sm" onClick={() => void checkRemote()} disabled={remoteChecking}>
                {remoteChecking ? <LoadingSpinner size={12} /> : <RefreshCw size={12} />}
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
                onOpenVisualize={openVisualize}
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
      {visualizeOpen && visualizeEntry && (
        <VisualizeModal entry={visualizeEntry} onClose={() => setVisualizeOpen(false)} />
      )}
      <ConfigModal
        open={configOpen}
        onClose={() => setConfigOpen(false)}
        config={config}
        forms={forms}
        setForms={(updater) => {
          if (typeof updater === 'function') {
            setForms((updater)(useStore.getState().forms));
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
