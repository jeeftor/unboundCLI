import { afterEach, describe, expect, it, vi } from 'vitest';
import { previewSync, useStore } from '../src/store';
import type { ConfigResponse } from '../src/types';

type Deferred<T> = {
  promise: Promise<T>;
  resolve: (value: T) => void;
};

function deferred<T>(): Deferred<T> {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((next) => { resolve = next; });
  return { promise, resolve };
}

function planResponse(planID: string, actionID: string): Response {
  return new Response(JSON.stringify({
    plan_id: planID,
    action_ids: [actionID],
    actions: [{ id: actionID, type: 'add', service: 'unbound', hostname: `${planID}.example.test` }],
  }), { status: 200, headers: { 'Content-Type': 'application/json' } });
}

const enabledConfig: ConfigResponse = {
  caddy: { server_ip: '10.0.0.15', server_port: 2019 },
  enabled: { caddy: true, unbound: true, adguard: false, dhcp: false, cloudflare: false },
  mutation_enabled: true,
  save_target: '/tmp/config.json',
  revision: 'test',
  summary: {} as ConfigResponse['summary'],
};

afterEach(() => {
  vi.unstubAllGlobals();
  useStore.getState().clearPlan();
  useStore.getState().setSyncLoading(false);
});

describe('sync previews', () => {
  it('keeps the newest plan when an earlier preview resolves late', async () => {
    const first = deferred<Response>();
    const second = deferred<Response>();
    vi.stubGlobal('fetch', vi.fn((url: string) => {
      if (url.includes('first.example.test')) return first.promise;
      return second.promise;
    }));
    useStore.setState({ config: enabledConfig });

    const firstPreview = previewSync('unbound', 'first.example.test');
    const secondPreview = previewSync('unbound', 'second.example.test');
    second.resolve(planResponse('second', 'action-second'));
    await expect(secondPreview).resolves.toBe(true);
    first.resolve(planResponse('first', 'action-first'));

    await expect(firstPreview).resolves.toBe(false);
    expect(useStore.getState().plan).toMatchObject({
      planID: 'second',
      hostname: 'second.example.test',
      actionIDs: ['action-second'],
    });
  });
});
