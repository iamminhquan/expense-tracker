export function FieldError({ children }: { children?: string | null }) {
  if (!children) return null
  return <p className="mt-2 text-[13px] text-expense">{children}</p>
}
