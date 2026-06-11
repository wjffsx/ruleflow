# Hot-Reload Example

Demonstrates configuration hot-reload: RuleFlow watches a YAML file via fsnotify and applies changes atomically without restarting the process.

## Run

```bash
go run main.go
```

## What it covers

- **FileWatcher**: `config.NewFileWatcher()` — monitors a single YAML file for changes.
- **DirWatcher**: `config.NewDirWatcher()` — monitors a directory for `*.yaml`/`*.yml`/`*.json` files.
- **ChainReloader**: A callback function invoked on each reload, giving you a hook to validate or log new chains.
- **Two-phase parsing**: Config → intermediate representation → Registry-backed instantiation. The intermediate phase is zero-dependency and suitable for editor preview.
- **Debounce**: Built-in 200ms debounce prevents rapid-reload thrashing.

## How to test

While the program is running, edit the temporary YAML file. The engine picks up changes automatically and applies them via copy-on-write — no read-side locking needed.

## Key takeaway

For production deployments, use `FileWatcher` or `DirWatcher` to decouple rule configuration from application code. Hot-reload uses atomic COW snapshots; in-flight evaluations complete on the old snapshot.
