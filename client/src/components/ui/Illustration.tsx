export type IllustrationName = 'receipt' | 'search' | 'slices' | 'upload'

/* Small flat drawings for empty states, painted only with theme tokens so they follow dark mode. */
export function Illustration({ name, className = '' }: { name: IllustrationName; className?: string }) {
  return (
    <svg viewBox="0 0 160 112" aria-hidden="true" className={`w-auto ${className}`}>
      <rect x="30" y="18" width="100" height="84" rx="20" className="fill-surface-2" />
      {name === 'receipt' && (
        <>
          <path
            d="M52 10h56a6 6 0 0 1 6 6v76l-8-6-8 6-8-6-8 6-8-6-8 6-8-6-8 6V16a6 6 0 0 1 6-6Z"
            className="fill-surface stroke-border-strong"
            strokeWidth="1.5"
            strokeLinejoin="round"
          />
          <rect x="58" y="24" width="30" height="6" rx="3" className="fill-ink" opacity=".8" />
          <rect x="58" y="38" width="44" height="4" rx="2" className="fill-border" />
          <rect x="58" y="48" width="36" height="4" rx="2" className="fill-border" />
          <rect x="58" y="58" width="40" height="4" rx="2" className="fill-border" />
          <circle cx="118" cy="80" r="17" className="fill-accent" />
          <path d="M118 72v16M110 80h16" className="stroke-on-accent" strokeWidth="3" strokeLinecap="round" />
        </>
      )}
      {name === 'search' && (
        <>
          <rect x="44" y="28" width="62" height="60" rx="12" className="fill-surface stroke-border-strong" strokeWidth="1.5" />
          <rect x="54" y="40" width="34" height="5" rx="2.5" className="fill-border" />
          <rect x="54" y="52" width="24" height="5" rx="2.5" className="fill-border" />
          <rect x="54" y="64" width="30" height="5" rx="2.5" className="fill-border" />
          <circle cx="104" cy="60" r="19" className="fill-accent-tint stroke-ink" strokeWidth="4" />
          <path d="m118 74 14 14" className="stroke-ink" strokeWidth="6" strokeLinecap="round" />
          <path d="m97 53 14 14M111 53 97 67" className="stroke-accent" strokeWidth="3.5" strokeLinecap="round" />
        </>
      )}
      {name === 'slices' && (
        <>
          <circle cx="80" cy="58" r="32" className="fill-none stroke-surface" strokeWidth="16" />
          <circle cx="80" cy="58" r="32" className="fill-none stroke-border" strokeWidth="16" strokeDasharray="4 8" />
          <path d="M80 26a32 32 0 0 1 30.4 22" className="fill-none stroke-accent" strokeWidth="16" />
          <circle cx="80" cy="58" r="10" className="fill-surface" />
          <circle cx="122" cy="26" r="6" className="fill-chart-income" />
          <circle cx="36" cy="88" r="4" className="fill-ink" opacity=".75" />
        </>
      )}
      {name === 'upload' && (
        <>
          <path d="M56 22h34l18 18v50a6 6 0 0 1-6 6H56a6 6 0 0 1-6-6V28a6 6 0 0 1 6-6Z" className="fill-surface stroke-border-strong" strokeWidth="1.5" strokeLinejoin="round" />
          <path d="M90 22v12a6 6 0 0 0 6 6h12" className="fill-surface-2 stroke-border-strong" strokeWidth="1.5" strokeLinejoin="round" />
          <rect x="60" y="54" width="10" height="4" rx="2" className="fill-border" />
          <rect x="74" y="54" width="24" height="4" rx="2" className="fill-border" />
          <rect x="60" y="64" width="10" height="4" rx="2" className="fill-border" />
          <rect x="74" y="64" width="18" height="4" rx="2" className="fill-border" />
          <rect x="60" y="74" width="10" height="4" rx="2" className="fill-border" />
          <rect x="74" y="74" width="22" height="4" rx="2" className="fill-border" />
          <circle cx="114" cy="84" r="17" className="fill-accent" />
          <path d="M114 92V76m-7 7 7-7 7 7" className="fill-none stroke-on-accent" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" />
        </>
      )}
    </svg>
  )
}
