import '../styles/LogBar.css';
import {
  ChevronDown,
  ChevronUp,
  Terminal
} from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { api } from '../api/client';

const MAX_LOG_LINES = 300;

export function LogBar({ open, onToggle }: { open: boolean; onToggle: () => void }) {
  const preRef = useRef<HTMLPreElement>(null);
  const cursorRef = useRef(0);
  const [lines, setLines] = useState<string[]>([]);
  const [active, setActive] = useState(false);
  const [height, setHeight] = useState(160);
  const dragRef = useRef<{ startY: number; startH: number } | null>(null);

  const onDragStart = (e: React.MouseEvent) => {
    e.preventDefault();
    dragRef.current = { startY: e.clientY, startH: height };
    const onMove = (ev: MouseEvent) => {
      if (!dragRef.current) return;
      const delta = dragRef.current.startY - ev.clientY; // drag up = bigger
      setHeight(Math.max(80, Math.min(600, dragRef.current.startH + delta)));
    };
    const onUp = () => {
      dragRef.current = null;
      window.removeEventListener('mousemove', onMove);
      window.removeEventListener('mouseup', onUp);
    };
    window.addEventListener('mousemove', onMove);
    window.addEventListener('mouseup', onUp);
  };

  // Continuously poll /api/logs regardless of open state so the count stays current.
  useEffect(() => {
    let cancelled = false;
    let activityTimer: ReturnType<typeof setTimeout> | null = null;

    const poll = async () => {
      while (!cancelled) {
        try {
          const data = await api.logs(cursorRef.current);
          if (data.lines.length > 0) {
            cursorRef.current = data.cursor;
            setActive(true);
            if (activityTimer) clearTimeout(activityTimer);
            activityTimer = setTimeout(() => setActive(false), 2000);
            setLines(prev => {
              const next = [...prev, ...data.lines.map(l => `[${l.level}] ${l.message}`)];
              return next.length > MAX_LOG_LINES ? next.slice(next.length - MAX_LOG_LINES) : next;
            });
          }
        } catch { /* ignore */ }
        await new Promise(r => setTimeout(r, 600));
      }
    };
    void poll();
    return () => {
      cancelled = true;
      if (activityTimer) clearTimeout(activityTimer);
    };
  }, []);

  // Auto-scroll to bottom when new lines arrive and bar is open.
  useEffect(() => {
    if (open && preRef.current) {
      preRef.current.scrollTop = preRef.current.scrollHeight;
    }
  }, [lines, open]);

  return (
    <div className={`log-bar${open ? ' open' : ''}`}>
      {open && <div className="log-bar-resize-handle" onMouseDown={onDragStart} title="Drag to resize" />}
      <button type="button" className="log-bar-toggle" onClick={onToggle}>
        <Terminal size={13} />
        <span>Server log</span>
        {active && <span className="log-bar-pill working">Live</span>}
        {!active && lines.length > 0 && <span className="log-bar-pill">{lines.length} line{lines.length !== 1 ? 's' : ''}</span>}
        {open ? <ChevronDown size={13} /> : <ChevronUp size={13} />}
      </button>
      {open && (
        <pre ref={preRef} className="log-bar-content" style={{ height: `${height}px` }}>
          {lines.length ? lines.join('\n') : 'Waiting for server log output...'}
        </pre>
      )}
    </div>
  );
}
