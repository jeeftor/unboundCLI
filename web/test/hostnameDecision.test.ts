import assert from 'node:assert/strict';
import test from 'node:test';
import { getHostnameDecision } from '../src/lib/hostnameDecision.ts';
import type { Entry } from '../src/types.ts';

const baseEntry: Entry = {
  hostname: 'adsb.vookie.net',
  caddy_upstream: '192.168.1.81:1099',
  caddy_ip: '192.168.1.81',
  caddy_port: '1099',
  unbound_status: { configured: true, ip: '192.168.1.15', in_sync: true },
  adguard_status: { configured: true, ip: '192.168.1.15', in_sync: true },
  dhcp_status: {
    configured: true,
    type: 'static',
    ip: '192.168.1.82',
    mac: '00:11:22:33:44:55',
    hostname: 'adsb',
    in_sync: false
  },
  dns_resolved: '192.168.1.15',
  cloudflare_status: {
    configured: false,
    tunnel_name: '',
    tunnel_id: '',
    service: '',
    path: '',
    is_default_tunnel: false,
    http_host_header: '',
    no_tls_verify: false,
    http2_origin: false,
    has_access_policy: false
  },
  overall_status: 1,
  status_label: 'Synced',
  data_source: 'Caddy'
};

test('flags a Caddy-backed hostname that collides with a DHCP machine name', () => {
  const decision = getHostnameDecision(baseEntry, '192.168.1.15');

  assert.equal(decision.kind, 'collision');
  assert.equal(decision.severity, 'warning');
  assert.equal(decision.title, 'Direct access risk');
  assert.match(decision.summary, /ssh adsb\.vookie\.net/i);
  assert.deepEqual(decision.facts, [
    'DNS resolves to Caddy at 192.168.1.15',
    'DHCP has adsb at 192.168.1.82',
    'Caddy proxies this name to 192.168.1.81:1099'
  ]);
  assert.deepEqual(decision.actions.map(action => action.label), [
    'Split host and service',
    'Keep as service'
  ]);
});

test('treats a Caddy service without a matching DHCP hostname as informational', () => {
  const decision = getHostnameDecision({
    ...baseEntry,
    hostname: 'ai-pages.vookie.net',
    dhcp_status: { ...baseEntry.dhcp_status, configured: false, ip: '', hostname: '' }
  }, '192.168.1.15');

  assert.equal(decision.kind, 'service');
  assert.equal(decision.severity, 'info');
  assert.equal(decision.title, 'Caddy service');
});
