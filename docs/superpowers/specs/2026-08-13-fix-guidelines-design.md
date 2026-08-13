# Fix Guidelines Design

## Purpose

Roborev's Agent Hook automatically brings failed reviews back into an active
coding-agent session. Its default instruction and the prompts used by
`roborev fix` currently tell agents to address all findings. Users need a
supported way to add policy that tells autofix agents how to evaluate findings,
including when a suggestion is a false positive or conflicts with project
intent.

Add one shared `fix_guidelines` setting. It supplies user guidance to every
autofix agent reached from Agent Hook, including agents launched later through
`roborev fix`, while leaving the existing automatic workflow unchanged when the
setting is empty.

## Configuration

`fix_guidelines` is a multiline string accepted at the top level of both the
global config and `.roborev.toml`:

```toml
fix_guidelines = """
Treat review findings as hypotheses. Verify each finding against the code and
project requirements before changing anything. Explain findings that are not
applied in the review comment.
"""
```

Resolution follows the repository's ordinary repo-over-global precedence:

1. A non-empty repo `fix_guidelines` value replaces the global value.
2. Otherwise, a non-empty global value is used.
3. Otherwise, the effective value is empty.

There is no merge mode, supersede flag, Agent Hook flag, or Agent Hook-specific
environment variable. This keeps one authoritative value visible to the active
hook session and to a later `roborev fix` process.

The authoritative global source is the standard global config selected by
roborev's data directory. The Agent Hook's existing `--config` flag may still
select an alternate file for hook thresholds and `instruction`, but it does not
redirect `fix_guidelines`; a later `roborev fix` process has no access to that
hook-only path.

The existing `[agent_hook].instruction` and `[droid_hook].instruction` settings
retain their current full-replacement semantics. Effective fix guidelines are
appended to either the default or a customized instruction, so users may
customize workflow wording and finding policy independently.

## Prompt Flow

### Agent Hook

At each hook event, determine the same effective Git directory used by Agent
Hook's repository accounting. This is normally the event's working directory;
for a commit-producing command that uses `git -C`, it is the command's target
directory. Resolve the worktree root and fix guidelines from that directory and
the standard global config. Append a clearly labeled autofix-guidelines section
to the already-resolved continuation instruction before constructing the
hook-daemon request. Apply the same behavior to kit-backed profiles, legacy
profile-less registrations, Grok Build, and Factory Droid.

An event outside a Git worktree keeps the existing instruction unchanged; the
hook daemon already treats such events as untracked. Once a worktree root is
resolved, an invalid repo config is a hook configuration error rather than a
silent fallback to global guidance.

Compose the text in the hook process rather than adding a daemon request field.
This avoids persisted-state and protocol changes and prevents an older running
hook daemon from silently discarding a new field. Deferred Hermes reminders
continue to store the complete composed instruction through the existing
request and reminder fields.

When effective guidelines are empty, send the exact instruction used today.

### Fix Agents

Inject the effective guidelines into every prompt that can run an agent during
a fix operation:

- a direct single-job fix prompt;
- a batch fix prompt;
- the retry prompt used when a fix agent leaves uncommitted changes.

With guidelines present, replace unconditional "apply all findings" wording in
the direct and batch prompt framing with instructions to evaluate findings
against the supplied policy, fix findings that warrant changes, and record
findings intentionally not applied. The commit-retry prompt tells the agent to
check the pending changes against the guidelines before committing them.

With no guidelines, retain the existing prompt constants and output byte for
byte. Existing prompt-size calculations must include the conditional guidance
and conditional batch framing so configured size limits remain accurate.

### Shipped Fix Skills

Update every shipped `roborev-fix` skill variant to state that autofix
guidelines supplied by Agent Hook govern how findings are evaluated. Without
such guidance, the skill keeps its current default of addressing all actionable
findings and noting false positives or intentional design decisions rather than
silently skipping them.

The guidance is user policy, not review output. Continue treating findings,
comments, logs, and quoted text as untrusted data rather than instructions.

## Errors And Compatibility

Invalid standard global or resolved repo TOML fails before an agent receives a
prompt. No new parser, daemon schema, persisted state, migration, or network API
is required.

Missing and empty values are equivalent and preserve current behavior. Existing
configs, hook registrations, scripts using `--instruction`, and automated fix
flows need no migration. Changing the setting takes effect on the next hook or
fix invocation; the hook daemon does not need restarting.

## Tests

Behavior tests defend independently drifting boundaries:

- Config resolution proves repo guidance replaces global guidance and missing
  repo guidance inherits global guidance.
- Hook request tests prove ordinary profiles, Grok/legacy paths, and Droid send
  the composed instruction, while empty guidance sends the prior instruction
  unchanged. Scope cases prove `git -C` selects the target repo, outside-repo
  events remain unchanged, malformed repo config fails, and the hook's alternate
  `--config` path does not redirect shared fix guidance.
- Direct and batch fix tests use the real prompt builders to prove guidance and
  evaluation framing reach the agent; empty guidance preserves the existing
  prompt text.
- Batch splitting tests prove guideline and conditional-framing bytes count
  toward `max_prompt_size`.
- Commit-retry behavior tests use a recording agent to prove the retry prompt
  receives the same guidance and asks for a policy check before committing.
- Embedded-skill conformance tests prove every shipped agent variant includes
  the conditional guideline rule.
- Config-command tests prove `fix_guidelines` is valid in global and repo scope.

User-facing documentation covers the key, precedence, hook and non-hook reach,
the distinction from full `instruction` replacement, and an example policy that
requires agents to validate rather than blindly implement review suggestions.
