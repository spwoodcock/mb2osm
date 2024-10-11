# mb2osm

A command line tool convert MBTiles format to OSMAnd SQLite format.

**This repo is mirrored to Github, but please see the original on
[Gitlab](https://gitlab.com/spwoodcock/mb2osm)**

The files should have an `.sqlitedb` extension to load into OSMAnd.

Credit to @tarwirdur for Python code / inspiration.

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
  -quality int
        JPEG quality 0-100 (default 80)
  -v    Show debug logs
```
