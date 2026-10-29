# Outpost

A self-hosted game tracker: log the games you play, rate them, record playthroughs, and keep track of what you own on which platform. Game data comes from [IGDB](https://www.igdb.com/).

## Running with Docker

```sh
docker run -d --name outpost -p 8080:8080 --env-file .env -v outpost-data:/data koenvanzuijlen/outpost:latest
```

`.env` holds the settings below; start from [`.env.example`](.env.example). The database lives in the `outpost-data` volume. The container runs as a non-root user.

Images are published to [Docker Hub](https://hub.docker.com/r/koenvanzuijlen/outpost) for every release. Pin a version with `:0.0.1`, or follow non-breaking updates with `:0.0`.

## Configuration

Outpost is configured with environment variables.

| Variable | Default | Description |
|---|---|---|
| `TWITCH_CLIENT_ID` | required | Twitch app credentials for IGDB access. Create an app in the [Twitch developer console](https://dev.twitch.tv/console/apps). |
| `TWITCH_CLIENT_SECRET` | required | |
| `TZ` | system time zone (UTC in Docker) | Time zone that decides what "today" is for playthrough dates, e.g. `Europe/Amsterdam`. |
| `DATABASE_PATH` | `outpost.db` (`/data/outpost.db` in Docker) | Location of the SQLite database. Backups made before migrations are stored next to it. |
| `PORT` | `8080` | Port to listen on. In Docker, change the host side of `-p` instead. |

## Development

Requires [mise](https://mise.jdx.dev/).

```sh
mise install
cp .env.example .env
mise setup
mise start
```

To build the Docker image locally:

```sh
docker build -t outpost .
```

### Vocabulary

These terms are used the same way in the UI, the code and the database.

| Term | Meaning | Schema / code |
|---|---|---|
| **catalog** | Game and platform data cached from IGDB. Never shown as such. | `games`, `platforms`, `game_platforms` |
| **library** | Games and DLC you log: status, rating, notes, playthroughs. Owned or not. | `user_games` where `NOT wishlisted` |
| **wishlist** | Games you want. No status, no copies. | `user_games` where `wishlisted` |
| **user game** | Code only: an entry in either the library or the wishlist. | `user_games`, `GetGame`, `RemoveUserGame` |
| **owned on** | The platforms you own a game on. A property of a library game, not a separate list. | derived from the tables below |
| **owned copy** | One platform you own a game on. | `owned_copies` |
| **owned DLC** | A DLC you own on one of your copies of the base game. | `owned_dlc` |
| **playthrough** | One attempt at playing a game, with an outcome. | `playthroughs` |

Rules that follow from it:

- **Ownership and the library are separate.** Adding a game to the library asks where you own it, with "Not owned" as an option. DLC can be owned per copy without being in the library, and added to the library ("Add to library separately") without being owned.
- **Remove vs delete.** *Remove* takes something out of a set: a game from the library or wishlist, a copy from a game. *Delete* destroys a record: a playthrough.
- **Status vs outcome.** A library entry has a *status* (backlog, playing, completed, shelved, abandoned, retired). A playthrough has an *outcome* (playing, completed, abandoned, retired); `playing` is shown as "in progress".
- **UI text** uses sentence case and "DLC" for both singular and plural.

## License

Outpost is licensed under the [GNU Affero General Public License v3.0 or later](LICENSE).
