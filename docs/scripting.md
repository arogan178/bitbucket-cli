# Scripting

Every `bt` list/view command supports the same output flags:

| Flag | Behaviour |
| --- | --- |
| *(default)* | Human-readable table. |
| `--json` | JSON array / object. |
| `--yaml` | YAML. |
| `--template 'tmpl'` | Go `text/template`. |
| `--jq 'expr'` | Filter via [`gojq`](https://github.com/itchyny/gojq). |

## Examples

```bash
# IDs of every open PR
bt pr list --state open --jq '.[].id'

# Table of (id, title, age) using Go template
bt pr list --template \
  '{{range .}}{{.ID}}	{{.Title}}	{{timeSince .CreatedOn}}{{"\n"}}{{end}}'

# Pipe JSON straight into jq
bt pr list --json | jq 'group_by(.author.display_name) | map({(.[0].author.display_name): length}) | add'
```

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | Success. |
| `1` | Expected error (API 4xx, validation, etc.). |
| `2` | Usage error (bad flags). |
| `3` | Auth missing / invalid. |
| `4` | Transient network / 5xx (safe to retry). |

Use them for retry logic:

```bash
until bt pr merge 142 --strategy squash; do
  case $? in
    4) sleep 30;;
    *) exit 1;;
  esac
done
```

## Pagination

List commands auto-paginate by default. Use `--limit N` to cap, or
`--page N` to start from a specific page. `bt api --paginate` does the
same for raw API calls and emits a single concatenated JSON array.
