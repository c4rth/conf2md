# conf2md

## Build and test

```sh
go test ./...
go build -trimpath -ldflags='-s -w' -o bin/conf2md ./cmd/conf2md
```

## Confluence

### Configuration

If present, `.env` in the current working directory is loaded automatically:

```dotenv
CONFLUENCE_URL=https://<domain>.atlassian.net
CONFLUENCE_TOKEN=your-api-token
```

These values populate the `--url` (`-u`) and `--token` (`-t`) arguments.
Explicit flags take precedence over environment variables, and existing
environment variables take precedence over `.env`. A missing `.env` is allowed;
an unreadable or malformed file reports an error. Keep API tokens out of version
control; `.env` is already ignored.

### cloud id
https://<domain>.atlassian.net/_edge/tenant_info

### token

https://id.atlassian.com/manage-profile/security/api-tokens

## References

https://github.com/Spenhouet/confluence-markdown-exporter

https://github.com/yousafgill/draw.io/