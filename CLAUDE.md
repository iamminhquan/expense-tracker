# $pend

Server-rendered no more: `$pend` is a monorepo with a Gin JSON API in
`server/` and a React SPA in `client/`, talking to each other over
`/api/*` with JWT auth. The project knowledge lives under `.claude/`; read
only what the task needs.

@.claude/context/README.md

Then, by task:

- Backend work (routes, data model, auth, deploy, how to run, test and build) -> `.claude/context/server.md`
- Frontend work (pages, components, API client, theming) -> `.claude/context/client.md`
- Coding rules for an area -> `.claude/rules/`, loaded on their own when you touch a matching file
- Committing or opening a pull request -> `.claude/skills/`
