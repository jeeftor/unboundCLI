import { useEffect, useMemo, useState } from 'react';
import type { Entry } from '../types';

export function useEntryFilters(entries: Entry[]) {
  const [statusFilter, setStatusFilter] = useState('all');
  const [serviceFilter, setServiceFilter] = useState('all');
  const [search, setSearch] = useState('');
  const [selectedHostname, setSelectedHostname] = useState('');

  const filteredEntries = useMemo(() => entries.filter((entry) => {
    if (statusFilter === 'out_of_sync' && entry.overall_status !== 2) return false;
    if (statusFilter === 'caddy_only' && entry.overall_status !== 3) return false;
    if (statusFilter === 'stale' && entry.overall_status !== 4) return false;
    if (statusFilter === 'cloudflare' && !entry.cloudflare_status?.configured) return false;
    if (serviceFilter === 'unbound' && !entry.unbound_status?.configured) return false;
    if (serviceFilter === 'adguard' && !entry.adguard_status?.configured) return false;
    if (serviceFilter === 'dhcp' && !entry.dhcp_status?.configured) return false;
    if (serviceFilter === 'cloudflare' && !entry.cloudflare_status?.configured) return false;
    return !search.trim() || entry.hostname.toLowerCase().includes(search.trim().toLowerCase());
  }), [entries, search, serviceFilter, statusFilter]);

  useEffect(() => {
    if (selectedHostname && filteredEntries.some((entry) => entry.hostname === selectedHostname)) return;
    setSelectedHostname(filteredEntries[0]?.hostname || '');
  }, [filteredEntries, selectedHostname]);

  const selectedEntry = useMemo(
    () => entries.find((entry) => entry.hostname === selectedHostname),
    [entries, selectedHostname]
  );

  const summary = useMemo(() => ({
    entries: filteredEntries.length,
    inSync: filteredEntries.filter((entry) => entry.overall_status === 0 || entry.overall_status === 1).length,
    out: filteredEntries.filter((entry) => entry.overall_status === 2).length,
    caddyOnly: filteredEntries.filter((entry) => entry.overall_status === 3).length,
    stale: filteredEntries.filter((entry) => entry.overall_status === 4).length,
    cloudflare: filteredEntries.filter((entry) => entry.cloudflare_status?.configured).length
  }), [filteredEntries]);

  return {
    statusFilter,
    setStatusFilter,
    serviceFilter,
    setServiceFilter,
    search,
    setSearch,
    selectedHostname,
    setSelectedHostname,
    filteredEntries,
    selectedEntry,
    summary
  };
}
