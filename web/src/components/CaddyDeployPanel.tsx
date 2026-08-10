import {
  CheckCircle2,
  Play,
  Terminal,
  X,
  XCircle
} from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { api } from '../api/client';
import type { CaddyDeployEvent, CaddyValidateResult } from '../types';
import { LoadingSpinner } from './LoadingSpinner';

export function DeployPanel({ onClose }: { onClose: () => void }) {
  const [log, setLog] = useState<string[]>([]);
  const [running, setRunning] = useState(false);
  const [result, setResult] = useState<'ok' | 'error' | null>(null);
  const [validateResult, setValidateResult] = useState<CaddyValidateResult | null>(null);
  const [validating, setValidating] = useState(false);
  const logRef = useRef<HTMLPreElement>(null);

  const handleValidate = async () => {
    setValidating(true);
    try {
      const res = await api.caddyValidate();
      setValidateResult(res);
    } catch (err) {
      setValidateResult({ ok: false, output: String(err) });
    } finally {
      setValidating(false);
    }
  };

  const handleDeploy = async () => {
    setRunning(true);
    setLog([]);
    setResult(null);

    const response = await fetch('/api/caddy/deploy', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-UnboundCLI-Token': window.UNBOUNDCLI_WEB_CONFIG?.applyToken ?? ''
      },
      body: JSON.stringify({})
    });

    if (!response.ok || !response.body) {
      setLog(['Error: deploy request failed']);
      setResult('error');
      setRunning(false);
      return;
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';

    while (true) {
      const { value, done } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });
      const parts = buffer.split('\n\n');
      buffer = parts.pop() ?? '';
      for (const part of parts) {
        const dataLine = part.split('\n').find((l) => l.startsWith('data: '));
        if (!dataLine) continue;
        try {
          const event = JSON.parse(dataLine.slice(6)) as CaddyDeployEvent;
          if ('done' in event && event.done) {
            setResult(event.status);
            setRunning(false);
          } else if ('line' in event) {
            setLog((prev) => [...prev, event.line]);
          }
        } catch {
          // skip malformed events
        }
      }
    }

    setRunning(false);
  };

  useEffect(() => {
    if (logRef.current) {
      logRef.current.scrollTop = logRef.current.scrollHeight;
    }
  }, [log]);

  return (
    <div className="modal-overlay" onClick={(e) => { if (e.target === e.currentTarget && !running) onClose(); }}>
      <div className="modal caddy-modal caddy-deploy-modal">
        <div className="modal-header">
          <h3><Terminal size={16} /> Deploy</h3>
          <button type="button" className="modal-close" onClick={onClose} disabled={running}><X size={18} /></button>
        </div>
        <div className="modal-body">
          <div className="deploy-actions">
            <button type="button" onClick={() => void handleValidate()} disabled={validating || running}>
              {validating ? <LoadingSpinner size={14} /> : <CheckCircle2 size={14} />}
              Validate config
            </button>
            <button type="button" className="btn-deploy" onClick={() => void handleDeploy()} disabled={running}>
              {running ? <LoadingSpinner size={14} /> : <Play size={14} />}
              {running ? 'Deploying...' : 'Deploy'}
            </button>
          </div>

          {validateResult && (
            <div className={`validate-result ${validateResult.ok ? 'ok' : 'error'}`}>
              {validateResult.ok ? <CheckCircle2 size={14} /> : <XCircle size={14} />}
              {validateResult.ok ? ' Config valid' : ' Validation failed'}
              {validateResult.output && <pre>{validateResult.output}</pre>}
            </div>
          )}

          {log.length > 0 && (
            <pre className="deploy-log" ref={logRef}>
              {log.join('\n')}
            </pre>
          )}

          {result && (
            <div className={`deploy-result ${result}`}>
              {result === 'ok'
                ? <><CheckCircle2 size={14} /> Deployment successful</>
                : <><XCircle size={14} /> Deployment failed</>}
            </div>
          )}
        </div>
        <div className="modal-footer">
          <button type="button" onClick={onClose} disabled={running}>Close</button>
        </div>
      </div>
    </div>
  );
}
