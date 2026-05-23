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

Go code must be `gofmt` formatted. Use short package names, exported identifiers only for cross-package APIs, and file names that match their role, such as `user_handler.go` or `user_repository.go`.

Frontend code uses TypeScript, React, Next.js App Router, Tailwind CSS, and shadcn-style components. Use PascalCase for React components, camelCase for functions and variables, and lowercase route folders under `app/`.

## Testing Guidelines

Add Go tests next to the package under test using `*_test.go` and standard `TestXxx` names. Prefer table-driven tests for services and handlers.

For frontend changes, add component or integration tests when a test framework is introduced. Until then, verify with `bun run lint` and `bun run build`.

## Commit & Pull Request Guidelines

Recent commits use Conventional Commits, for example `feat: add health handler and routes` and `refactor: simplify main bootstrap`. Continue using `feat:`, `fix:`, `refactor:`, `test:`, or `docs:` with concise imperative summaries.

Pull requests should include a short description, linked issue when applicable, test or build results, and screenshots for visible frontend changes. Mention any required environment variables or database setup changes.

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
