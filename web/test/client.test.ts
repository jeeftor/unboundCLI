import { afterEach, describe, expect, it, vi } from 'vitest';
import { api } from '../src/api/client';

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('sync plan requests', () => {
  it('requests a server-issued unsync plan for a removal', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      plan_id: 'plan-removal',
      action_ids: ['action-remove'],
      actions: [],
    }), { status: 200, headers: { 'Content-Type': 'application/json' } }));
    vi.stubGlobal('fetch', fetchMock);

    await api.planSync('unbound', 'printer.example.test', true);

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/sync/plan?service=unbound&hostname=printer.example.test&unsync=true',
      undefined,
    );
  });

  it('does not request an unsync plan for an ordinary preview', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      plan_id: 'plan-sync',
      action_ids: [],
      actions: [],
    }), { status: 200, headers: { 'Content-Type': 'application/json' } }));
    vi.stubGlobal('fetch', fetchMock);

    await api.planSync('unbound', 'printer.example.test');

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/sync/plan?service=unbound&hostname=printer.example.test',
      undefined,
    );
  });
});
