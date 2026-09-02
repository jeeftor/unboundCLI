import {
  CheckCircle2,
  ChevronDown,
  ShieldCheck,
  X,
  XCircle
} from 'lucide-react';
import { useCallback, useEffect, useState } from 'react';
import { api } from '../api/client';
import type { CaddyEntry, CaddyValidateResult } from '../types';
import { LoadingSpinner } from './LoadingSpinner';

// ─── Template param schema ─────────────────────────────────────────────────
// Defines which extra inputs to show for parameterised built-in templates.

type TemplateParamDef = { key: string; label: string; placeholder?: string; required?: boolean };

const TEMPLATE_PARAMS: Record<string, TemplateParamDef[]> = {
  'forward-auth': [
    { key: 'authentik_url', label: 'Authentik URL', placeholder: '10.0.0.112:9000', required: true },
    { key: 'external_cidrs', label: 'External CIDRs (space-separated)', placeholder: 'private_ranges' },
  ],
};

// ─── Add / Edit modal ─────────────────────────────────────────────────────

export function EntryModal({
  entry,
  templates,
  defaultTemplate,
  onClose,
  onSaved
}: {
  entry: CaddyEntry | null;
  templates: string[];
  defaultTemplate: string;
  repoPath: string;
  onClose: () => void;
  onSaved: () => void;
}) {
  const isEdit = Boolean(entry);
  const [hostname, setHostname] = useState(entry?.hostname ?? '');
  const [upstream, setUpstream] = useState(entry?.upstream ?? '');
  const [template, setTemplate] = useState(defaultTemplate);
  const [params, setParams] = useState<Record<string, string>>({});
  const [preview, setPreview] = useState('');
  const [previewLoading, setPreviewLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [validating, setValidating] = useState(false);
  const [validateResult, setValidateResult] = useState<CaddyValidateResult | null>(null);
  const [error, setError] = useState('');

  const paramDefs = TEMPLATE_PARAMS[template] ?? [];

  // Reset params and validation whenever template changes.
  useEffect(() => {
    // eslint-disable-next-line @eslint-react/set-state-in-effect
    setParams({});
    // eslint-disable-next-line @eslint-react/set-state-in-effect
    setValidateResult(null);
  }, [template]);

  // Reset validation whenever other inputs change.
  useEffect(() => {
    // eslint-disable-next-line @eslint-react/set-state-in-effect
    setValidateResult(null);
  }, [hostname, upstream, params]);

  const updatePreview = useCallback(async (h: string, u: string, t: string, p: Record<string, string>) => {
    if (!h || !u) { setPreview(''); return; }
    setPreviewLoading(true);
    try {
      const res = await api.caddyPreview(h, u, t, p);
      setPreview(res.content);
    } catch {
      setPreview('');
    } finally {
      setPreviewLoading(false);
    }
  }, []);

  useEffect(() => {
    void updatePreview(hostname, upstream, template, params);
  }, [hostname, upstream, template, params, updatePreview]);

  // Load initial preview for edit mode.
  useEffect(() => {
    if (entry) void updatePreview(entry.hostname, entry.upstream, template, params);
  // eslint-disable-next-line react-hooks/exhaustive-deps, @eslint-react/exhaustive-deps
  }, []);

  const handleValidate = async () => {
    if (!hostname.trim() || !upstream.trim()) {
      setError('Hostname and upstream are required.');
      return;
    }
    setValidating(true);
    setError('');
    setValidateResult(null);
    try {
      const res = await api.caddyValidateDraft({ hostname, upstream, template, params });
      setValidateResult(res);
    } catch (err) {
      setValidateResult({ ok: false, output: String(err) });
    } finally {
      setValidating(false);
    }
  };

  const handleSave = async () => {
    setSaving(true);
    setError('');
    try {
      if (isEdit) {
        await api.caddyUpdateEntry(hostname, { upstream, template, params });
      } else {
        await api.caddyCreateEntry({ hostname, upstream, template, params });
      }
      onSaved();
    } catch (err) {
      setError(String(err));
    } finally {
      setSaving(false);
    }
  };

  const canSave = validateResult?.ok === true;

  return (
    <div className="modal-overlay" onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}>
      <div className="modal caddy-modal">
        <div className="modal-header">
          <h3>{isEdit ? 'Edit entry' : 'Add entry'}</h3>
          <button type="button" className="modal-close" onClick={onClose}><X size={18} /></button>
        </div>
        <div className="modal-body">
          <label>
            Hostname
            <input
              type="text"
              value={hostname}
              onChange={(e) => setHostname(e.target.value)}
              placeholder="myapp.example.com"
              disabled={isEdit}
            />
          </label>
          <label>
            Upstream
            <div className="upstream-field">
              <button
                type="button"
                className="upstream-scheme-toggle"
                onClick={() => setUpstream(u => {
                  if (u.startsWith('https://')) return u.replace('https://', 'http://');
                  if (u.startsWith('http://')) return u.replace('http://', 'https://');
                  return `http://${  u}`;
                })}
                title="Toggle http/https"
              >
                {upstream.startsWith('https://') ? 'https' : 'http'}
              </button>
              <input
                type="text"
                value={upstream}
                onChange={(e) => setUpstream(e.target.value)}
                placeholder="10.0.0.100:3000"
              />
            </div>
          </label>
          <label>
            Template
            <div className="select-wrapper">
              <select value={template} onChange={(e) => setTemplate(e.target.value)}>
                {templates.map((t) => (
                  <option key={t} value={t}>{t}</option>
                ))}
              </select>
              <ChevronDown size={14} />
            </div>
          </label>

          {paramDefs.length > 0 && (
            <div className="template-params">
              {paramDefs.map((def) => (
                <label key={def.key}>
                  {def.label}{def.required && <span className="required"> *</span>}
                  <input
                    type="text"
                    value={params[def.key] ?? ''}
                    onChange={(e) => setParams(p => ({ ...p, [def.key]: e.target.value }))}
                    placeholder={def.placeholder ?? ''}
                  />
                </label>
              ))}
            </div>
          )}

          {(preview || previewLoading) && (
            <div className="preview-block">
              <span className="preview-label">Preview</span>
              {previewLoading
                ? <div className="preview-loading"><LoadingSpinner size={14} /></div>
                : <pre>{preview}</pre>}
            </div>
          )}

          {validateResult && (
            <div className={`validate-result ${validateResult.ok ? 'ok' : 'error'}`}>
              {validateResult.ok
                ? <><CheckCircle2 size={14} /> Config valid — ready to write</>
                : <><XCircle size={14} /> {validateResult.output || 'Validation failed'}</>}
            </div>
          )}

          {error && <div className="caddy-error"><XCircle size={14} /> {error}</div>}
        </div>
        <div className="modal-footer">
          <button type="button" onClick={onClose}>Cancel</button>
          <button type="button" onClick={() => void handleValidate()} disabled={validating || saving}>
            {validating ? <><LoadingSpinner size={14} /> Validating...</> : <><ShieldCheck size={14} /> Validate</>}
          </button>
          <button type="button" className="btn-primary" onClick={() => void handleSave()} disabled={saving || !canSave}>
            {saving ? <><LoadingSpinner size={14} /> Saving...</> : 'Write to file'}
          </button>
        </div>
      </div>
    </div>
  );
}
