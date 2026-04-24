# Contributing to bt

Thanks for your interest! `bt` aims to be a small, boring, reliable CLI
for Bitbucket. Contributions that keep it that way are very welcome.

## Ground rules

- One feature per PR. Keep commits focused and message them in the
  imperative mood ("add foo", not "added foo").
- All new code ships with tests, or a clear explanation why not.
- Be kind — see the [Code of Conduct](./CODE_OF_CONDUCT.md).

## Dev loop

```
# run tests
go test ./...

# run the CLI during development
go run ./cmd/bt pr list

# lint
golangci-lint run

# release snapshot (no publish)
goreleaser release --snapshot --clean
```

## Project layout

```
cmd/bt                # main entry point
internal/cli          # cobra subcommands (gh-style)
internal/bitbucket    # backend interface + Cloud/DC implementations
internal/config       # on-disk config + contexts
internal/auth         # keyring + env fallback
internal/output       # table / JSON / YAML / template / jq / diff
```

## Adding a new Bitbucket endpoint

1. Add the types in `internal/bitbucket/types.go`.
2. Add the method to the appropriate service interface in
   `internal/bitbucket/client.go`.
3. Implement it for **both** `cloud.go` and `datacenter.go`. If DC or
   Cloud genuinely doesn't support it, return a clear `ErrUnsupported`.
4. Wire a `cobra.Command` in `internal/cli/…`.
5. Add an entry to `docs/commands.md` and to the README if
   user-visible.

## Testing against a real Bitbucket

There are integration tests gated behind `-tags=integration`. You need
`BT_TEST_WORKSPACE`, `BT_TEST_REPO`, and a token in `BT_TOKEN`:

```
go test -tags=integration ./internal/bitbucket/...
```

Never hit a live instance from unit tests; use `httptest.Server`.

## Releases

Maintainers push a `v*` tag; GitHub Actions + GoReleaser handle the
rest (brew tap, scoop bucket, GHCR image, signed checksums, SBOMs).
