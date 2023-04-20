Assets

# SYNOPSIS

./assets

```
[--dir-perm]=[value]
[--dir]=[value]
[--dsn]=[value]
[--file-perm]=[value]
[--help|-h]
[--http-user-agent]=[value]
[--max-remote-size]=[value]
[--max-size]=[value]
[--original-url-pattern]=[value]
[--path-depth]=[value]
```

# DESCRIPTION

Asset storage service

**Usage**:

```
./assets [GLOBAL OPTIONS] command [COMMAND OPTIONS] [ARGUMENTS...]
```

# GLOBAL OPTIONS

**--dir**="": Directory to store asset files.
Example: `./storage`.

Environment variable: `ASSETS_DIR`.

**--dir-perm**="": Permission flags for new directories within a tree.
Default: `0755`.

Environment variable: `ASSETS_DIR_PERM`.

**--dsn**="": Data source name (only sqlite3 is supported for now).
Example: `sqlite3:./storage/assets.db?mode=rwc&_journal=TRUNCATE`.

Environment variable: `ASSETS_DSN`.

**--file-perm**="": Permission flags for new files within a tree.
Default: `0655`.

Environment variable: `ASSETS_FILE_PERM`.

**--help, -h**: Show help.

**--http-user-agent**="": User-Agent header string used by HTTP client
when fetching remote resources.
Default: `AssetsClient`.

Environment variable: `ASSETS_HTTP_USER_AGENT`.

**--max-remote-size**="": Size limit for resources fetched by URL.
Default: `1073741824` (1GiB).

Environment variable: `ASSETS_MAX_REMOTE_SIZE`.

**--max-size**="": Size limit for resources pushed directly.
Default: `0` (no limit).

Environment variable: `ASSETS_MAX_SIZE`.

**--original-url-pattern**="": RegExp pattern to check URLs
before fetch.
Example: `^https?://.`. If not set, download by original URL
is disabled.

Environment variable: `ASSETS_ORIGINAL_URL_PATTERN`.

**--path-depth**="": Directory tree depth.
Default: `2`.

Environment variable: `ASSETS_PATH_DEPTH`.


# COMMANDS

## migrate

Apply migrations on current database.

## http

Start pure HTTP server.

**--bind**="": Address to bind HTTP server.
Default: `:8080`.

Environment variable: `ASSETS_HTTP_BIND`.

**--fallback-mimetype**="": Fallback value for response
Content-Type header.
Default: `application/octet-stream`.

Environment variable: `ASSETS_HTTP_FALLBACK_MIMETYPE`.

## storeurls

Store assets by original URLs.

Provide URLs after the command name:

```bash
./assets storeurls http://example.com/1.jpg 'http://example.com/dl?n=foobar'
```

You may feed a dash instead of URL if you want to pass URLs to stdin:

```bash
cat urls.lst | ./assets storeurls -'
```

## storefiles

Store local files as assets.

Provide file names after the command name:

```bash
./assets storefiles ~/image1.jpg ~/image2.jpg video.mp4
```

You may feed a dash instead of file name if you want
to pass file names to stdin:

```bash
find -type f -iname '*.jpg' | ./assets storefiles -
```

## storepipe

Read stdin and store the data as an asset.

**--content-type, --type, --mime**="": value for asset's
content_type field.

**--info**="": value for asset's info field.

**--original-name, --name**="": value for asset's
original_name field.

**--original-url, --url**="": value for asset's
original_url field.

```bash
ffmpeg -i foo.avi <options> -f mp4 - | ./assets storepipe --original-name foo.mp4 --content-type video/mp4
```

## help, h

Shows a list of commands or help for one command.
