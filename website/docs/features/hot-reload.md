# Hot Reload

[Home](/) > [Features](index) > Hot Reload

---

mockr watches the config file or directory for changes using filesystem notifications. Edit a case, change a `fallback`, or drop a new `.toml` file into the config directory — the next request picks up the changes with no restart.

## How it works

- mockr uses `fsnotify` to watch all config files
- When a file change is detected, the config is reloaded before the next request
- No server restart, no connection drops, no downtime

## What triggers a reload

- Editing any config file (`.toml`, `.yaml`, `.json`)
- Adding a new config file to the config directory
- Removing a config file from the config directory
- Renaming a config file

## Terminal output

mockr logs indicate where each response comes from:

```
via=stub   — response served from a local mock file
via=proxy  — response forwarded to the upstream API
```

## Notes

- [Response transitions](response-transitions.md) are **reset** when a hot reload occurs — the timeline restarts from the beginning
- Stub files (in `stubs/`) are **not watched** — they are read on each request, so changes to stub files always take effect immediately

---

**See also:** [Configuration](../configuration/) | [Response Transitions](response-transitions.md)
