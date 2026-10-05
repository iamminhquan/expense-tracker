# $pend client

React + Vite + TypeScript + Tailwind v4 SPA, talking to the Gin JSON API
in `../server/` over `/api/*`. See the root `README.md` and
`.claude/context/frontend.md` for the full picture.

```bash
pnpm install
pnpm dev     # proxies /api/* to http://localhost:8080, see vite.config.ts
pnpm build   # outputs to dist/
pnpm lint    # oxlint
```
