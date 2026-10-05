# $pend client

React + Vite + TypeScript + Tailwind v4 SPA, replacing the server-rendered
`html/template` + htmx frontend in `../server/`. See the root `README.md`
and `CLAUDE.md`'s "Migration in Progress" section for context — this
package is currently a placeholder scaffold (`src/pages/`,
`src/components/`, `src/api/` are empty) with no real pages yet.

```bash
pnpm install
pnpm dev     # proxies /api/* to http://localhost:8080, see vite.config.ts
pnpm build   # outputs to dist/
pnpm lint    # oxlint
```
