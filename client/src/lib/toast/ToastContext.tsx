import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { Check, CircleAlert, X } from 'lucide-react'

type ToastKind = 'success' | 'error'

interface Toast {
  id: number
  kind: ToastKind
  message: string
}

interface ToastContextValue {
  success: (message: string) => void
  error: (message: string) => void
}

const ToastContext = createContext<ToastContextValue | null>(null)

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([])
  const nextId = useRef(1)

  const dismiss = useCallback((id: number) => setToasts((all) => all.filter((t) => t.id !== id)), [])
  const push = useCallback((kind: ToastKind, message: string) => {
    const id = nextId.current++
    setToasts((all) => [...all.slice(-2), { id, kind, message }])
  }, [])

  const value = useMemo<ToastContextValue>(
    () => ({ success: (message) => push('success', message), error: (message) => push('error', message) }),
    [push],
  )

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div
        role="status"
        aria-live="polite"
        className="pointer-events-none fixed inset-x-4 bottom-[calc(108px+env(safe-area-inset-bottom))] z-[60] flex flex-col items-center gap-2 md:right-6 md:bottom-6 md:left-auto md:items-end"
      >
        {toasts.map((toast) => (
          <ToastItem key={toast.id} toast={toast} onDismiss={() => dismiss(toast.id)} />
        ))}
      </div>
    </ToastContext.Provider>
  )
}

function ToastItem({ toast, onDismiss }: { toast: Toast; onDismiss: () => void }) {
  const [paused, setPaused] = useState(false)

  useEffect(() => {
    if (paused) return
    const timer = window.setTimeout(onDismiss, toast.kind === 'error' ? 8000 : 4000)
    return () => window.clearTimeout(timer)
  }, [paused, toast.kind, onDismiss])

  const success = toast.kind === 'success'
  return (
    <div
      onPointerEnter={() => setPaused(true)}
      onPointerLeave={() => setPaused(false)}
      onFocus={() => setPaused(true)}
      onBlur={() => setPaused(false)}
      className={`pointer-events-auto flex min-h-14 w-full max-w-[400px] animate-toast-in items-center gap-3 rounded-[20px] py-2 pr-1.5 pl-2.5 text-[14px] leading-5 font-semibold shadow-popover ${
        success ? 'bg-ink text-app' : 'border border-danger/40 bg-danger-tint text-ink'
      }`}
    >
      <span
        aria-hidden="true"
        className={`flex size-9 shrink-0 items-center justify-center rounded-full ${success ? 'bg-accent text-on-accent' : 'bg-danger text-on-danger'}`}
      >
        {success ? <Check className="size-[18px]" strokeWidth={3} /> : <CircleAlert className="size-[18px]" />}
      </span>
      <p className="flex-1">{toast.message}</p>
      <button
        type="button"
        onClick={onDismiss}
        aria-label="Dismiss"
        className="flex size-11 shrink-0 items-center justify-center rounded-full opacity-70 hover:opacity-100"
      >
        <X aria-hidden="true" className="size-4" />
      </button>
    </div>
  )
}

export function useToast(): ToastContextValue {
  const ctx = useContext(ToastContext)
  if (!ctx) throw new Error('useToast must be called within a ToastProvider')
  return ctx
}
