import { describe, expect, it } from 'vitest';
import { diagnosticsEmptyState } from '../src/lib/diagnostics';

describe('diagnostics empty state', () => {
  it('does not report all clear when filters hide existing issues', () => {
    expect(diagnosticsEmptyState(3, 8)).toEqual({
      title: 'No issues match these filters',
      detail: '3 issues exist across 8 entries. Adjust the filters to review them.',
    });
  });

  it('reports all clear only when the complete inventory has no issues', () => {
    expect(diagnosticsEmptyState(0, 8)).toEqual({
      title: 'All clear!',
      detail: 'No issues found across 8 entries.',
    });
  });
});
