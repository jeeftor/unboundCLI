import type { Entry } from '../types';

export type HostnameDecisionAction = {
  label: string;
  description: string;
};

export type HostnameDecision = {
  kind: 'collision' | 'service' | 'host' | 'unknown';
  severity: 'warning' | 'info' | 'ok';
  title: string;
  summary: string;
  facts: string[];
  actions: HostnameDecisionAction[];
};

function firstLabel(hostname: string): string {
  return hostname.split('.')[0] || hostname;
}

function serviceIPs(entry: Entry): string[] {
  return [entry.dns_resolved, entry.unbound_status?.ip, entry.adguard_status?.ip]
    .filter((ip): ip is string => Boolean(ip));
}

function pointsToCaddy(entry: Entry, caddyServerIP: string): boolean {
  return Boolean(caddyServerIP && serviceIPs(entry).includes(caddyServerIP));
}

function sameShortHostname(entry: Entry): boolean {
  const dhcpHostname = entry.dhcp_status?.hostname;
  return Boolean(dhcpHostname && dhcpHostname === firstLabel(entry.hostname));
}

export function getHostnameDecision(entry: Entry, caddyServerIP: string): HostnameDecision {
  const shortName = firstLabel(entry.hostname);
  const dhcpIP = entry.dhcp_status?.ip || '';
  const hasMatchingDHCPHost = entry.dhcp_status?.configured === true && sameShortHostname(entry);
  const isCaddyBacked = entry.caddy_upstream !== '' || pointsToCaddy(entry, caddyServerIP);

  if (hasMatchingDHCPHost && isCaddyBacked && pointsToCaddy(entry, caddyServerIP) && dhcpIP !== caddyServerIP) {
    return {
      kind: 'collision',
      severity: 'warning',
      title: 'Direct access risk',
      summary: `Direct access like ssh ${entry.hostname} will resolve to Caddy, not the ${shortName} host.`,
      facts: [
        `DNS resolves to Caddy at ${caddyServerIP}`,
        `DHCP has ${shortName} at ${dhcpIP}`,
        `Caddy proxies this name to ${entry.caddy_upstream || entry.caddy_ip || 'an upstream service'}`
      ],
      actions: [
        {
          label: 'Split host and service',
          description: `${entry.hostname} points to ${dhcpIP}; create ${shortName}-web.${entry.hostname.split('.').slice(1).join('.')} for Caddy.`
        },
        {
          label: 'Keep as service',
          description: `Leave ${entry.hostname} on Caddy and create a host alias for SSH.`
        }
      ]
    };
  }

  if (isCaddyBacked) {
    return {
      kind: 'service',
      severity: 'info',
      title: 'Caddy service',
      summary: `${entry.hostname} behaves like a web service name. DNS should point at Caddy.`,
      facts: [
        caddyServerIP ? `Caddy entrypoint is ${caddyServerIP}` : 'Caddy entrypoint is not configured',
        entry.caddy_upstream ? `Caddy proxies to ${entry.caddy_upstream}` : 'No Caddy upstream was reported',
        dhcpIP ? `DHCP context is ${dhcpIP}` : 'No matching DHCP host was found'
      ],
      actions: [
        {
          label: 'No change',
          description: 'Keep this name as a Caddy-backed service.'
        }
      ]
    };
  }

  if (entry.dhcp_status?.configured) {
    return {
      kind: 'host',
      severity: 'ok',
      title: 'Direct host',
      summary: `${entry.hostname} behaves like a machine name.`,
      facts: [`DHCP has ${shortName} at ${dhcpIP}`],
      actions: [{ label: 'No change', description: 'Keep using this name for direct host access.' }]
    };
  }

  return {
    kind: 'unknown',
    severity: 'info',
    title: 'Needs classification',
    summary: 'This name does not have enough host or service context yet.',
    facts: ['No matching DHCP host or Caddy upstream was reported'],
    actions: [{ label: 'Review later', description: 'Classify this as a host or service when you know its intent.' }]
  };
}
