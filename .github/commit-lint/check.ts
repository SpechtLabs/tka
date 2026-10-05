// Checks that release-please can read the commit a squash merge of this pull
// request would create.
//
// Squash merges take the PR title as the subject and the PR description as the
// body. release-please parses the whole message, and when the parser throws it
// logs the error at debug level and drops the commit: no changelog entry, no
// version bump, and a green run. A description with a line that starts like
// `assert("x",` or `f(g("a"))` is enough, because the parser reads it as a
// `type(scope):` footer.
//
// This runs release-please's own parseConventionalCommits, pinned in
// package.json to the version the release-please action bundles, so it applies
// BEGIN_COMMIT_OVERRIDE and splits nested commits exactly as a release does.
//
// Reads PR_NUMBER, PR_TITLE and PR_BODY from the environment.

import { parseConventionalCommits, type Commit } from "release-please/build/src/commit.js";
import type { Logger } from "release-please/build/src/util/logger.js";

const title = process.env.PR_TITLE ?? "";
const body = (process.env.PR_BODY ?? "").replace(/\r\n/g, "\n");
const number = Number(process.env.PR_NUMBER ?? 0);

const failures: string[] = [];
const ignore = () => {};
const logger: Logger = {
  error: ignore,
  warn: ignore,
  info: ignore,
  trace: ignore,
  debug: (msg: unknown) => {
    const line = String(msg);
    // The parser names the token it choked on, and that token is often a raw
    // newline, which would end the ::error annotation early.
    if (line.startsWith("error message: ")) {
      failures.push(line.slice("error message: ".length).replace(/\n/g, "\\n"));
    }
  },
};

const commit: Commit = {
  sha: "squash",
  message: body.trim() ? `${title}\n\n${body}` : title,
  files: [],
  pullRequest: {
    headBranchName: "",
    baseBranchName: "main",
    number,
    title,
    body,
    labels: [],
    files: [],
  },
};

const parsed = parseConventionalCommits([commit], logger);

for (const error of failures) {
  console.log(`::error title=release-please cannot parse this commit::${error}`);
}
if (failures.length > 0 && body.includes("BEGIN_COMMIT_OVERRIDE")) {
  console.log(`
release-please would drop this commit from the changelog and the version bump.
The description contains BEGIN_COMMIT_OVERRIDE, so release-please parses only
the text between it and END_COMMIT_OVERRIDE (or the end of the description),
and the error's line:column counts from there. If the marker is only mentioned
in prose, reword it: release-please treats any occurrence as the override.`);
} else if (failures.length > 0) {
  console.log(`
release-please would drop this commit from the changelog and the version bump.
The error's line:column counts from the commit subject (the PR title) as line 1,
and the body starts on line 3. Indent the offending line, or wrap it in text so
it no longer starts with \`identifier(\`. Code in a fenced block is not exempt.`);
}

for (const c of parsed) {
  const scope = c.scope ? `(${c.scope})` : "";
  console.log(`release-please reads: ${c.type}${scope}${c.breaking ? "!" : ""}: ${c.bareMessage}`);
}
if (parsed.length > 1) {
  console.log(
    `::notice title=More than one commit::release-please reads ${parsed.length} commits from this message: every paragraph that starts with a conventional type becomes its own changelog entry.`,
  );
}

process.exit(failures.length > 0 || parsed.length === 0 ? 1 : 0);
