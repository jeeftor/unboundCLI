import type { Entry } from '../types';

export type HostnameDecisionAction = {
  label: string;
  description: string;
};

export type HostnameWarning = {
  kind: 'collision' | 'mismatch' | 'cf_no_dns';
  title: string;
  summary: string;
  facts: string[];
  actions: HostnameDecisionAction[];
};

export type HostnameDecision = {
  kind: 'collision' | 'mismatch' | 'service' | 'host' | 'unknown';
  severity: 'warning' | 'info' | 'ok';
  title: string;
  summary: string;
  facts: string[];
  actions: HostnameDecisionAction[];
  warnings: HostnameWarning[]; // additional issues beyond the primary
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

function cfNoDNSWarning(entry: Entry): HostnameWarning | null {
  if (!entry.cloudflare_status?.configured) return null;
  if (entry.cloudflare_status.has_dns_record) return null;
  return {
    kind: 'cf_no_dns',
    title: 'Missing Cloudflare DNS record',
    summary: `${entry.hostname} has a tunnel ingress rule but no CNAME in Cloudflare DNS — it won't resolve publicly.`,
    facts: [
      'Tunnel ingress rule exists (cloudflared knows where to route traffic)',
      `No CNAME record found: ${  entry.hostname  } → <tunnel-id>.cfargotunnel.com`,
      'External DNS returns NXDOMAIN — the tunnel is unreachable from the internet'
    ],
    actions: [
      { label: 'Fix automatically', description: 'Use "Via Caddy" or "Switch to Direct" in the Cloudflare panel to re-save the route — it will now also create the DNS record.' },
      { label: 'Fix manually', description: `In Cloudflare DNS, add a proxied CNAME: ${  entry.hostname  } → <your-tunnel-id>.cfargotunnel.com` }
    ]
  };
}

export function suppressionKey(hostname: string, kind: string): string {
  return `${hostname}:${kind}`;
}

export function isIssue(entry: Entry, caddyServerIP: string, suppressed?: Set<string>): boolean {
  const d = getHostnameDecision(entry, caddyServerIP);
  const primaryActive = (d.kind === 'collision' || d.kind === 'mismatch')
    && !suppressed?.has(suppressionKey(entry.hostname, d.kind));
  const warningsActive = d.warnings.some(w => !suppressed?.has(suppressionKey(entry.hostname, w.kind)));
  return primaryActive || warningsActive;
}

function appendCFWarning(warnings: HostnameWarning[], entry: Entry): HostnameWarning[] {
  const w = cfNoDNSWarning(entry);
  return w ? [...warnings, w] : warnings;
}

export function getHostnameDecision(entry: Entry, caddyServerIP: string): HostnameDecision {
  const shortName = firstLabel(entry.hostname);
  const dhcpIP = entry.dhcp_status?.ip || '';
  const hasMatchingDHCPHost = entry.dhcp_status?.configured === true && sameShortHostname(entry);
  const isCaddyBacked = entry.caddy_upstream !== '' || pointsToCaddy(entry, caddyServerIP);

  // Extract just the IP from the upstream (strip protocol + port)
  const upstreamIP = (entry.caddy_upstream || '').replace(/^https?:\/\//, '').split(':')[0];
  const hasUpstreamMismatch = Boolean(
    hasMatchingDHCPHost && dhcpIP && upstreamIP && dhcpIP !== upstreamIP && upstreamIP !== caddyServerIP
  );

  const mismatchWarning: HostnameWarning = {
    kind: 'mismatch',
    title: 'Wrong upstream',
    summary: `Caddy proxies to ${upstreamIP} but DHCP says ${shortName} is at ${dhcpIP}.`,
    facts: [
      `DHCP has ${shortName} at ${dhcpIP}`,
      `Caddy upstream is ${entry.caddy_upstream}`,
      'Caddy may be pointing at the wrong host — verify this is intentional'
    ],
    actions: [
      {
        label: 'Fix upstream',
        description: `Update the Caddy upstream to ${dhcpIP} if the service runs on the ${shortName} host.`
      },
      {
        label: 'Mark intentional',
        description: 'The service intentionally runs on a different host than the DHCP name suggests.'
      }
    ]
  };

  // Direct access risk — DNS → Caddy but a DHCP host shares the same short name
  if (hasMatchingDHCPHost && isCaddyBacked && pointsToCaddy(entry, caddyServerIP) && dhcpIP !== caddyServerIP) {
    return {
      kind: 'collision',
      severity: 'warning',
      title: 'Direct access risk',
      summary: `ssh ${entry.hostname} resolves to Caddy (${caddyServerIP}), not the ${shortName} host at ${dhcpIP}.`,
      facts: [
        `DNS resolves to Caddy at ${caddyServerIP}`,
        `DHCP has ${shortName} at ${dhcpIP}`,
        `Caddy proxies this name to ${entry.caddy_upstream || entry.caddy_ip || 'an upstream service'}`
      ],
      actions: [
        {
          label: 'Keep web, add SSH alias',
          description: `Keep ${entry.hostname} → Caddy for web access. Add a new DNS host record (e.g. ${shortName}-host.${entry.hostname.split('.').slice(1).join('.')}) → ${dhcpIP} for SSH/direct access.`
        },
        {
          label: 'Keep SSH, add web alias',
          description: `Point ${entry.hostname} → ${dhcpIP} for direct/SSH access. Create a new Caddy route (e.g. ${shortName}-web.${entry.hostname.split('.').slice(1).join('.')}) → Caddy for web access.`
        }
      ],
      warnings: appendCFWarning(hasUpstreamMismatch ? [mismatchWarning] : [], entry)
    };
  }

  // Standalone upstream mismatch — no DNS collision but Caddy points to wrong host
  if (hasMatchingDHCPHost && isCaddyBacked && hasUpstreamMismatch) {
    return {
      kind: 'mismatch',
      severity: 'warning',
      title: 'Wrong upstream',
      summary: `Caddy proxies ${entry.hostname} to ${upstreamIP} but DHCP says ${shortName} is at ${dhcpIP}.`,
      facts: [
        `DHCP has ${shortName} at ${dhcpIP}`,
        `Caddy upstream is ${entry.caddy_upstream}`,
        'Caddy may be pointing at the wrong host — verify this is intentional'
      ],
      actions: [
        {
          label: 'Fix upstream',
          description: `Update the Caddy upstream to ${dhcpIP} if the service runs on the ${shortName} host.`
        },
        {
          label: 'Mark intentional',
          description: 'The service intentionally runs on a different host than the DHCP name suggests.'
        }
      ],
      warnings: appendCFWarning([], entry)
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
      actions: [{ label: 'No change', description: 'Keep this name as a Caddy-backed service.' }],
      warnings: appendCFWarning([], entry)
    };
  }

  if (entry.dhcp_status?.configured) {
    return {
      kind: 'host',
      severity: 'ok',
      title: 'Direct host',
      summary: `${entry.hostname} behaves like a machine name.`,
      facts: [`DHCP has ${shortName} at ${dhcpIP}`],
      actions: [{ label: 'No change', description: 'Keep using this name for direct host access.' }],
      warnings: appendCFWarning([], entry)
    };
  }

  return {
    kind: 'unknown',
    severity: 'info',
    title: 'Needs classification',
    summary: 'This name does not have enough host or service context yet.',
    facts: ['No matching DHCP host or Caddy upstream was reported'],
    actions: [{ label: 'Review later', description: 'Classify this as a host or service when you know its intent.' }],
    warnings: appendCFWarning([], entry)
  };
}
