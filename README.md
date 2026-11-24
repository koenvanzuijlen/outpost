# Outpost

A self-hosted game tracker: log the games you play, rate them, record playthroughs, and keep track of what you own on which platform. Game data comes from [IGDB](https://www.igdb.com/).

## Running with Docker

You need a Twitch app for access to IGDB. Create one in the [Twitch developer console](https://dev.twitch.tv/console/apps) with the client type "Confidential", and copy its client ID and a client secret.

```sh
docker run -d --name outpost -p 8080:8080 --env-file .env -v outpost-data:/data koenvanzuijlen/outpost:latest
```

`.env` holds the settings below; start from [`.env.example`](.env.example). The database lives in the `outpost-data` volume. The container runs as a non-root user.

Images are published to [Docker Hub](https://hub.docker.com/r/koenvanzuijlen/outpost) for every release. Use `:1` to get updates without breaking changes.

### First start

On the first start, Outpost asks you to create the admin account. On an existing installation, your library and wishlist move to this account. After that, every page needs a signed-in user. If Outpost is reachable from the internet, serve it over HTTPS and set `SECURE_COOKIES=true`.

### Backups

Outpost stores everything in one SQLite file in `/data`. To back it up, stop the container and copy the volume. Before Outpost runs a database migration after an update, it writes a backup next to the database, for example `outpost.db.v1.bak`.

## Configuration

Outpost is configured with environment variables.

| Variable               | Default                                     | Description                                                                                                     |
| ---------------------- | ------------------------------------------- | --------------------------------------------------------------------------------------------------------------- |
| `TWITCH_CLIENT_ID`     | required                                    | Client ID of your Twitch app, for IGDB access.                                                                  |
| `TWITCH_CLIENT_SECRET` | required                                    | Client secret of your Twitch app.                                                                               |
| `TZ`                   | system time zone (UTC in Docker)            | Time zone of the server, for example `Europe/Amsterdam`. Users without an own time zone in the settings use it. |
| `DATABASE_PATH`        | `outpost.db` (`/data/outpost.db` in Docker) | Location of the SQLite database. Backups made before migrations are stored next to it.                          |
| `PORT`                 | `8080`                                      | Port to listen on. In Docker, change the host side of `-p` instead.                                             |
| `SECURE_COOKIES`       | `false`                                     | Set to `true` when Outpost is served over HTTPS. The browser then sends the session cookie only over HTTPS.     |

## Development

See [CONTRIBUTING.md](CONTRIBUTING.md) for the development setup and conventions.

## License

Outpost is licensed under the [GNU Affero General Public License v3.0 or later](LICENSE).
