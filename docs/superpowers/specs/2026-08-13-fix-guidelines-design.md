# Fix Guidelines Design

## Purpose

Roborev's Agent Hook automatically brings failed reviews back into an active
coding-agent session. Its default instruction and the prompts used by
`roborev fix` currently tell agents to address all findings. Users need a
supported way to add policy that tells autofix agents how to evaluate findings,
including when a suggestion is a false positive or conflicts with project
intent.

Add one global `fix_guidelines` setting. It supplies user guidance to every
autofix agent reached from Agent Hook, including agents launched later through
`roborev fix`, while leaving the existing automatic workflow unchanged when the
setting is empty.

## Configuration

`fix_guidelines` is a multiline string accepted at the top level of the global
config:

```toml
fix_guidelines = """
Treat review findings as hypotheses. Verify each finding against the code and
project requirements before changing anything. Explain findings that are not
applied in the review comment.
"""
```

There is no repository override, merge mode, supersede flag, Agent Hook flag, or
Agent Hook-specific environment variable. This keeps one authoritative value
visible to the active hook session and to a later `roborev fix` process.

The authoritative global source is the standard global config selected by
roborev's data directory. The Agent Hook's existing `--config` flag may still
select an alternate file for hook thresholds and `instruction`, but it does not
redirect `fix_guidelines`; a later `roborev fix` process has no access to that
hook-only path.

The existing `[agent_hook].instruction` and `[droid_hook].instruction` settings
retain their current full-replacement semantics. Fix guidelines are appended to
the complete triggered reminder, so users may customize workflow wording and
finding policy independently.

## Prompt Flow

### Agent Hook

Resolve fix guidelines from the standard global config when the hook command
starts, independently of the hook-only `--config` path. Send the existing
instruction to the hook daemon unchanged. When the daemon returns a triggered
reminder, first build the profile's complete agent-facing reminder, including
roborev's existing continuation sentence, then append a clearly labeled
autofix-guidelines section. Apply the same behavior to kit-backed profiles,
legacy profile-less registrations, Grok Build, and Factory Droid.

Composing only after a trigger keeps policy outside the daemon-generated trigger
description and avoids changing output for hook events that do not prompt the
agent. Deferred Hermes reminders receive the current global policy when the
reminder is delivered. No daemon request field, persisted state, or protocol
change is required.

When effective guidelines are empty, send the exact instruction used today.

### Fix Agents

Inject the effective guidelines into every prompt that can run an agent during
the foreground `roborev fix` operation:

- a direct single-job fix prompt;
- a batch fix prompt;
- the retry prompt used when a fix agent leaves uncommitted changes.

Do not apply this setting to `roborev analyze --fix` or `roborev refine`.
Those are explicit analysis and iterative refinement workflows rather than the
review-autofix path reached from Agent Hook, and their existing prompts remain
unchanged.

With guidelines present, replace unconditional "apply all findings" wording in
the direct and batch prompt framing with instructions to evaluate findings
against the supplied policy, fix findings that warrant changes, and record
findings intentionally not applied. The commit-retry prompt tells the agent to
check the pending changes against the guidelines before committing them.

With no guidelines, retain the existing prompt constants, output, and batching
behavior byte for byte. With guidelines configured, prompt-size calculations
must include the conditional guidance, conditional batch framing, and severity
separator so configured size limits remain accurate without changing the
legacy empty-policy boundary behavior.

### Shipped Fix Skills

Update every shipped `roborev-fix` skill variant to state that supplied autofix
guidelines govern how findings are evaluated. Without such guidance, the skill
keeps its current default of addressing all actionable findings and noting false
positives or intentional design decisions rather than silently skipping them.

The guidance is user policy, not review output. Continue treating findings,
comments, logs, and quoted text as untrusted data rather than instructions.

## Errors And Compatibility

Invalid standard global TOML fails before a hook event is posted or a fix agent
receives a prompt. No new parser, daemon schema, persisted state, migration, or
network API is required.

Missing and empty values are equivalent and preserve current behavior. Existing
configs, hook registrations, scripts using `--instruction`, and automated fix
flows need no migration. Changing the setting takes effect on the next hook or
fix invocation; the hook daemon does not need restarting.

## Tests

Behavior tests defend independently drifting boundaries:

- Config tests prove the global key loads and remains invalid in repository
  scope.
- Hook output tests prove ordinary profiles, Grok/legacy paths, and Droid send
  the composed agent-facing reminder, while empty guidance sends the prior text
  unchanged. The policy section must be terminal in encoded output. Additional
  cases prove untriggered events do not append policy and the hook's alternate
  `--config` path does not redirect shared fix guidance.
- Direct and batch fix tests use the real prompt builders to prove guidance and
  evaluation framing reach the agent; empty guidance preserves the existing
  prompt text.
- Batch splitting tests derive exact fit and split boundaries from the real
  builder, proving guideline, conditional-framing, and severity-separator bytes
  count toward `max_prompt_size` without changing empty-policy grouping.
- Commit-retry behavior tests use a recording agent to prove the retry prompt
  receives the same guidance and asks for a policy check before committing.
- `analyze --fix` and `refine` behavior tests use unique policy sentinels to
  prove the separate workflows do not receive global fix guidelines.
- Embedded-skill conformance tests prove every shipped agent variant includes
  the conditional guideline rule.
- Config-command tests prove `fix_guidelines` is valid only in global scope.

User-facing documentation covers the global-only key, hook and non-hook reach,
the distinction from full `instruction` replacement, and an example policy that
requires agents to validate rather than blindly implement review suggestions.
