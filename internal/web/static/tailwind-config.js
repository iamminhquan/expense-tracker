// Tailwind Play CDN configuration. Loaded immediately after the CDN script
// itself, which does its first scan on DOMContentLoaded, so assigning the
// config here still reaches that first pass.
//
// Every colour resolves through a --c-* variable declared in app.css rather
// than a literal, which is what lets one palette swap recolour the whole UI;
// the <alpha-value> placeholder is what keeps opacity modifiers such as
// bg-accent/10 working.
tailwind.config = {
  theme: {
    extend: {
      colors: {
        app: 'rgb(var(--c-app) / <alpha-value>)',
        surface: 'rgb(var(--c-surface) / <alpha-value>)',
        'surface-alt': 'rgb(var(--c-surface-alt) / <alpha-value>)',
        track: 'rgb(var(--c-track) / <alpha-value>)',
        'border-card': 'rgb(var(--c-border-card) / <alpha-value>)',
        'border-input': 'rgb(var(--c-border-input) / <alpha-value>)',
        'border-list': 'rgb(var(--c-border-list) / <alpha-value>)',
        'border-nav': 'rgb(var(--c-border-nav) / <alpha-value>)',
        ink: 'rgb(var(--c-ink) / <alpha-value>)',
        'ink-muted': 'rgb(var(--c-ink-muted) / <alpha-value>)',
        'ink-faint': 'rgb(var(--c-ink-faint) / <alpha-value>)',
        'ink-faintest': 'rgb(var(--c-ink-faintest) / <alpha-value>)',
        placeholder: 'rgb(var(--c-placeholder) / <alpha-value>)',
        'ink-zero': 'rgb(var(--c-ink-zero) / <alpha-value>)',
        'nav-idle': 'rgb(var(--c-nav-idle) / <alpha-value>)',
        accent: 'rgb(var(--c-accent) / <alpha-value>)',
        expense: 'rgb(var(--c-expense) / <alpha-value>)',
        income: 'rgb(var(--c-income) / <alpha-value>)',
        'danger-tint': 'rgb(var(--c-danger-tint) / <alpha-value>)',
        'danger-bg': 'rgb(var(--c-danger-bg) / <alpha-value>)',
        'danger-border': 'rgb(var(--c-danger-border) / <alpha-value>)',
        'on-solid': 'rgb(var(--c-on-solid) / <alpha-value>)',
      },
      fontFamily: {
        sans: ['"Be Vietnam Pro"', 'system-ui', 'sans-serif'],
        mono: ['"JetBrains Mono"', 'monospace'],
      },
    },
  },
};
