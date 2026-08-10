import {
  Cloud,
  Globe,
  Network,
  Route,
  ShieldX,
  Trash2,
} from 'lucide-react';
import { memo, useMemo } from 'react';
import { LoadingSpinner } from './LoadingSpinner';
import type { PruneAction, PruneResponse } from '../types';

// ─── Prune action row ───────────────────────────────────────────────────────

const PruneActionRow = memo(function PruneActionRow({ action }: { action: PruneAction }) {
  const serviceIcon = action.service === 'unbound' ? <Globe size={12} />
    : action.service === 'adguard' ? <ShieldX size={12} />
    : action.service === 'cloudflare_tunnel' ? <Route size={12} />
    : <Cloud size={12} />;
  return (
    <div className={`prune-action-row ${action.error ? 'error' : action.success ? 'success' : ''}`}>
      <div className="prune-action-service">{serviceIcon} {action.service}</div>
      <div className="prune-action-detail">
        <strong>{action.hostname}</strong> — {action.detail}
        {action.error && <span className="prune-action-error"> ✗ {action.error}</span>}
        {action.success && <span className="prune-action-success"> ✓</span>}
      </div>
    </div>
  );
});

// ─── Prune modal ────────────────────────────────────────────────────────────

export function PruneModal({
  prunePreview,
  pruneResult,
  pruneError,
  pruneSelected,
  pruneLoading,
  onClose,
  onSetSelected,
  onExecute,
  onCancel,
}: {
  prunePreview: PruneResponse | null;
  pruneResult: PruneResponse | null;
  pruneError: string | null;
  pruneSelected: Set<string>;
  pruneLoading: boolean;
  onClose: () => void;
  onSetSelected: (updater: (prev: Set<string>) => Set<string>) => void;
  onExecute: (hostnames: string[]) => void;
  onCancel: () => void;
}) {
  const pruneHostnames = useMemo(() => {
    if (!prunePreview) return [];
    return Array.from(new Set(prunePreview.actions.map(a => a.hostname))).sort();
  }, [prunePreview]);

  return (
    <div className="prune-modal-overlay" onClick={onClose}>
      <div className="prune-modal" onClick={e => e.stopPropagation()}>
        <div className="prune-modal-header">
          <Trash2 size={18} />
          <h3>{pruneResult ? 'Prune Results' : 'Prune Preview (Dry Run)'}</h3>
          <button type="button" className="btn-sm" onClick={onClose}>Close</button>
        </div>
        {pruneError && (
          <div className="auth-error">
            <ShieldX size={16} />
            <div><strong>Prune failed:</strong> {pruneError}</div>
          </div>
        )}
        {prunePreview && (
          <>
            <p className="prune-modal-desc">
              Found <strong>{prunePreview.total}</strong> action{prunePreview.total !== 1 ? 's' : ''} across <strong>{pruneHostnames.length}</strong> hostname{pruneHostnames.length !== 1 ? 's' : ''}.
              Select which hostnames to prune, then confirm.
            </p>
            <div className="prune-actions-list">
              {pruneHostnames.map(hn => {
                const hostActions = prunePreview.actions.filter(a => a.hostname === hn);
                const checked = pruneSelected.has(hn);
                return (
                  <div key={hn} className={`prune-host-group ${checked ? 'selected' : 'deselected'}`}>
                    <label className="prune-host-checkbox">
                      <input
                        type="checkbox"
                        checked={checked}
                        onChange={(e) => {
                          onSetSelected(prev => {
                            const next = new Set(prev);
                            if (e.target.checked) next.add(hn);
                            else next.delete(hn);
                            return next;
                          });
                        }}
                      />
                      <Network size={13} />
                      <strong>{hn}</strong>
                      <span className="prune-host-action-count">{hostActions.length} action{hostActions.length !== 1 ? 's' : ''}</span>
                    </label>
                    {checked && (
                      <div className="prune-host-actions">
                        {hostActions.map((a) => <PruneActionRow key={`${hn}-${a.service}`} action={a} />)}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
            {prunePreview.total > 0 && (
              <div className="prune-modal-footer">
                <div className="prune-select-controls">
                  <button type="button" className="btn-sm" onClick={() => onSetSelected(() => new Set(pruneHostnames))}>Select All</button>
                  <button type="button" className="btn-sm" onClick={() => onSetSelected(() => new Set())}>Deselect All</button>
                  <span className="prune-selected-count">{pruneSelected.size} of {pruneHostnames.length} selected</span>
                </div>
                <div className="prune-confirm-controls">
                  <button type="button" className="btn-sm" onClick={onCancel}>Cancel</button>
                  <button
                    type="button"
                    className="btn-sm danger"
                    onClick={() => onExecute(Array.from(pruneSelected))}
                    disabled={pruneLoading || pruneSelected.size === 0}
                  >
                    {pruneLoading ? <LoadingSpinner size={14} /> : <Trash2 size={14} />} Delete {pruneSelected.size > 0 ? `(${pruneSelected.size})` : ''}
                  </button>
                </div>
              </div>
            )}
          </>
        )}
        {pruneResult && (
          <>
            <p className="prune-modal-desc">
              Completed <strong>{pruneResult.total}</strong> action{pruneResult.total !== 1 ? 's' : ''}.
            </p>
            <div className="prune-actions-list">
              {pruneResult.actions.map((a) => <PruneActionRow key={`${a.hostname}-${a.service}`} action={a} />)}
            </div>
          </>
        )}
      </div>
    </div>
  );
}
