# Repository Guidelines

## Project Structure & Module Organization
- Use `cmd/server` for the WebSocket room process and `cmd/client` for the CLI participant. Shared helpers (token hashing, checksum validation, message fan-out) stay inside `internal/chat` so both binaries consume one implementation.
- Keep protocol notes in `docs/sequence.md` or nearby docs and update diagrams whenever the handshake changes.
- Tests (`*_test.go`) live beside their packages, with deterministic fixtures in local `testdata/` folders; ignore generated transcripts through `.gitignore`.

## Build, Test, and Development Commands
- `go run ./cmd/server --port 28080` boots the room service; keep the port flag configurable even if 28080 is the default.
- `go run ./cmd/client --chat-id 1234564 --name alice` opens a CLI session; launch multiple clients with different names to simulate a room.
- `go test ./... -cover` executes every package and reports coverage; run it before every PR.
- `golangci-lint run` (or `make lint`) enforces gofmt, goimports, vet, and static analysis in one invocation.

## Coding Style & Naming Conventions
- Run gofmt/goimports on every change; prefer tabs, short descriptive names, and early returns. Exported identifiers need GoDoc comments.
- Use suffixes reflecting behavior (`RoomService`, `TokenHasher`); filenames stay snake_case (`room_service.go`, `token_hasher_test.go`).
- Constants such as ports, salts, and chat ID lengths are uppercase (`const DefaultPort = 28080`). Return typed errors like `ErrInvalidChecksum`.

## Testing Guidelines
- Favor table-driven tests with case labels such as `"rejects invalid checksum"` or `"broadcasts to peers"`. Cover valid tokens, checksum failures, disconnects, and salt mismatches.
- Maintain ≥85% coverage for chat packages and add regression tests for every bug. Manual verification = one server plus at least two clients to confirm broadcast order.

## Commit & Pull Request Guidelines
- Adopt Conventional Commits so history stays searchable (`feat: add md5 token helper`, `fix: handle disconnect timeout`). Branch names should mirror scope (`feature/token-checksum`).
- Each PR must describe intent, reference the issue, attach test/lint output, and include terminal captures for behavior changes. Ask for review from someone who can run both binaries.

## Security & Configuration Tips
- Never commit derived tokens or salts; keep placeholders in `config/example.env` and load real values via env vars. Validate chat IDs before hashing to avoid processing arbitrary data.
- Keep CLI flags for port and token salt so multiple agents can run parallel sessions without collisions.
