import type { InputHTMLAttributes } from "react";

type FormFieldProps = InputHTMLAttributes<HTMLInputElement> & {
  label: string;
};

export function FormField({ label, id, ...props }: FormFieldProps) {
  return (
    <label className="grid gap-2 text-sm text-muted-foreground" htmlFor={id}>
      {label}
      <input
        id={id}
        className="h-11 rounded-lg border border-border bg-background/70 px-3 text-foreground outline-none transition placeholder:text-muted-foreground/60 focus:border-accent focus:ring-2 focus:ring-accent/20"
        {...props}
      />
    </label>
  );
}
