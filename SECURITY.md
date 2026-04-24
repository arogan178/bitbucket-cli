# Security Policy

## Supported versions

We support the latest minor release on `main`. Older tags are not
patched.

## Reporting a vulnerability

Please **do not** open a public issue. Email andrea.bugeja@hotmail.com
with:

- a description of the issue and its impact,
- the commit or version affected,
- reproduction steps (a minimal script or `bt api` trace is ideal), and
- whether you want public credit.

We aim to acknowledge reports within 2 business days and publish a fix
or mitigation within 30 days for confirmed issues.

## Credentials

`bt` stores credentials in the OS keyring by default. On headless CI,
prefer short-lived tokens via environment variables (`BT_TOKEN`, etc.)
and never commit `~/.config/bt/config.yaml` — though that file contains
no secrets, it does record the host and workspace names.
