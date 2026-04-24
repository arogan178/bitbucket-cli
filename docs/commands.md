# Commands

Full command reference. Every list/view command supports `--json`,
`--yaml`, `--template`, and `--jq`; every command supports
`--context`, `--repo`, and `-h`.

## `bt auth`

| Command | What it does |
| --- | --- |
| `bt auth login` | Guided sign-in. Can open the browser for token creation, validates the creds, then stores them in the keyring. |
| `bt auth logout` | Remove creds for the current context. |
| `bt auth status` | Show authenticated user / host. |

## `bt context`

| Command | What it does |
| --- | --- |
| `bt context list` | List configured contexts. |
| `bt context use <name>` | Set the active context. |
| `bt context create --name … --kind cloud\|dc --host … --workspace …` | Add a context. |
| `bt context show` | Print the active context. |
| `bt context delete <name>` | Remove a context + its keyring entry. |

## `bt repo`

| Command | What it does |
| --- | --- |
| `bt repo list` | List repos in the current workspace/project. |
| `bt repo view <slug>` | Show repo details. |
| `bt repo create <slug>` | Create a repo. |
| `bt repo clone <slug>` | `git clone` the repo via HTTPS/SSH. |
| `bt repo browse <slug>` | Open the repo in the browser. |
| `bt repo delete <slug>` | Delete (prompts). |

## `bt pr`

| Command | What it does |
| --- | --- |
| `bt pr list` | List pull requests. |
| `bt pr view <id>` | Show PR metadata, description, checks. |
| `bt pr create` | Create a PR (`--title`, `--body`, `--base`, `--head`). |
| `bt pr edit <id>` | Update title/body/reviewers. |
| `bt pr merge <id>` | Merge with `--strategy=merge\|squash\|fast-forward`. |
| `bt pr decline <id>` | Decline or close without merging. |
| `bt pr approve <id>` or `bt pr unapprove <id>` | Approvals. |
| `bt pr comment <id>` | Add a comment. |
| `bt pr checks <id>` | List build statuses. |
| `bt pr checkout <id>` | Fetch + check out the PR branch. |
| `bt pr diff <id>` | Colourised, paged diff. See [Diffs](diffs.md). |

## `bt branch`

| Command | What it does |
| --- | --- |
| `bt branch list` | List branches. |
| `bt branch create <name> --from <ref>` | Create a branch. |
| `bt branch delete <name>` | Delete a branch. |
| `bt branch set-default <name>` | Change the default branch. |

## `bt compare`

See the [Diffs guide](diffs.md) for full detail.

```text
bt compare <base>..<head>      # two-dot
bt compare <base>...<head>     # three-dot (merge-base)
bt compare <base>...<head> --stat
bt compare <base>...<head> --commits [--json]
bt compare                     # default: upstream...HEAD
```

## `bt issue` *(Cloud only)*

Standard list/view/create/close/reopen/comment lifecycle.

## `bt webhook`

| Command | What it does |
| --- | --- |
| `bt webhook list` | List webhooks. |
| `bt webhook create --url … --event …` | Create a webhook. |
| `bt webhook delete <id>` | Delete. |

## `bt pipeline` *(Cloud only)*

| Command | What it does |
| --- | --- |
| `bt pipeline list` | List pipeline runs. |
| `bt pipeline view <uuid>` | Show a run. |
| `bt pipeline run --ref <branch>` | Trigger a run. |
| `bt pipeline cancel <uuid>` | Cancel. |
| `bt pipeline logs <uuid>` | Stream step logs. |

## `bt api`

Low-level escape hatch: authenticates and proxies to the configured
host's REST API.

```bash
bt api /2.0/user                     # Cloud
bt api /rest/api/1.0/application-properties   # DC
bt api /2.0/repositories/myorg --paginate --jq '.[].slug'
bt api -X POST /2.0/… --input payload.json
```
