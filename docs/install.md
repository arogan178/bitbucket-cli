# Install

## Homebrew (macOS, Linux)

```bash
brew install arogan178/bitbucket-cli/bt
```

## Scoop (Windows)

```powershell
scoop bucket add bitbucket-cli https://github.com/arogan178/scoop-bitbucket-cli
scoop install bt
```

## Go

```bash
go install github.com/arogan178/bitbucket-cli/cmd/bt@latest
```

## Docker / OCI

```bash
docker run --rm -it ghcr.io/arogan178/bitbucket-cli:latest --help
```

Images are multi-arch (`amd64`, `arm64`) and distroless. They're signed
with cosign keyless; verify with:

```bash
cosign verify ghcr.io/arogan178/bitbucket-cli:latest \
  --certificate-identity-regexp 'https://github.com/arogan178/bitbucket-cli/' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```

## Binary downloads

Every release on [GitHub Releases](https://github.com/arogan178/bitbucket-cli/releases)
ships signed checksums (`*_checksums.txt`, `*.sig`, `*.pem`) and SBOMs.
