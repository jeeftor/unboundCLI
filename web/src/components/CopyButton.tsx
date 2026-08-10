import { useState } from 'react';
import { Check, Copy } from 'lucide-react';

export function CopyButton({ value, label = 'Copy' }: { value: string; label?: string }) {
  const [copied, setCopied] = useState(false);

  const onCopy = async (e: React.MouseEvent) => {
    e.stopPropagation();
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      // clipboard not available — silently ignore
    }
  };

  return (
    <button
      type="button"
      className="copy-btn"
      title={copied ? 'Copied!' : `Copy ${label}`}
      aria-label={copied ? 'Copied' : `Copy ${label}`}
      onClick={(e) => { void onCopy(e); }}
    >
      {copied ? <Check size={12} /> : <Copy size={12} />}
    </button>
  );
}
