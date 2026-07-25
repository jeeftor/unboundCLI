import { ChevronDown } from 'lucide-react';

export function Select({ value, onChange, options, ariaLabel, disabledValues = new Set<string>(), idForAll, className }: { value: string; onChange: (value: string) => void; options: [string, string][]; ariaLabel: string; disabledValues?: Set<string>; idForAll?: string; className?: string }) {
  return (
    <span className={`select-wrap ${className || ''}`}>
      <select value={value} aria-label={ariaLabel} onClick={(event) => event.stopPropagation()} onChange={(event) => onChange(event.target.value)}>
        {options.map(([optionValue, label]) => (
          <option key={optionValue} id={idForAll && optionValue === 'all' ? idForAll : undefined} value={optionValue} data-service={optionValue !== 'all' ? optionValue : undefined} disabled={disabledValues.has(optionValue)} className={disabledValues.has(optionValue) ? 'disabled-service' : undefined}>{label}</option>
        ))}
      </select>
      <ChevronDown size={14} />
    </span>
  );
}
