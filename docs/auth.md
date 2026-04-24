# Contexts & auth

`bt` models each Bitbucket instance as a **context**. You can mix Cloud
and Data Center contexts in the same install and switch between them.

## Creating contexts

```bash
bt context create \
  --name cloud-main \
  --kind cloud \
  --host api.bitbucket.org \
  --workspace myorg

bt context create \
  --name dc-internal \
  --kind dc \
  --host bitbucket.internal.example \
  --project ENG
```

Switch with `bt context use <name>`, inspect with `bt context show`.

## Credentials

`bt auth login` stores credentials in the OS keyring under the service
`bt:<context-name>`:

- **Cloud**: Atlassian account email + API token. `bt auth login` can open
  the browser directly to
  <https://id.atlassian.com/manage-profile/security/api-tokens>.
  Bitbucket Cloud app passwords are deprecated and stop working on
  `2026-06-09`.
- **DC**: personal access token (Profile → Manage account → Personal
  access tokens).

### Environment-variable fallback

For CI or headless boxes where the keyring isn't practical:

| Variable | Meaning |
| --- | --- |
| `BT_TOKEN` | Cloud API token or DC personal access token. |
| `BT_EMAIL` | Cloud Atlassian account email (used with `BT_TOKEN`). |
| `BT_USERNAME` | DC username, or Cloud username only for legacy app-password mode. |
| `BT_APP_PASSWORD` | Deprecated Cloud app password. |
| `BT_PAT` | DC personal access token (alias under DC). |

Env vars win when both are set — handy for impersonating a bot in CI
without touching the keyring.

## Config file

`~/.config/bt/config.yaml` (or `$XDG_CONFIG_HOME/bt/config.yaml`):

```yaml
active: cloud-main
contexts:
  cloud-main:
    kind: cloud
    host: api.bitbucket.org
    workspace: myorg
  dc-internal:
    kind: dc
    host: bitbucket.internal.example
    project: ENG
```

Never contains secrets.
