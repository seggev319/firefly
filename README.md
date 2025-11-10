## Firefly (Go bootstrap)

**Quick start**

1. Build:
   ```bash
   make build
   ```
2. Run (default port 8080):
   ```bash
   make run
   ```
   Or:
   ```bash
   PORT=9090 ./bin/firefly
   ```
3. Test:
   ```bash
   make test
   ```

**Endpoints**
- GET `/` → simple text
- GET `/healthz` → `{"status":"ok"}`

**Configuration**
- **PORT**: listening port (default `8080`)

**Version metadata**
Injected at build time (via `-ldflags`):
- `pkg/version.Version`
- `pkg/version.Commit`
- `pkg/version.BuiltAt`

**Docker**
Build and run:
```bash
docker build -t firefly:latest .
docker run --rm -p 8080:8080 -e PORT=8080 firefly:latest
```


