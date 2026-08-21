# GitHub Issues Worker for Shandalar 30 (s30)

This Cloudflare Worker receives in-game bug reports and crash reports from Shandalar 30 and opens a GitHub Issue in your repository.

## How It Works

1. **In-game Report**: The player presses **F8** (or the game encounters an unhandled panic).
2. **Payload Generation**: The client captures environment info, current world state (`save.SaveData`), and active duel state / turn history, formatted into markdown with collapsible JSON blocks.
3. **HTTP POST**: The client POSTs the JSON payload to this Cloudflare Worker.
4. **Issue Creation**: The worker uses a GitHub Personal Access Token to create a new issue with labels `["in-app-report", "bug"]` or `["in-app-report", "crash"]`.
5. **Local Backup**: The client also automatically writes the full report to `~/.s30/bug_reports/` (or `~/.s30/crashes/`) and copies the markdown to the system clipboard.

## 5-Minute Setup on Cloudflare Workers

1. Go to [Cloudflare Dashboard](https://dash.cloudflare.com/) -> **Compute (Workers & Pages)** -> **Create Application** -> **Create Worker**.
2. Paste the contents of `github_issues_worker.js` into the worker editor.
3. Go to **Settings** -> **Variables and Secrets** and add:
   - `GITHUB_TOKEN` (Secret): GitHub Personal Access Token (Fine-grained token with "Repository Permissions" -> **Issues: Read and Write** for `s30`).
   - `GITHUB_REPO_OWNER` (Variable): `benprew` (or your GitHub username / org).
   - `GITHUB_REPO_NAME` (Variable): `s30`.
4. Deploy the worker and copy your Worker URL (e.g. `https://s30-bug-report.<your-subdomain>.workers.dev`).

## Configuring the Game Client

Set the worker URL via environment variable:

```bash
export S30_BUG_WORKER_URL="https://s30-bug-report.<your-subdomain>.workers.dev"
```

Or set `bugreport.DefaultWorkerURL` in code.
