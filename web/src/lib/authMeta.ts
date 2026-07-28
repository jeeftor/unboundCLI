import {
  AlertTriangle,
  ArrowLeftRight,
  Cloud,
  Fingerprint,
  HelpCircle,
  KeyRound,
  LockKeyhole,
  Monitor,
  Route,
  ShieldCheck,
  ShieldQuestion,
  ShieldX,
  Ticket,
  Unlock,
  UnlockKeyhole,
} from 'lucide-react';
import type { ComponentType } from 'react';

// ── Shared auth metadata (used by AuthFlowsTab + VisualizeModal) ──

export type AuthMeta = {
  label: string;
  desc: string;
  icon: ComponentType<{ size?: number }>;
  tone: string;
};

export const WAN_AUTH_INFO: Record<string, AuthMeta> = {
  none: {
    label: 'None',
    desc: 'No WAN authentication. If WAN-exposed, this is a security risk — the host is reachable from the internet without any auth barrier.',
    icon: Unlock,
    tone: 'danger',
  },
  cf_access: {
    label: 'CF Access',
    desc: 'Cloudflare Access sits at the edge and requires IdP login before traffic reaches the tunnel. This is the standard edge-auth pattern.',
    icon: Cloud,
    tone: 'orange',
  },
  forward_auth: {
    label: 'Forward Auth',
    desc: "Caddy's forward_auth directive delegates authentication to Authentik. CF Access (if present) must have a bypass policy to avoid double-login.",
    icon: ArrowLeftRight,
    tone: 'blue',
  },
  app_native: {
    label: 'App-Native',
    desc: 'The application handles its own authentication (e.g., Jellyfin, Audiobookshelf have built-in login). No external auth layer is enforced.',
    icon: LockKeyhole,
    tone: 'green',
  },
};

export const LAN_AUTH_INFO: Record<string, AuthMeta> = {
  none: {
    label: 'None',
    desc: 'No LAN authentication. The app is directly accessible on the LAN. This is normal for apps with their own login (app-native).',
    icon: Unlock,
    tone: 'neutral',
  },
  forward_auth: {
    label: 'Forward Auth',
    desc: "Caddy's forward_auth delegates to Authentik even on LAN requests. Users must authenticate via Authentik before reaching the app.",
    icon: Route,
    tone: 'blue',
  },
  app_native: {
    label: 'App-Native',
    desc: 'The application handles its own authentication on LAN traffic.',
    icon: Monitor,
    tone: 'green',
  },
};

export const API_AUTH_INFO: Record<string, AuthMeta> = {
  none: {
    label: 'None',
    desc: 'No API-specific authentication. API calls use the same auth as browser traffic (or none if WAN/LAN is none).',
    icon: UnlockKeyhole,
    tone: 'neutral',
  },
  cf_service_token: {
    label: 'CF Service Token',
    desc: 'Cloudflare Access service token. Machine-to-machine calls send CF-Access-Client-Id and CF-Access-Client-Secret headers. Browsers are unaffected.',
    icon: Ticket,
    tone: 'orange',
  },
  authentik_bearer: {
    label: 'Authentik Bearer',
    desc: 'Authentik bearer token. API calls send Authorization: Bearer <token>. The Authentik proxy provider validates the token.',
    icon: Fingerprint,
    tone: 'blue',
  },
  app_native_key: {
    label: 'App-Native Key',
    desc: 'The application has its own API key mechanism (e.g., Jellyfin API keys). No external auth layer for API access.',
    icon: KeyRound,
    tone: 'green',
  },
};

export const STATUS_INFO: Record<string, AuthMeta> = {
  ok: {
    label: 'OK',
    desc: 'Auth is properly configured for this host.',
    icon: ShieldCheck,
    tone: 'green',
  },
  warning: {
    label: 'Warning',
    desc: 'Auth works but has a non-ideal configuration (e.g., split WAN/LAN modes, forward_auth without CF bypass).',
    icon: AlertTriangle,
    tone: 'yellow',
  },
  error: {
    label: 'Error',
    desc: 'Auth is missing or broken (e.g., WAN-exposed host with no auth, double-login risk from CF Access + forward_auth without bypass).',
    icon: ShieldX,
    tone: 'red',
  },
  unknown: {
    label: 'Unknown',
    desc: "Auth state couldn't be determined (e.g., Authentik/CF Access API unavailable).",
    icon: ShieldQuestion,
    tone: 'gray',
  },
};

// ── Auth pattern detection (mirrors backend classifyAuth logic) ──

export type AuthPattern = {
  name: string;
  verdict: 'ok' | 'warning' | 'error';
  summary: string;
  detail: string;
};

export function detectAuthPattern(auth: {
  wan_exposed: boolean;
  wan_auth: string;
  lan_auth: string;
  has_forward_auth: boolean;
  cf_access_app_id?: string;
  cf_access_decisions?: string[];
  notes?: string[];
}): AuthPattern | null {
  if (!auth.wan_exposed) {
    // LAN-only host
    if (auth.lan_auth === 'forward_auth') {
      return {
        name: 'Forward Auth (LAN-only)',
        verdict: 'ok',
        summary: 'LAN-only host protected by Authentik forward auth',
        detail: 'This host is not exposed to the internet. LAN traffic is authenticated via Caddy forward_auth → Authentik.',
      };
    }
    return {
      name: 'LAN-only',
      verdict: 'ok',
      summary: 'Not exposed to the internet — LAN access only',
      detail: 'No Cloudflare tunnel configured. The host is only reachable from the LAN. No external auth layer is needed unless the app itself requires login.',
    };
  }

  const hasCF = Boolean(auth.cf_access_app_id);
  const hasBypass = auth.cf_access_decisions?.includes('bypass') ?? false;
  const hasFA = auth.has_forward_auth;

  // Pattern F: CF Access + forward_auth without bypass = double login
  if (hasCF && hasFA && !hasBypass) {
    return {
      name: 'Double-Login Risk',
      verdict: 'error',
      summary: 'CF Access + forward_auth without bypass — users will be prompted to log in twice',
      detail: 'Cloudflare Access requires IdP login at the edge, then Caddy forward_auth requires Authentik login again. Fix: add a bypass policy in CF Access for this hostname so CF Access lets traffic through after the first auth, and forward_auth handles the actual authentication.',
    };
  }

  // Pattern D: CF Access bypass → forward_auth
  if (hasCF && hasFA && hasBypass) {
    return {
      name: 'CF Bypass → Forward Auth',
      verdict: 'ok',
      summary: 'CF Access bypasses to Authentik forward_auth — single login',
      detail: 'Cloudflare Access has a bypass policy for this host, so it lets traffic through without challenging. Caddy forward_auth then delegates to Authentik for the actual authentication. This is the recommended pattern for forward_auth + CF Access.',
    };
  }

  // Pattern A/B: CF Access only (but check for bypass-only)
  if (hasCF && !hasFA) {
    if (hasBypass) {
      // CF Access has a bypass policy and no forward_auth — effectively no auth
      return {
        name: 'CF Access Bypass (No Auth)',
        verdict: 'warning',
        summary: 'CF Access bypass policy lets traffic through without challenge, and no forward_auth is configured',
        detail: 'Cloudflare Access has a bypass policy for this host, so it lets all traffic through without requiring login. No forward_auth is configured either. The host is effectively unauthenticated on the WAN unless the app has its own built-in login. Fix: either remove the bypass policy from CF Access, or add forward_auth (Authentik) so Caddy enforces authentication after CF Access lets traffic through.',
      };
    }
    return {
      name: 'CF Access Only',
      verdict: 'ok',
      summary: 'Cloudflare Access handles authentication at the edge',
      detail: 'Cloudflare Access requires IdP login before traffic reaches the tunnel. No forward_auth is configured, so the app receives authenticated traffic directly.',
    };
  }

  // Pattern C: forward_auth without CF Access
  if (!hasCF && hasFA) {
    return {
      name: 'Forward Auth Only',
      verdict: 'ok',
      summary: 'Authentik forward_auth handles authentication',
      detail: 'Caddy forward_auth delegates to Authentik. No CF Access policy is configured, so the tunnel passes traffic directly to Caddy, which then enforces Authentik login.',
    };
  }

  // No auth layer at all
  if (!hasCF && !hasFA) {
    return {
      name: 'No Auth Layer',
      verdict: 'warning',
      summary: 'WAN-exposed with no CF Access or forward_auth — relying on app-native auth',
      detail: 'This host is reachable from the internet but has no external auth layer (no CF Access, no forward_auth). The application must handle its own authentication. If the app does not have built-in login, this is a security risk.',
    };
  }

  return null;
}

// ── Request flow steps (what happens when a user hits the service) ──

export type FlowStep = {
  step: number;
  actor: string;
  action: string;
  result: string;
  warn?: boolean;
};

export function buildWanRequestFlow(auth: {
  wan_exposed: boolean;
  wan_auth: string;
  has_forward_auth: boolean;
  cf_access_app_id?: string;
  cf_access_decisions?: string[];
  authentik_provider_mode?: string;
  authentik_app_slug?: string;
  hostname: string;
  upstream: string;
}): FlowStep[] {
  if (!auth.wan_exposed) {
    return [{
      step: 1,
      actor: '—',
      action: 'Not WAN-exposed',
      result: 'No Cloudflare tunnel. Requests from the internet cannot reach this host.',
    }];
  }

  const hasCF = Boolean(auth.cf_access_app_id);
  const hasBypass = auth.cf_access_decisions?.includes('bypass') ?? false;
  const hasFA = auth.has_forward_auth;
  const steps: FlowStep[] = [];
  let n = 1;

  steps.push({ step: n++, actor: 'User', action: `Requests https://${auth.hostname}`, result: 'Browser sends HTTPS request' });

  if (hasCF) {
    steps.push({ step: n++, actor: 'Cloudflare', action: 'Intercepts request at edge', result: 'Checks for CF_Authorization JWT cookie' });
    if (hasBypass) {
      steps.push({ step: n++, actor: 'CF Access', action: 'Bypass policy matches', result: 'No challenge — traffic passes through' });
      if (!hasFA) {
        steps.push({ step: n++, actor: 'CF Access', action: 'No forward_auth after bypass', result: 'Request continues unauthenticated', warn: true });
      }
    } else {
      steps.push({ step: n++, actor: 'CF Access', action: 'No JWT cookie found', result: '302 redirect to IdP login (Google/GitHub/etc.)' });
      steps.push({ step: n++, actor: 'IdP', action: 'User authenticates', result: 'IdP redirects back to CF with auth code' });
      steps.push({ step: n++, actor: 'CF Access', action: 'Exchanges code, sets JWT cookie', result: 'CF_Authorization cookie set, request continues' });
    }
    steps.push({ step: n++, actor: 'Cloudflare Tunnel', action: 'Forwards to origin', result: `Encrypted tunnel to Caddy (${auth.hostname})` });
  }

  steps.push({ step: n++, actor: 'Caddy', action: 'Receives request', result: 'Processes route handler' });

  if (hasFA) {
    const isDoubleLogin = hasCF && !hasBypass;
    const mode = auth.authentik_provider_mode || 'forward_single';

    steps.push({ step: n++, actor: 'Caddy', action: `forward_auth subrequest to Authentik`, result: `Checks session cookie with Authentik outpost (${mode})` });

    if (isDoubleLogin) {
      steps.push({ step: n++, actor: 'Authentik', action: 'No Authentik session cookie', result: '302 redirect to Authentik login', warn: true });
      steps.push({ step: n++, actor: 'Authentik', action: 'User logs in AGAIN', result: 'Session cookie set, redirect back to Caddy', warn: true });
    } else {
      steps.push({ step: n++, actor: 'Authentik', action: 'Validates session cookie', result: '200 OK (authenticated) or 302 to login' });
      steps.push({ step: n++, actor: 'Authentik', action: 'If no session: redirect to login', result: `User authenticates at Authentik (${auth.authentik_app_slug || 'app'}), cookie set` });
    }

    steps.push({ step: n++, actor: 'Caddy', action: 'forward_auth returned 200', result: 'Forwards request to upstream' });
  } else {
    steps.push({ step: n++, actor: 'Caddy', action: 'No forward_auth configured', result: 'Forwards directly to upstream' });
  }

  steps.push({ step: n++, actor: 'Service', action: `Receives request at ${auth.upstream}`, result: 'Serves content' });

  return steps;
}

export function buildLanRequestFlow(auth: {
  has_forward_auth: boolean;
  authentik_provider_mode?: string;
  authentik_app_slug?: string;
  hostname: string;
  upstream: string;
  dns_resolved: string;
  unbound_ip?: string;
}): FlowStep[] {
  const steps: FlowStep[] = [];
  let n = 1;

  steps.push({ step: n++, actor: 'LAN Client', action: `Requests https://${auth.hostname}`, result: 'Browser sends request' });
  steps.push({ step: n++, actor: 'Unbound DNS', action: `Resolves ${auth.hostname}`, result: auth.dns_resolved ? `→ ${auth.dns_resolved}` : 'NXDOMAIN (not in DNS)' });
  steps.push({ step: n++, actor: 'Caddy', action: 'Receives request', result: 'Processes route handler' });

  if (auth.has_forward_auth) {
    const mode = auth.authentik_provider_mode || 'forward_single';
    steps.push({ step: n++, actor: 'Caddy', action: `forward_auth subrequest to Authentik`, result: `Checks session cookie (${mode})` });
    steps.push({ step: n++, actor: 'Authentik', action: 'Validates session', result: '200 OK or 302 to Authentik login' });
    steps.push({ step: n++, actor: 'Caddy', action: 'forward_auth returned 200', result: 'Forwards to upstream' });
  } else {
    steps.push({ step: n++, actor: 'Caddy', action: 'No forward_auth', result: 'Forwards directly to upstream' });
  }

  steps.push({ step: n++, actor: 'Service', action: `Receives request at ${auth.upstream}`, result: 'Serves content' });

  return steps;
}
