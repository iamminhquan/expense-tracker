# JSON response envelope for `/api/v1/*`

Date: 2026-10-06
Status: approved in chat, implementing

## Goal

Every JSON response from `server/internal/api` has the same top-level shape, success and error alike, produced through one type (`JSONResponse`) rather than a convention each handler follows by hand. Done means every JSON response, including the ones Gin produces on its own (unknown route, wrong method, recovered panic) and the auth middleware's 401s, carries the envelope, with tests that pin each case.

## Shape

```json
{ "success": true,  "message": "category created", "data": { "id": 7, "name": "Food" } }
{ "success": false, "message": "category not found", "data": null }
```

- All three keys are always present.
- `data` is `null` on every error, and on a success that has nothing to return.
- `message` is a short, lowercase English phrase, in the same voice as the existing error strings. On success it is specific to the endpoint (table below); on error it is exactly the string the handler already sends today, moved from the old `error` key.
- The HTTP status code keeps doing its job (400, 401, 404, 409, 500, ...). The envelope adds to the status, it does not replace it.

## Out of scope

- `GET /healthz` stays a bare `200` with no body: it is Render's probe, outside `/api/v1`.
- `GET /api/v1/transactions/export` stays a CSV download. Its *errors* are JSON and use the envelope.
- No machine-readable error codes and no per-field errors. `message` stays one flat string.
- The client does not display success messages yet (it has no toast system). They are in the response for when it does.

## Server

### `internal/api/response.go` (new)

```go
type JSONResponse[T any] struct {
    Success bool   `json:"success"`
    Message string `json:"message"`
    Data    T      `json:"data"`
}

func respondOK[T any](c *gin.Context, status int, message string, data T)
func respondError(c *gin.Context, status int, message string) // aborts; Data is nil
```

`respondError` replaces `errorResponse` (deleted, along with its home in `auth_handlers.go`). An endpoint with no payload calls `respondOK[any](c, http.StatusOK, "...", nil)`.

### No more `204`

Every endpoint that answered `204 No Content` now answers `200` with the envelope and `data: null`, since a `204` cannot carry a body (and so cannot carry the message).

### Responses Gin produces itself

| Case | Status | Message |
|---|---|---|
| `r.NoRoute` | 404 | `not found` |
| `r.NoMethod` (with `r.HandleMethodNotAllowed = true`) | 405 | `method not allowed` |
| `gin.CustomRecovery` (replaces `gin.Recovery`) | 500 | `internal server error` |

The recovered panic is still logged, never echoed into the response. `RequireAuth`'s two 401s switch from inline `gin.H` to `respondError` with their current messages. CORS preflight is answered by the CORS middleware before routing, so it stays a `204`; a test pins that the new `NoMethod` handler doesn't swallow it.

### Success messages

| Endpoint | Status | `message` | `data` |
|---|---|---|---|
| `POST /register` | 200 | `account created` | auth response |
| `POST /login` | 200 | `logged in` | auth response |
| `POST /refresh` | 200 | `session refreshed` | auth response |
| `POST /logout` | 200 (was 204) | `logged out` | `null` |
| `POST /forgot-password` | 200 (was 204) | `if that email is registered, a reset link is on its way` | `null` |
| `GET /reset-password` | 200 (was 204) | `reset link is valid` | `null` |
| `POST /reset-password` | 200 | `password reset` | auth response |
| `POST /verify-email`, verified | 200 | `email verified` | `{verified: true, conflict: false}` |
| `POST /verify-email`, conflict | 200 | `that email is already used by another account` | `{verified: false, conflict: true}` |
| `POST /verify-email`, bad link | 200 | `that verification link is invalid or has expired` | `{verified: false, conflict: false}` |
| `GET /me` | 200 | `user retrieved` | user |
| `GET /categories` | 200 | `categories retrieved` | list |
| `POST /categories` | 201 | `category created` | category |
| `PATCH /categories/:id` | 200 | `category updated` | category |
| `DELETE /categories/:id` | 200 (was 204) | `category deleted` | `null` |
| `GET /transactions` | 200 | `transactions retrieved` | list |
| `POST /transactions` | 201 | `transaction created` | transaction |
| `PATCH /transactions/:id` | 200 | `transaction updated` | transaction |
| `DELETE /transactions/:id` | 200 (was 204) | `transaction deleted` | `null` |
| `POST /transactions/import`, mapping step | 200 | `column mapping needed` | mapping response |
| `POST /transactions/import`, preview step | 200 | `import preview ready` | preview response |
| `POST /transactions/import`, confirm step | 200 | `transactions imported` | result |
| `GET /dashboard` | 200 | `dashboard retrieved` | dashboard |
| `GET /settings` | 200 | `settings retrieved` | settings |
| `PATCH /settings/profile` | 200 (was 204) | `profile updated` | `null` |
| `PATCH /settings/email` | 200 (was 204) | `check your new inbox to confirm the change` | `null` |
| `POST /settings/resend-verification` | 200 (was 204) | `verification email sent` | `null` |
| `PATCH /settings/password` | 200 (was 204) | `password updated` | `null` |
| `POST /settings/delete-account` | 200 (was 204) | `account deleted` | `null` |
| `DELETE /settings/sessions/:id` | 200 (was 204) | `session revoked` | `null` |
| `POST /settings/sessions/revoke-others` | 200 (was 204) | `other sessions revoked` | `null` |
| `PUT /settings/theme` | 200 (was 204) | `theme updated` | `null` |

`issueAuthResponse` (shared by register, login and reset-password) takes the message as a parameter.

## Client

- `client/src/lib/api/client.ts` gains `ApiResponse<T> = { success: boolean; message: string; data: T }`.
- `apiFetch` parses the body, throws `ApiError(status, body.message)` when `!res.ok || !body.success` (falling back to `statusText` when the body isn't JSON, as today), and otherwise returns `body.data`. The `204` branch goes. Hooks and pages keep receiving the same `T`.
- The two places that read the refresh response directly (`refreshAccessToken` in `client.ts`, the bootstrap in `lib/auth/AuthContext.tsx`) read `.data.accessToken`/`.data.user`.
- `lib/api/import.ts`'s own fetch does the same unwrap and reads errors from `message`.

## Testing

- Test helpers: `decodeJSON[T]` becomes `decodeData[T](t, rec) T`, which asserts `success == true` and returns `data`; new `decodeError(t, rec) string` asserts `success == false` and a literal `"data":null` in the raw body, and returns `message`. Every existing decode and every `204` assertion moves to these.
- New tests (no database needed): unknown route → 404 envelope; wrong method → 405 envelope; a panicking route added to `NewRouter`'s engine → 500 envelope with no panic text; missing/garbage bearer token → 401 envelope; CORS preflight still `204`.
- The existing raw-body "no `null` arrays" tests keep checking the arrays, now under `data`.
- Gate: `go vet ./...`, `go test ./...` (with `TEST_DATABASE_URL`), `pnpm build`, `pnpm lint`, and a real-browser smoke test (register, every page, CSV import round trip, logout) per `.claude/rules/json-api-conventions.md`.

## Docs

- `.claude/rules/json-api-conventions.md`: the "Error shape" bullet becomes the envelope rule.
- `.claude/context/server.md`: wherever it describes the response/error shape or `204`s.
- `.claude/context/README.md`: a Change Log line.
