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

- **Cloud**: username + app password (create at
  <https://bitbucket.org/account/settings/app-passwords/>).
- **DC**: personal access token (Profile → Manage account → Personal
  access tokens).

### Environment-variable fallback

For CI or headless boxes where the keyring isn't practical:

| Variable | Meaning |
| --- | --- |
| `BT_TOKEN` | DC personal access token, or Cloud app password. |
| `BT_USERNAME` | Cloud username or DC username. |
| `BT_APP_PASSWORD` | Cloud app password (alias for `BT_TOKEN` under Cloud). |
| `BT_PAT` | DC personal access token (alias under DC). |
| `BT_EMAIL` | Optional; used only for commit authoring helpers. |

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
