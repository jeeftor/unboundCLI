import { useCallback, useRef, useState } from 'react';
import { api } from '../api/client';
import { renderActionLine, serviceEnabled } from '../lib/services';
import type { ConfigResponse, ServiceKey, SyncAction } from '../types';

type PlanState = {
  actions: SyncAction[];
  actionIDs: string[];
  planID: string;
  service: string;
  hostname: string;
};

const emptyPlan: PlanState = { actions: [], actionIDs: [], planID: '', service: '', hostname: '' };

export function useSyncPlan(args: {
  config: ConfigResponse | null;
  mutationEnabled: boolean;
  refreshEntries: () => Promise<void>;
}) {
  const [syncService, setSyncServiceState] = useState('all');
  const [syncLoading, setSyncLoading] = useState(false);
  const [syncProgress, setSyncProgress] = useState({ title: 'Planning sync', detail: 'Building a server-issued action plan...' });
  const [syncLog, setSyncLog] = useState('No actions planned.');
  const [plan, setPlan] = useState<PlanState>(emptyPlan);
  const planRef = useRef<PlanState>(emptyPlan);
  const syncServiceRef = useRef('all');

  const enabledServices: Partial<Record<ServiceKey, boolean>> = args.config?.enabled || {};
  const canSyncNow = args.mutationEnabled && plan.planID !== '' && plan.actionIDs.length > 0;

  const clearPlan = useCallback((message?: string) => {
    planRef.current = emptyPlan;
    setPlan(emptyPlan);
    if (message) setSyncLog(message);
  }, []);

  const setSyncService = useCallback((service: string) => {
    syncServiceRef.current = service;
    setSyncServiceState(service);
    clearPlan('No actions planned.');
  }, [clearPlan]);

  const renderActions = useCallback((actions: SyncAction[], title = 'Planned actions') => {
    if (!actions.length) {
      setSyncLog('No actions needed.');
      return;
    }
    setSyncLog(`${title}\n${actions.map(renderActionLine).join('\n')}`);
  }, []);

  const previewSync = useCallback(async (service = syncServiceRef.current, hostname = '') => {
    if (service === 'dhcp') {
      clearPlan('DHCP apply is not implemented; preview only.');
      return false;
    }
    if (!serviceEnabled(enabledServices, service)) {
      clearPlan(`${service} is not available in this web session.`);
      return false;
    }
    setSyncLoading(true);
    setSyncProgress({
      title: hostname ? 'Planning selected host' : 'Planning sync',
      detail: hostname ? `Checking available actions for ${hostname}...` : 'Checking Caddy entries against available DNS targets...'
    });
    setSyncLog(hostname ? `Planning ${service} sync for ${hostname}...` : `Planning ${service} sync...`);
    try {
      const data = await api.planSync(service, hostname);
      const nextPlan = {
        actions: data.actions || [],
        actionIDs: data.action_ids || [],
        planID: data.plan_id || '',
        service,
        hostname
      };
      planRef.current = nextPlan;
      setPlan(nextPlan);
      renderActions(nextPlan.actions, hostname ? `Planned actions for ${hostname}` : 'Planned actions');
      return nextPlan.actionIDs.length > 0;
    } catch (err) {
      clearPlan(err instanceof Error ? err.message : String(err));
      return false;
    } finally {
      setSyncLoading(false);
    }
  }, [clearPlan, enabledServices, renderActions]);

  const dryRunSync = useCallback(async () => {
    const activePlan = planRef.current;
    const activeService = activePlan.service || syncServiceRef.current;
    if (activeService === 'dhcp') {
      setSyncLog('DHCP apply is not implemented.');
      return;
    }
    if (!activePlan.actions.length) {
      setSyncLog('Preview sync before running a dry run.');
      return;
    }
    if (!activePlan.hostname && activePlan.service !== syncServiceRef.current) {
      clearPlan('Preview sync again for the selected target.');
      return;
    }
    setSyncLoading(true);
    setSyncProgress({ title: 'Dry-running plan', detail: 'Simulating changes without writing DNS records...' });
    try {
      const data = await api.applySync({ dry_run: true, actions: activePlan.actions });
      setSyncLog(`${data.result.message}\nadded=${data.result.items_added} updated=${data.result.items_updated} deleted=${data.result.items_deleted}`);
    } catch (err) {
      setSyncLog(err instanceof Error ? err.message : String(err));
    } finally {
      setSyncLoading(false);
    }
  }, [clearPlan]);

  const syncNow = useCallback(async () => {
    const activePlan = planRef.current;
    if (!args.mutationEnabled || !activePlan.planID || !activePlan.actionIDs.length) {
      setSyncLog(args.mutationEnabled ? 'Preview sync before syncing.' : 'Sync is unavailable for this web session.');
      return;
    }
    setSyncLoading(true);
    setSyncProgress({
      title: activePlan.hostname ? 'Syncing selected host' : 'Applying sync plan',
      detail: activePlan.hostname ? `Applying DNS updates for ${activePlan.hostname}...` : 'Applying server-issued DNS updates...'
    });
    setSyncLog(activePlan.hostname ? `Syncing ${activePlan.hostname}...` : 'Syncing planned actions...');
    try {
      const data = await api.applySync({
        dry_run: false,
        plan_id: activePlan.planID,
        action_ids: activePlan.actionIDs
      });
      setSyncLog(`${data.result.message}\nadded=${data.result.items_added} updated=${data.result.items_updated} deleted=${data.result.items_deleted}`);
      clearPlan();
      await args.refreshEntries();
    } catch (err) {
      setSyncLog(err instanceof Error ? err.message : String(err));
    } finally {
      setSyncLoading(false);
    }
  }, [args, clearPlan]);

  return {
    syncService,
    setSyncService,
    syncLoading,
    syncProgress,
    syncLog,
    plannedActions: plan.actions,
    canSyncNow,
    clearPlan,
    previewSync,
    dryRunSync,
    syncNow
  };
}
