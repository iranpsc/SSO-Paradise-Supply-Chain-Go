import type { InputHTMLAttributes } from "react";
export function Field({
  label,
  error,
  name,
  "aria-describedby": describedBy,
  ...props
}: InputHTMLAttributes<HTMLInputElement> & {
  label: string;
  name: string;
  error?: string;
}) {
  return (
    <div className="space-y-2">
      <label htmlFor={name} className="block text-sm font-bold">
        {label}
      </label>
      <input
        id={name}
        name={name}
        className="field"
        aria-invalid={Boolean(error)}
        aria-describedby={
          [describedBy, error ? `${name}-error` : undefined]
            .filter(Boolean)
            .join(" ") || undefined
        }
        {...props}
      />
      {error && (
        <p
          id={`${name}-error`}
          className="field-error text-xs text-red-600 dark:text-red-400"
          aria-live="polite"
        >
          {error}
        </p>
      )}
    </div>
  );
}
