import { Search, Zap } from 'lucide-react';
import { Select } from './Select';
import {
  serviceFilterOptions,
  serviceOrder,
  statusFilterOptions
} from '../lib/services';
import type { ServiceKey } from '../types';

export function EntriesToolbar({
  entriesCount,
  statusFilter,
  setStatusFilter,
  serviceFilter,
  setServiceFilter,
  search,
  setSearch,
  enabledServices,
  mutationEnabled,
  onSyncAll,
}: {
  entriesCount: number;
  statusFilter: string;
  setStatusFilter: (value: string) => void;
  serviceFilter: string;
  setServiceFilter: (value: string) => void;
  search: string;
  setSearch: (value: string) => void;
  enabledServices: Partial<Record<ServiceKey, boolean>>;
  mutationEnabled: boolean;
  onSyncAll: () => Promise<void>;
}) {
  const disabledServices = new Set(serviceOrder.filter((service) => service !== 'caddy' && enabledServices[service] === false));
  return (
    <section className="entries-toolbar panel">
      <div className="search-box">
        <Search size={15} />
        <input id="search" type="search" aria-label="Search hostnames" placeholder="Search hostnames..." value={search} onChange={(event) => setSearch(event.target.value)} />
      </div>
      <Select value={serviceFilter} onChange={setServiceFilter} ariaLabel="Service filter" options={serviceFilterOptions} disabledValues={disabledServices} />
      <Select value={statusFilter} onChange={setStatusFilter} ariaLabel="Status filter" options={statusFilterOptions} />
      <span className="entry-count">{entriesCount} entries</span>
      {mutationEnabled && (
        <button type="button" className="btn-primary btn-sm toolbar-sync-all" onClick={() => void onSyncAll()}>
          <Zap size={13} /> Sync All
        </button>
      )}
    </section>
  );
}
