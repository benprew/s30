/**
 * Cloudflare Worker for accepting in-game bug reports and crash reports from Shandalar 30 (s30)
 * and creating GitHub Issues directly in the repository.
 *
 * Environment Variables / Secrets required in Cloudflare Worker settings:
 * - GITHUB_TOKEN: A GitHub Personal Access Token (Fine-grained token with "Issues: Read and write" permission)
 * - GITHUB_REPO_OWNER: GitHub user or organization (e.g. "benprew")
 * - GITHUB_REPO_NAME: GitHub repository name (e.g. "s30")
 * - AUTH_SECRET (Optional): A shared secret header (e.g. "X-Bug-Report-Secret") if you want to restrict requests.
 */

export default {
  async fetch(request, env) {
    // Handle CORS preflight
    if (request.method === "OPTIONS") {
      return new Response(null, {
        status: 204,
        headers: {
          "Access-Control-Allow-Origin": "*",
          "Access-Control-Allow-Methods": "POST, OPTIONS",
          "Access-Control-Allow-Headers": "Content-Type, Authorization, X-Bug-Report-Secret",
        },
      });
    }

    if (request.method !== "POST") {
      return new Response(JSON.stringify({ success: false, error: "Method not allowed" }), {
        status: 405,
        headers: { "Content-Type": "application/json" },
      });
    }

    // Optional shared secret validation
    if (env.AUTH_SECRET) {
      const authHeader = request.headers.get("X-Bug-Report-Secret");
      if (authHeader !== env.AUTH_SECRET) {
        return new Response(JSON.stringify({ success: false, error: "Unauthorized" }), {
          status: 401,
          headers: { "Content-Type": "application/json" },
        });
      }
    }

    try {
      const payload = await request.json();
      const title = payload.title || "[Bug] In-game report";
      const body = payload.body || "No description provided.";
      const isCrash = Boolean(payload.is_crash);

      const owner = env.GITHUB_REPO_OWNER || "benprew";
      const repo = env.GITHUB_REPO_NAME || "s30";
      const token = env.GITHUB_TOKEN;

      if (!token) {
        return new Response(
          JSON.stringify({ success: false, error: "GITHUB_TOKEN secret is not set in Worker environment" }),
          { status: 500, headers: { "Content-Type": "application/json" } }
        );
      }

      const labels = ["in-app-report"];
      if (isCrash) {
        labels.push("crash");
      } else {
        labels.push("bug");
      }

      const ghResponse = await fetch(`https://api.github.com/repos/${owner}/${repo}/issues`, {
        method: "POST",
        headers: {
          "Accept": "application/vnd.github+json",
          "Authorization": `Bearer ${token}`,
          "User-Agent": "s30-github-worker/1.0",
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          title,
          body,
          labels,
        }),
      });

      if (!ghResponse.ok) {
        const ghError = await ghResponse.text();
        return new Response(
          JSON.stringify({ success: false, error: `GitHub API error: ${ghResponse.status} ${ghError}` }),
          { status: 502, headers: { "Content-Type": "application/json" } }
        );
      }

      const issueData = await ghResponse.json();

      return new Response(
        JSON.stringify({
          success: true,
          issue_url: issueData.html_url,
          issue_number: issueData.number,
          message: `Created issue #${issueData.number}`,
        }),
        {
          status: 200,
          headers: {
            "Content-Type": "application/json",
            "Access-Control-Allow-Origin": "*",
          },
        }
      );
    } catch (err) {
      return new Response(
        JSON.stringify({ success: false, error: `Internal error: ${err.message}` }),
        {
          status: 500,
          headers: { "Content-Type": "application/json", "Access-Control-Allow-Origin": "*" },
        }
      );
    }
  },
};
