# How-to: Publish to the MCP Registry

> **Get dino-mcp discoverable at [registry.modelcontextprotocol.io](https://registry.modelcontextprotocol.io).**
> This is the literal, worked record of the `v0.1.0` publish for `io.github.shennawardana23/mcp-dino` —
> real commands, real values, real failures hit along the way and how they were fixed. Not a
> generic template with placeholders.

**Official references:**
- [MCP Registry — GitHub](https://github.com/modelcontextprotocol/registry)
- [MCPB Bundle spec — GitHub](https://github.com/modelcontextprotocol/mcpb)
- [server.json schema](https://static.modelcontextprotocol.io/schemas/2025-12-11/server.schema.json)

---

## Prerequisites

| Tool | Check | Install |
|------|-------|---------|
| `mcp-publisher` | `mcp-publisher --version` | see [registry releases](https://github.com/modelcontextprotocol/registry/releases) |
| `gh` (GitHub CLI) | `gh auth status` | `brew install gh` |
| GitHub auth matching your registry namespace | `gh auth status` shows `shennawardana23` active | `gh auth login` |

`io.github.<user>/<name>` namespaces authenticate via GitHub OAuth device flow — no domain/DNS proof needed. That only matters for `com.example.*`-style custom-domain namespaces.

---

## Step-by-Step (as actually run)

### 1. Clean the workspace before committing

`git status -uall` first. Excluded: `dino-mcp copy` (22MB stray binary), `go copy.mod` (stale module path from before the `mcp-dino` rename), `README copy.md` (turned out to be the *real* README, content recovered into `README.md` rather than deleted), local tool caches `.genkit/` and `.impeccable/` (added to `.gitignore`, not committed).

### 2. Commit real source

One clean commit (`c6cca8c`): `internal/`, `cmd/`, `ui/`, `docs/`, `Makefile`, `.github/`, restored `README.md`. `server.json` and `mcp-exporter/` (a separate, still-in-progress sibling project) held back at this point.

### 3. Fix version injection before tagging

`Makefile` and `.github/workflows/release.yml` both had:

```
-ldflags="-X github.com/msw/dino-mcp/internal/server.Version=$(VERSION)"
```

but `go.mod` declares `module mcp-dino` — a different import path, left over from before a module rename. `-X` silently no-ops when the path doesn't resolve; every build reported the hardcoded `0.1.0` regardless of the actual tag. Fixed to `-X mcp-dino/internal/server.Version=...` in both files, then proved it:

```bash
go build -ldflags="-X github.com/msw/dino-mcp/internal/server.Version=v9.9.9-TEST" -o /tmp/t ./cmd/dino-mcp
/tmp/t stdio -version              # printed "dino-mcp 0.1.0" — confirmed the bug

go build -ldflags="-X mcp-dino/internal/server.Version=v9.9.9-TEST" -o /tmp/t ./cmd/dino-mcp
/tmp/t stdio -version              # printed "dino-mcp v9.9.9-TEST" — confirmed the fix
```

Committed as `484f897`.

### 4. Push

```bash
git push origin main
```

### 5. Tag and push the tag

```bash
git tag -a v0.1.0 -m "v0.1.0: initial public release"
git push origin v0.1.0
```

**First attempt failed.** The release workflow's "Create release" step errored: `Resource not accessible by integration`. Root cause, confirmed via:

```bash
gh api repos/shennawardana23/mcp-dino/actions/permissions/workflow
# {"default_workflow_permissions":"read", ...}
```

Fix — scoped to just the one job, not a repo-wide setting change:

```yaml
jobs:
  goreleaser:
    permissions:
      contents: write
```

Committed (`37beb52`), pushed, then the tag had to be moved to include the fix:

```bash
git tag -d v0.1.0 && git push origin :refs/tags/v0.1.0
git tag -a v0.1.0 -m "v0.1.0: initial public release"
git push origin v0.1.0
```

Second run went green: `gh run watch 32152423192 --exit-status`.

### 6. Build the `.mcpb` bundle

Go has no native `registryType` in the registry schema (only `npm`, `pypi`, `oci`, `nuget`, `mcpb`). For a compiled binary, `mcpb` (MCP's own bundle format) fits — no Docker, no package-manager account.

Structure (per the real `calculator-rust` example in [modelcontextprotocol/mcpb](https://github.com/modelcontextprotocol/mcpb/tree/main/examples/calculator-rust) — a Rust binary, same shape as a Go one):

```
dino-mcp-darwin-arm64.mcpb (a zip file)
├── manifest.json
└── server/
    └── dino-mcp
```

`manifest.json` — **`server.type` is `"binary"`**, not `"command"` (one AI-summarized source got this wrong; the real JSON Schema at `schemas/mcpb-manifest-latest.schema.json` in that repo is ground truth — checked directly via `gh api`):

```json
{
  "manifest_version": "0.3",
  "name": "dino-mcp",
  "display_name": "Dino MCP",
  "version": "0.1.0",
  "description": "MCP server with an interactive dinosaur dashboard (MCP Apps UI)",
  "author": { "name": "shennawardana23" },
  "server": {
    "type": "binary",
    "entry_point": "server/dino-mcp",
    "mcp_config": {
      "command": "${__dirname}/server/dino-mcp",
      "args": ["stdio"],
      "env": {}
    }
  },
  "tools": [
    { "name": "dino_think", "description": "Returns a random dinosaur fact and the species it's about." },
    { "name": "dino_ask", "description": "Ask a dinosaur-related question and get an informative answer." },
    { "name": "dino_dashboard", "description": "Opens an interactive dinosaur dashboard with filterable visual cards (MCP App)." }
  ],
  "license": "MIT",
  "compatibility": { "platforms": ["darwin"] }
}
```

Downloaded the actual released binary (not the local build) to package it — proves the artifact being shipped is the one CI actually built and released:

```bash
gh release download v0.1.0 --repo shennawardana23/mcp-dino --pattern "dino-mcp-darwin-arm64"
mkdir -p /tmp/mcpb-build/server
cp dino-mcp-darwin-arm64 /tmp/mcpb-build/server/dino-mcp && chmod +x /tmp/mcpb-build/server/dino-mcp
cp manifest.json /tmp/mcpb-build/
cd /tmp/mcpb-build && zip -r ../dino-mcp-darwin-arm64.mcpb manifest.json server/
```

### 7. Attach it to the release

```bash
gh release upload v0.1.0 /tmp/dino-mcp-darwin-arm64.mcpb --repo shennawardana23/mcp-dino
```

### 8. Point `server.json` at it, with the hash it will demand

```json
{
  "$schema": "https://static.modelcontextprotocol.io/schemas/2025-12-11/server.schema.json",
  "name": "io.github.shennawardana23/mcp-dino",
  "description": "MCP server with an interactive dinosaur dashboard (MCP Apps UI). No API keys required.",
  "repository": { "url": "https://github.com/shennawardana23/mcp-dino", "source": "github" },
  "version": "0.1.0",
  "packages": [{
    "registryType": "mcpb",
    "identifier": "https://github.com/shennawardana23/mcp-dino/releases/download/v0.1.0/dino-mcp-darwin-arm64.mcpb",
    "version": "0.1.0",
    "fileSha256": "f4e4445f8f881d5df59f0c257b06f04b661a05b9e37eb494b4ea40707f7ba8d7",
    "transport": { "type": "stdio" }
  }]
}
```

**Gotcha:** `mcp-publisher validate` does *not* check `fileSha256` or that the identifier URL resolves — it only checks the schema shape. `mcp-publisher publish` *does* check, and rejected the first attempt with `must include a fileSha256 hash for integrity verification`. Computed against the actual uploaded asset (downloaded fresh, not reused from the local build, to be certain they're byte-identical):

```bash
curl -sL "https://github.com/shennawardana23/mcp-dino/releases/download/v0.1.0/dino-mcp-darwin-arm64.mcpb" -o /tmp/verify.mcpb
shasum -a 256 /tmp/verify.mcpb
# f4e4445f8f881d5df59f0c257b06f04b661a05b9e37eb494b4ea40707f7ba8d7  — matched the local build
```

### 9. Log in

```bash
mcp-publisher login github
```

Prints a device-code flow: visit `https://github.com/login/device`, enter the one-time code shown, authorize. This proves ownership of the `io.github.shennawardana23` namespace via GitHub OAuth.

### 10. Validate, then publish

```bash
mcp-publisher validate
mcp-publisher publish
```

Result:
```
✓ Successfully published
✓ Server io.github.shennawardana23/mcp-dino version 0.1.0
```

---

## Verify It's Live

**The easiest way — the actual public website, no login needed:**

Visit **https://registry.modelcontextprotocol.io/** and type `mcp-dino` into the "Search servers by name..." box (URL becomes `?q=mcp-dino`). Confirmed in a real browser: shows `io.github.shennawardana23/mcp-dino v0.1.0`, correct description, "Updated 8/18/2026", "Showing 1 servers". Anyone in the world can do this, right now, with no auth.

Don't trust the CLI's own success message alone either — the API can be queried independently:

```bash
curl -s "https://registry.modelcontextprotocol.io/v0/servers?search=shennawardana23" | python3 -m json.tool
```

Look for `"status": "active"` and a fresh `publishedAt` timestamp under `_meta["io.modelcontextprotocol.registry/official"]`. This is the canonical, protocol-level registry — the one `mcp-publisher` actually writes to, and the one any MCP-aware client or install tool should be reading from.

**Note on searching by name:** the raw REST API's `search=` param (`/v0/servers?search=mcp-dino`) returned zero results, and so did `search=dino` — only `search=shennawardana23` (the namespace owner) worked there. The *website's* own search box uses a different param (`?q=`) and matched `mcp-dino` correctly by name. If the raw API's `search=` doesn't hit, try the website's `q=` or search by owner instead.

**`github.com/mcp` is a *different*, GitHub-curated directory** (220+ servers shown, e.g. Notion, Playwright, GitHub's own server) — not the same as `registry.modelcontextprotocol.io`, and its sourcing/sync process isn't publicly documented. Searching `github.com/mcp?search=mcp-dino` (or `dino`, or `shennawardana23`) returned **"No MCPs found"** immediately after a confirmed-live registry publish. That is *not* evidence the publish failed — it's a separate index with unknown criteria and/or lag. The registry API above is the authoritative check; treat `github.com/mcp` as a bonus discovery surface that may catch up later, not a pass/fail gate.

### Testing the published server actually runs

The registry entry only proves the metadata is correct — it doesn't run the server for you. To confirm the shipped `.mcpb` actually works end to end:

```bash
curl -sL "https://github.com/shennawardana23/mcp-dino/releases/download/v0.1.0/dino-mcp-darwin-arm64.mcpb" -o /tmp/dino.mcpb
mkdir -p /tmp/dino-test && cd /tmp/dino-test && unzip /tmp/dino.mcpb
chmod +x server/dino-mcp
./server/dino-mcp stdio -version    # should print "dino-mcp v0.1.0"
```

Then drive it through MCP Inspector the same way as local development (see [`test-inspector.md`](test-inspector.md)) — point Inspector's stdio transport at `/tmp/dino-test/server/dino-mcp stdio` and confirm `tools/list`, a `dino_think` call, and the `dino_dashboard` App tab all still work from the packaged binary, not just the dev build.

---

## Understanding `fileSha256`

`server.json`'s package entry carries a hash:

```json
"fileSha256": "f4e4445f8f881d5df59f0c257b06f04b661a05b9e37eb494b4ea40707f7ba8d7"
```

**What it is.** A SHA-256 hash of the exact `.mcpb` file's bytes — a fingerprint. The same input file always produces the same 64-character hex string; changing even one byte produces a completely different hash.

**What it's for.** Integrity verification. When Claude Desktop, `mcp-publisher`, or any install tool downloads the bundle from the release URL, it re-hashes the bytes it received and compares them to this value. Match → the file arrived intact, install proceeds. Mismatch → reject. This guards against a corrupted download, a swapped/tampered release asset, or a MITM substituting a different binary at that URL.

`mcp-publisher publish` actually enforces this — it's the exact error hit during step 8:
```
must include a fileSha256 hash for integrity verification
```
It refuses to publish an `mcpb` package entry without one. (`mcp-publisher validate` does *not* check it — only `publish` does.)

**How to generate it:**
```bash
shasum -a 256 dino-mcp-darwin-arm64.mcpb
# f4e4445f8f881d5df59f0c257b06f04b661a05b9e37eb494b4ea40707f7ba8d7  dino-mcp-darwin-arm64.mcpb
```
Take just the hash (before the two spaces).

**Gotcha:** hash the file you actually uploaded to the release, not just your local build directory — they must be byte-identical. Verify by downloading it back and hashing that:
```bash
curl -sL "https://github.com/shennawardana23/mcp-dino/releases/download/v0.1.0/dino-mcp-darwin-arm64.mcpb" -o /tmp/verify.mcpb
shasum -a 256 /tmp/verify.mcpb
```
If you ever rebuild and re-upload the `.mcpb` (bug fix, new binary), the file bytes change → the hash changes → `server.json`'s `fileSha256` must be regenerated and republished, or every future install will fail the integrity check.

---

## Use It in Claude Desktop

Two ways to actually run the published server, not just see its registry listing.

### A. Point at your local build (fastest for development)

Edit `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "dino-mcp": {
      "command": "/absolute/path/to/mcp-dino/bin/dino-mcp",
      "args": ["stdio"]
    }
  }
}
```

Restart Claude Desktop. A hammer icon (🔨) appears in chat when tools are available.

### B. Install the published `.mcpb` (the real end-user flow)

This is the artifact that's actually live on the registry — installing it this way exercises the whole publish, not just the dev binary.

```bash
curl -sL "https://github.com/shennawardana23/mcp-dino/releases/download/v0.1.0/dino-mcp-darwin-arm64.mcpb" -o ~/Downloads/dino-mcp.mcpb
```

In Claude Desktop: open **Settings**, then drag `dino-mcp.mcpb` into the Settings window. Per Anthropic's Desktop Extensions documentation, this triggers an install flow showing the extension's name/description, required permissions, and an **Install** button — no manual config file editing, no restart-and-hope.

### Try it

In chat, once either path is set up:

```
Show me the dinosaur dashboard with carnivores
```

Claude detects `_meta.ui.resourceUri` on `dino_dashboard`, calls `resources/read`, and renders the iframe. Recommend path **B** first since it tests the exact thing now live on the registry, not just a local dev build.

---

## What to Check for Each Issue

| Symptom | Likely Cause | Fix |
|---------|-------------|-----|
| Release workflow fails at "Create release" | Repo's default Actions permission is `read` | Add `permissions: contents: write` to the job, re-tag |
| `mcp-publisher publish` rejects with "must include a fileSha256" | `validate` doesn't check this, only `publish` does | `shasum -a 256` the actual uploaded asset, add `fileSha256` to `server.json` |
| Registry `search=<repo-name>` returns nothing after a successful publish | Search indexing doesn't reliably substring-match the server name | Search by your GitHub username instead |
| `github.com/mcp?search=...` shows "No MCPs found" right after publishing | Separate GitHub-curated directory, undocumented sync/criteria | Check `registry.modelcontextprotocol.io` directly — that's the authoritative source |
| `mcp-publisher init` picks `registryType: npm` with a fake `YOUR_API_KEY` env var | It's a generic template, not aware this is a Go binary | Replace with `mcpb` (see step 6), delete the env var block |
| `server.json` description rejected with `expected length <= 100` | Registry enforces a hard cap the local schema summary doesn't always surface | Shorten it |

---

## Related

- [`how-to/test-inspector.md`](test-inspector.md) — verify the server works before publishing it
- [`reference/cli.md`](../reference/cli.md) — CLI flags and version flag
- [`../../ARCHITECTURE.md`](../../ARCHITECTURE.md) — deployment topology
