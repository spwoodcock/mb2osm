# mb2osm

A command line tool convert MBTiles format to OSMAnd SQLite format.

**This repo is mirrored to Github, but please see the original on
[Gitlab](https://gitlab.com/spwoodcock/mb2osm)**

The files should have an `.sqlitedb` extension to load into OSMAnd.

> NOTE:
>
> OSMAnd uses "BigPlanet" SQLite as supported by MOBAC.
>
> The zoom levels for this format tops out at zoom level 16,
> so instead of incrementing, it decrements the zoom level.
>
> For example zoom level `19` is actually `-2`.
>
> This detail is poorly documented!

Usage:

```bash
mb2osm [-flags] input.mbtiles output.sqlitedb

  -f    Force overwrite
  -v    Show debug logs
```

## Making a new release

- This project users Goreleaser in combination with a Gitlab workflow.
- Versions are managed manually by making a new tag / release.
- The `GITLAB_TOKEN` variable expires every year, so must be regenerated
  via Project Access Tokens, then set in Settings > CI/CD > Variables.
- Once `GITLAB_TOKEN` is set, the workflow should be able to publish
  the built artifacts to the relevant release.
