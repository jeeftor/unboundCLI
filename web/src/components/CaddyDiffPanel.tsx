import { ChevronDown, GitBranch } from 'lucide-react';
import { useState } from 'react';

export function DiffPanel({ diff, status }: { diff: string; status: string }) {
  const [open, setOpen] = useState(false);
  if (!diff && !status) return null;

  const changedCount = status.split('\n').filter((l) => l.trim()).length;

  return (
    <section className="caddy-diff-panel panel">
      <button type="button" className="diff-toggle" onClick={() => setOpen(!open)}>
        <GitBranch size={14} />
        <span>Git status: {changedCount > 0 ? `${changedCount} file${changedCount !== 1 ? 's' : ''} changed` : 'clean'}</span>
        <ChevronDown size={14} className={open ? 'rotated' : ''} />
      </button>
      {open && diff && (
        <pre className="diff-content">{diff}</pre>
      )}
    </section>
  );
}
