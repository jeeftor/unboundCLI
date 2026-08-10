import { Loader2 } from 'lucide-react';

/**
 * Reusable loading spinner — replaces the 28+ ad-hoc
 * `<Loader2 size={N} className="spin" />` patterns across the codebase.
 */
export function LoadingSpinner({ size = 14 }: { size?: number }) {
  return <Loader2 size={size} className="spin" />;
}
