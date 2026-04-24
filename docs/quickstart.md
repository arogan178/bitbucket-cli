# Quick start

```bash
# 1. Sign in (interactive). Opens the browser for Cloud API tokens and stores creds in the OS keyring.
bt auth login

# 2. Pick a context if you have many Bitbucket instances.
bt context list
bt context use cloud-main

# 3. Work with repos.
bt repo list --workspace myorg
bt repo clone myorg/service && cd service

# 4. Pull requests.
bt pr list --state open
bt pr view 142
bt pr diff 142
bt pr create --title "Fix flaky probe" --body @./desc.md \
  --base main --head fix/health
bt pr checkout 142
bt pr merge 142 --strategy squash

# 5. Compare arbitrary branches (no PR required).
bt compare main...feat/kafka-38          # what's new on the branch
bt compare main...feat/kafka-38 --stat   # diffstat summary
bt compare main...feat/kafka-38 --commits --json
```

## Non-interactive / CI

```bash
export BT_TOKEN=…               # Bitbucket Cloud API token / DC PAT
export BT_EMAIL=ada@example.com  # Cloud only; use BT_USERNAME for DC
bt pr list --repo myorg/service --json
```

`bt` prefers env vars over the keyring when both are set, so CI doesn't
need keyring access.
