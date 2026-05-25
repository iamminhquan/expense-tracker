# AGENTS.md

Behavioral guidelines to reduce common LLM coding mistakes. Merge with project-specific instructions as needed.

**Tradeoff:** These guidelines bias toward caution over speed. For trivial tasks, use judgment.

## 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:
- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them - don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

## 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

## 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it - don't delete it.

When your changes create orphans:
- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

## 4. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:
- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:
```
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
```

Strong success criteria let you loop independently. Weak criteria ("make it work") require constant clarification.

---

**These guidelines are working if:** fewer unnecessary changes in diffs, fewer rewrites due to overcomplication, and clarifying questions come before implementation rather than after mistakes.

# Repository Guidelines

## Project Structure & Module Organization

This repository has two independent application roots, both nested one level under their area folder:

- `back-end/expense-tracker/`: Go API service, module `github.com/iamminhquan/expense-tracker`. `main.go` loads configuration, opens the database connection, and starts the router.
- `front-end/expense-tracker/`: Next.js 16 client managed with Bun. App Router files live in `app/`, reusable UI components in `components/`, helpers in `lib/`, and static assets in `public/`.

Backend domain code lives under `back-end/expense-tracker/internal/`:

- `config/`: environment configuration and DSN construction.
- `database/`: GORM PostgreSQL connection and migrations.
- `models/`: persistent model structs.
- `repositories/`: database access wrappers.
- `services/`: business logic and auth-secret aware operations.
- `handlers/`: Gin HTTP handlers.
- `auth/`: JWT creation and validation helpers.
- `middleware/`: auth and CORS middleware.
- `routes/`: Gin route registration.

Keep backend packages focused by layer. Keep frontend route-specific code in `app/` and reusable primitives in `components/` or `lib/`.

## Build, Test, and Development Commands

Backend commands, run from `back-end/expense-tracker/`:

- `go run .`: start the API on `APP_PORT` or `8080`.
- `go test ./...`: run all Go tests.
- `go fmt ./...`: format Go source before committing.
- `go mod tidy`: clean dependency metadata after adding or removing packages.

Frontend commands, run from `front-end/expense-tracker/`:

- `bun install`: install dependencies from `bun.lock`.
- `bun dev`: start the Next.js dev server.
- `bun run build`: create a production build.
- `bun run lint`: run ESLint.

This project uses Next.js 16 and React 19. Before changing frontend framework APIs or file conventions, check the relevant local docs under `node_modules/next/dist/docs/`.

## API Surface

Current backend routes:

- `GET /health`
- `GET /api/v1/hello`
- `GET /api/v1/hello/:name`
- `POST /api/v1/users`
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `GET /api/v1/auth/me` with bearer-token authentication

The frontend reads `NEXT_PUBLIC_API_BASE_URL`, defaulting to `http://localhost:8080/api/v1`, and stores the login token in `localStorage` under `auth_token`.

## Coding Style & Naming Conventions

### Go

- Run `go fmt ./...` before committing. Do not hand-format Go files.
- Use short, lowercase package names with no underscores or mixed caps.
- Keep packages focused by layer: handlers should handle HTTP, services should own business logic, and repositories should own database access.
- Export identifiers only when they are used across packages. Keep package-private helpers unexported.
- Prefer clear, role-based file names such as `user_handler.go`, `user_service.go`, `user_repository.go`, or `auth_middleware.go`.
- Keep handlers thin. Validate and bind request data in handlers, then delegate business rules to services.
- Pass `context.Context` through database or request-scoped operations when applicable.
- Return errors instead of panicking in application code. Wrap errors with useful context when crossing package boundaries.
- Prefer table-driven tests for services, handlers, and repository behavior.

### Next.js + TypeScript

- Use TypeScript for frontend code. Avoid `any`; prefer explicit types, inferred local types, or `unknown` with narrowing.
- Use PascalCase for React components and component files, such as `ExpenseCard.tsx`.
- Use camelCase for functions, variables, hooks, and utility files, such as `formatCurrency.ts`.
- Use lowercase route folders under `app/`, and keep route-specific UI close to its route.
- Put reusable UI primitives in `components/` and shared helpers in `lib/`.
- Prefer Server Components by default. Add `"use client"` only when the component needs state, effects, browser APIs, or event handlers.
- Keep client components small and move non-interactive data formatting or mapping logic outside them when possible.
- Use Tailwind CSS utility classes and existing shadcn-style components before adding new CSS.
- Keep component props typed with `type` aliases or interfaces. Name props types clearly, such as `ExpenseCardProps`.
- Validate external data at boundaries before rendering or storing it.
- Keep environment variables explicit. Only expose browser-safe values with the `NEXT_PUBLIC_` prefix.

## Testing Guidelines

Add Go tests next to the package under test using `*_test.go` and standard `TestXxx` names. Prefer table-driven tests for services and handlers.

For frontend changes, add component or integration tests when a test framework is introduced. Until then, verify with `bun run lint` and `bun run build`.

## Commit & Pull Request Guidelines

Use Conventional Commits for every commit:

```text
<type>(optional-scope): <imperative summary>
```

Allowed common types:

- `feat`: new user-facing feature or capability.
- `fix`: bug fix.
- `refactor`: code restructuring without behavior changes.
- `test`: add or update tests.
- `docs`: documentation-only changes.
- `style`: formatting-only changes.
- `chore`: maintenance, dependencies, config, or tooling.
- `build`: build system or dependency changes.
- `ci`: CI/CD workflow changes.

Commit rules:

- Keep each commit subject at or under 100 characters.
- Use a concise, imperative summary, for example `feat(auth): add login handler`.
- Keep commits small and focused. One commit should represent one logical change.
- Keep commits atomic. Each commit should contain only one logical change.
- Do not group unrelated changes into a single commit, even if they touch nearby files.
- Separate refactors, formatting, bug fixes, features, and dependency changes into different commits when possible.
- A reviewer should understand the purpose of the commit from the diff and commit message alone.
- Do not mix unrelated backend, frontend, refactor, and formatting changes in the same commit.
- Use a body when the reason or tradeoff is not obvious from the subject.
- Avoid vague subjects such as `update code`, `fix bug`, or `changes`.

Pull request rules:

- Include a short description of what changed and why.
- Link the related issue or task when applicable.
- List verification steps and results, such as `go test ./...`, `bun run lint`, or `bun run build`.
- Include screenshots or short recordings for visible frontend changes.
- Mention database migrations, environment variables, seed data, or setup changes.
- Keep PRs focused. Split large unrelated changes into separate PRs.

## Security & Configuration Tips

Backend configuration is read from environment variables:

- `APP_ENV`
- `APP_PORT`
- `DB_HOST`
- `DB_PORT`
- `DB_USER`
- `DB_PASSWORD`
- `DB_NAME`
- `DB_SSLMODE`
- `AUTH_SECRET`
- `CORS_ALLOWED_ORIGIN`

Do not commit real credentials. Use `.env`, local shell environment settings, or other ignored local overrides for development.
