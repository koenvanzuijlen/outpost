# Contributing to Outpost

This guide is for developers. To run Outpost, see the [README](README.md).

## Setup

Outpost needs [mise](https://mise.jdx.dev/). mise installs Go and Node in the versions that the project uses.

```sh
mise install
cp .env.example .env    # add your Twitch credentials
mise setup              # Prettier in node_modules/, dev tools in .tools/ for the VS Code extensions
mise start
```

The app runs on `http://localhost:8080` and uses `outpost.db` in the repository folder. `mise run start-fresh` deletes that database first.

To build the Docker image locally:

```sh
docker build -t outpost .
```

## Tasks

| Task                   | What it does                                        |
| ---------------------- | --------------------------------------------------- |
| `mise run test`        | Runs all tests with the race detector.              |
| `mise run lint`        | Runs golangci-lint.                                 |
| `mise run vulncheck`   | Checks for known vulnerabilities with govulncheck.  |
| `mise run gen-sql`     | Generates the query code in `dbgen` with sqlc.      |
| `mise run gen-templ`   | Generates the Go code of the templ components.      |
| `mise run gen-icons`   | Copies the Lucide icons in `icons.txt`.             |
| `mise run templ-fmt`   | Formats the templ files. Requires Prettier.         |
| `mise run fmt`         | Formats CSS, Markdown, YAML and JSON with Prettier. |
| `mise run fmt-check`   | Checks the formatting of these files with Prettier. |
| `mise run migrate-new` | Creates a new migration.                            |

CI runs the tests, lint, `templ generate`, `templ fmt`, `sqlc diff` and Prettier on every pull request. Run them before you push.

## Pull requests

- Each pull request is one change that is useful on its own. A large feature is split into several pull requests.
- Pull requests are squash merged. The pull request title becomes the commit message on `main` and the changelog line, so it must be a [Conventional Commit](https://www.conventionalcommits.org/): `feat(library): …`, `fix: …`.
- Use a feature as the scope (`library`, `wishlist`, `game`, `search`, `home`), not a Go package. Leave the scope out when no feature fits.
- Use a specific type instead of `chore`: `build`, `ci`, `docs`, `refactor`, `style` or `test`. Only `feat`, `fix`, `perf`, `revert` and `deps` appear in the changelog.

## Migrations

Migrations are in `internal/database/migrations`. A migration on `main` never changes: CI fails a pull request that edits, deletes or renames one. Add a new migration with `mise run migrate-new <name>`. Outpost backs up the database before it runs new migrations.

## Icons

The icons come from the npm package [`lucide-static`](https://lucide.dev/guide/packages/lucide-static). `internal/server/icons/icons.txt` lists the icons that the app uses, one name per line. `mise run gen-icons` copies these icons and the Lucide license from `node_modules` into `internal/server/icons/`. A test fails when the SVG files and `icons.txt` do not match.

- To add an icon, add its name to `icons.txt` and run `mise run gen-icons`. Use the icon in a template with `@icon("<name>")`.
- Dependabot updates `lucide-static`. On that pull request, run `npm ci` and `mise run gen-icons`, then commit the changed icons.

## Vocabulary

These terms are used the same way in the UI, the code and the database.

| Term            | Meaning                                                                             | Schema / code                             |
| --------------- | ----------------------------------------------------------------------------------- | ----------------------------------------- |
| **catalog**     | Game and platform data cached from IGDB. Never shown as such.                       | `games`, `platforms`, `game_platforms`    |
| **library**     | Games and DLC you log: status, rating, notes, playthroughs. Owned or not.           | `user_games` where `NOT wishlisted`       |
| **wishlist**    | Games you want. No status, no copies.                                               | `user_games` where `wishlisted`           |
| **user game**   | Code only: an entry in either the library or the wishlist.                          | `user_games`, `GetGame`, `RemoveUserGame` |
| **owned on**    | The platforms you own a game on. A property of a library game, not a separate list. | derived from the tables below             |
| **owned copy**  | One platform you own a game on.                                                     | `owned_copies`                            |
| **owned DLC**   | A DLC you own on one of your copies of the base game.                               | `owned_dlc`                               |
| **playthrough** | One attempt at playing a game, with an outcome.                                     | `playthroughs`                            |

Rules that follow from it:

- **Ownership and the library are separate.** Adding a game to the library asks where you own it, with "Not owned" as an option. DLC can be owned per copy without being in the library, and added to the library without being owned.
- **Remove vs delete.** _Remove_ takes something out of a set: a game from the library or wishlist, a copy from a game. _Delete_ destroys a record: a playthrough.
- **Status vs outcome.** A library entry has a _status_ (backlog, playing, completed, shelved, abandoned, retired). A playthrough has an _outcome_ (playing, completed, abandoned, retired); `playing` is shown as "in progress".
- **UI text** uses sentence case and "DLC" for both singular and plural.
