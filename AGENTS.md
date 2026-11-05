# Instructions for AI assistants

These instructions apply to every AI tool that works in this repository. For setup, tasks, pull requests, migrations and vocabulary, read [CONTRIBUTING.md](CONTRIBUTING.md).

If the file `AGENTS.local.md` exists, read it too. It holds personal instructions of the repository owner. Git ignores that file.

## Checks

Run these checks before you finish a change:

- `go vet ./...`
- `go test -race ./...`
- `go tool golangci-lint run`
- `go tool templ generate` (the working tree must show no diff)
- `go tool templ fmt -fail .`
- `go tool sqlc diff`

Never run a server that keeps running. Never run `git stash` or other commands that change the state of the repository, unless the user asks for it.

## Commits and pull requests

- Each commit is one change that builds and passes the tests on its own. Do not bundle unrelated changes.
- Write each commit message as a [Conventional Commit](https://www.conventionalcommits.org/).
- Types that appear in the changelog: `feat`, `fix`, `perf`, `revert`, `deps`.
- Types that do not appear in the changelog: `build`, `ci`, `docs`, `refactor`, `style`, `test`. Do not use `chore`.
- Use a feature as the scope (`library`, `wishlist`, `game`, `search`, `home`, `dlc`), not a Go package. Leave the scope out when no feature fits.
- One pull request is one change that is useful on its own, with one or more commits. Split large features into several pull requests.
- Do not use `BEGIN_COMMIT_OVERRIDE`.
- Start each pull request on a new branch from an updated `main`.
- The pull request title is a Conventional Commit. The description has the sections Summary, Changes and Tests. It contains no instructions for the reader, such as "before merge" steps.

## Code style

- Follow the existing code. Split files by feature, not by type (`wishlist.go` and `wishlist_test.go`).
- Name pages after their content, not their URL (`handleLibrary`, `libraryView`). Name template data structs `<name>Page`.
- Start every templ component name with its feature (`libraryEntry`, `gameInfo`).
- Wrap errors with `%w` and return them. Log only at the boundaries (`main` and handlers). Use `log/slog`, injected and not global.
- Handlers validate cheap local input first. They answer 400 for bad input, 404 for unknown IDs and 500 for failures of the application. A successful POST redirects with 303. Every error branch returns. Write the error message in the handler and never use `err.Error()` as the response.
- Use the IGDB ID at the boundary (URLs and forms). Use the internal ID after the game is looked up.
- Write plain forms first. Add JavaScript only where plain forms are not enough.
- Run `mise run gen-templ` after every `.templ` change. Generated files (`*_templ.go`, `dbgen`) are committed.
- Never edit an existing migration. Add a new one with `mise run migrate-new <name>`.

### Doc comments

Every function, type, handler and templ component has a doc comment, exported or not.

- Write full sentences in the form `Name <verb>s …`. Write rules as short sentences.
- Write errors last: "Returns ErrA if …, or ErrB if …".
- Handlers: state what the handler renders or does. Then write "It reads <path, form or query values>, and redirects to <page>." State idempotency, for example "… counts as success".
- Predicates: "reports whether".
- Maximum line length: 120 characters.

## Tests

- Use real SQLite databases in `t.TempDir()` and `httptest.NewServer`. Do not use mocks for the database.
- Test what can fail silently. Skip plumbing and markup that changes often.
- Use one table per question. Write expected values as literals.
- Name failure messages `X() = %v, want %v`. Use `Fatal` for setup and `Errorf` for assertions.
- Send test requests through `server.ServeHTTP`, not by calling a handler directly.
- Handler tests check the status and the `Location` header.

## Writing style

Write all text in English: answers, comments, documentation and pull request texts.

- Follow ASD-STE100 Simplified Technical English. Use short sentences with one idea each. Use active voice.
- Use few adjectives. Do not use hype.
- Describe the current state and intent. Do not describe how a decision was reached. History belongs in commit messages.
