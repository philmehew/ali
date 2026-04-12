# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in ali, please report it privately:

- **Email:** phil.mehew@gmail.com

Please do not open a public GitHub issue for security vulnerabilities.

## What to include

- A description of the vulnerability
- Steps to reproduce
- The affected version (run `ali version`)
- Any potential impact

## Response timeline

I aim to acknowledge reports within 48 hours and provide a fix or mitigation within 7 days.

## Known considerations

- ali executes command bodies via `/bin/sh -c`. This is by design — it enables pipes, redirects, and shell features. Users should only add functions they trust.
- The config file (`functions.yaml`) is stored in the user's platform config directory with default file permissions. It contains no secrets — only command snippets and their defaults.
- ali does not make network requests or phone home. It is a purely local tool.
