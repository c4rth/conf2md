# conf2md

## Build and test
```sh
go test ./...
go build -trimpath -ldflags='-s -w' -o bin/conf2md ./cmd/conf2md
```

Local builds report `dev` with `--version`. To set a version manually, add
`-X main.version=1.2.3` to `-ldflags`. GoReleaser injects the release version
automatically and builds for Linux, Windows, and macOS on amd64 and arm64.

### Diagram tests

The playground is excluded from normal test runs by the `playground` build tag.
Run it explicitly from the repository root:

```
go test -tags playground ./internal/converter -run '^TestPlaygroundDiagram$' -count=1 -v
```

This requires the local fixtures in `tests/gliffy/` and overwrites their
`*_converted.drawio` outputs beside the source files.

### Regenerate emoji.json

`internal/confluenceplugin/emoji.json` is the `emoji` dictionary from the
[pymdown-extensions Emojione database](https://github.com/facelessuser/pymdown-extensions/blob/main/pymdownx/emoji1_db.py).
To download it again and overwrite `emoji.json` (requires network access):

```sh
go test -tags emojigen ./internal/confluenceplugin -run '^TestGenerateEmojiMap$' -count=1 -v
```

## Confluence

### Configuration

If present, `.env` in the current working directory is loaded automatically:

```dotenv
CONF2MD_URL=https://<domain>.atlassian.net
CONF2MD_TOKEN=your-api-token
```

All options use the same precedence: explicit flags, environment variables,
`.env`, `config.yaml` (or another Viper-supported `config` file in the current
directory), then flag defaults. Existing environment variables are not
overwritten by `.env`. Explicit boolean flags such as `--children=false`
override environment/config values.

| Flag / config key | Environment variable | Default |
|---|---|---|
| `space` | `CONF2MD_SPACE` | empty |
| `page` | `CONF2MD_PAGE` | empty (required) |
| `output` | `CONF2MD_OUTPUT` | `output` |
| `url` | `CONF2MD_URL` | empty |
| `endpoint` | `CONF2MD_ENDPOINT` | `/rest/api` |
| `token` | `CONF2MD_TOKEN` | empty |
| `children` | `CONF2MD_CHILDREN` | `true` |
| `local` | `CONF2MD_LOCAL` | `false` |
| `mkdocs` | `CONF2MD_MKDOCS` | `true` |
| `debug` | `CONF2MD_DEBUG` | `false` |
| `timeout` | `CONF2MD_TIMEOUT` | `2m` |

`timeout` limits each Confluence HTTP request, including downloading the
response body. It accepts Go durations such as `30s` or `5m`; `0` disables it.
Press Ctrl-C (or send `SIGTERM`) to cancel an in-progress export.

A missing `.env` is allowed; an unreadable or malformed file logs a warning.
Invalid configuration files stop the command with an error. Keep API tokens out
of version control; `.env` is already ignored.

### Debug logs
```
 go run ./cmd/conf2md -p <pageid> -d > log.txt 2>&1     
 ```

### cloud id
https://<domain>.atlassian.net/_edge/tenant_info

### token

https://id.atlassian.com/manage-profile/security/api-tokens

## Backstage Techdocs

### Install

```
python3 -m venv ./py-mkdocs   
source ./py-mkdocs/bin/activate
python3 -m pip install zensical mkdocs mkdocs-techdocs-core mkdocs-section-index mkdocs-git-revision-date-localized-plugin zensical
```

### View

### MkDocs
```
npx @techdocs/cli serve:mkdocs --no-docker --verbose
or
npx @techdocs/cli serve --no-docker --verbose
```

### Zensical
```
zensical serve
```

## References

https://github.com/Spenhouet/confluence-markdown-exporter

https://github.com/yousafgill/draw.io/