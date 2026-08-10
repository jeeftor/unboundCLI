import { useEffect, useRef, type RefObject } from 'react';

const FOCUSABLE_SELECTOR = 'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';

export function useFocusTrap<T extends HTMLElement>(
  active: boolean,
  onEscape?: () => void
): RefObject<T | null> {
  const containerRef = useRef<T>(null);

  useEffect(() => {
    if (!active) return;

    // Store the currently focused element to restore later
    const previouslyFocused = document.activeElement as HTMLElement | null;

    // Focus the first focusable element in the container
    const container = containerRef.current;
    if (container) {
      const focusable = container.querySelectorAll(FOCUSABLE_SELECTOR);
      const first = focusable[0] as HTMLElement | undefined;
      if (first) {
        // Small delay to ensure the modal is rendered
        requestAnimationFrame(() => first.focus());
      } else {
        // Focus the container itself as fallback
        container.tabIndex = -1;
        requestAnimationFrame(() => container.focus());
      }
    }

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && onEscape) {
        e.preventDefault();
        onEscape();
        return;
      }

      if (e.key !== 'Tab') return;

      const container = containerRef.current;
      if (!container) return;

      const focusable = Array.from(container.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR));
      if (focusable.length === 0) return;

      const first = focusable[0];
      const last = focusable[focusable.length - 1];

      if (e.shiftKey) {
        if (document.activeElement === first || !container.contains(document.activeElement)) {
          e.preventDefault();
          last.focus();
        }
      } else {
        if (document.activeElement === last || !container.contains(document.activeElement)) {
          e.preventDefault();
          first.focus();
        }
      }
    };

    document.addEventListener('keydown', handleKeyDown);

    return () => {
      document.removeEventListener('keydown', handleKeyDown);
      // Restore focus to the previously focused element
      if (previouslyFocused && typeof previouslyFocused.focus === 'function') {
        previouslyFocused.focus();
      }
    };
  }, [active, onEscape]);

  return containerRef;
}
