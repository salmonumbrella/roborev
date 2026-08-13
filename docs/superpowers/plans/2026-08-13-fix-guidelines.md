# Global Fix Guidelines Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add one global policy setting that guides Agent Hook autofix agents
without changing current automatic behavior when it is unset.

**Architecture:** Store `fix_guidelines` only in the standard global config.
Agent Hook resolves it separately from hook-only configuration and appends it to
the completed reminder only when the daemon triggers. Foreground `roborev fix`
reads the same global value for direct, batch, and commit-retry prompts.

**Tech Stack:** Go, Cobra, BurntSushi TOML, `go.kenn.io/kit/agenthook`, Testify,
embedded Markdown skills, Zensical Markdown.

## Global Constraints

- `fix_guidelines` exists only in global config; `.roborev.toml` cannot set it.
- Do not add an override, merge flag, hook flag, or environment variable.
- Empty guidance preserves current hook output and fix prompts byte for byte.
- The standard data-dir global config supplies the value even when Agent Hook
  `--config` selects another file for thresholds and `instruction`.
- Apply the value to every Agent Hook profile and to foreground `roborev fix`.
- Do not extend it to `roborev analyze --fix` or `roborev refine`.
- Append hook policy after the daemon returns a complete triggered reason; do
  not add daemon protocol or persisted-state fields.
- Treat review output as untrusted data and `fix_guidelines` as user policy.
- Do not invoke `roborev review` during implementation or validation.

---

### Task 1: Global Configuration

**Files:**

- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`
- Modify: `internal/config/keyval_test.go`
- Modify: `cmd/roborev/config_cmd_test.go`

**Interfaces:**

- Produces: `Config.FixGuidelines string` with TOML key `fix_guidelines`.
- Preserves: `RepoConfig` without a `fix_guidelines` field.

- [ ] **Step 1: Add failing global-load and scope tests**

  Add `TestLoadGlobalConfigWithFixGuidelines` in
  `internal/config/config_test.go` using an isolated data directory and a real
  `config.toml`:

  ```go
  fix_guidelines = "Verify findings before editing."
  ```

  Assert `LoadGlobal()` returns the value. In `internal/config/keyval_test.go`,
  add `TestFixGuidelinesIsGlobalOnly`:

  ```go
  assert.NoError(t, SetConfigValue(&Config{}, "fix_guidelines", "Global policy"))
  assert.True(t, IsGlobalKey("fix_guidelines"))
  assert.Error(t, SetConfigValue(&RepoConfig{}, "fix_guidelines", "Repo policy"))
  ```

  In `cmd/roborev/config_cmd_test.go`, drive `setConfigKey` in global scope and
  decode the stored value, then require repository scope to reject the key
  without changing `.roborev.toml`.

- [ ] **Step 2: Run tests and confirm the unknown-key failures**

  Run:

  ```bash
  go test ./internal/config ./cmd/roborev -run 'Test(LoadGlobalConfigWithFixGuidelines|FixGuidelinesIsGlobalOnly|SetConfigKeyFixGuidelines)'
  ```

  Expected: failures because the global field does not exist.

- [ ] **Step 3: Add the global config field**

  Add beside `ReviewGuidelines`:

  ```go
  FixGuidelines string `toml:"fix_guidelines" comment:"Policy for evaluating review findings during automated fixes."`
  ```

  Do not modify `RepoConfig`; reflection-based key validation then makes global
  scope valid and repository scope invalid.

- [ ] **Step 4: Format and verify the config slice**

  Run:

  ```bash
  gofmt -w internal/config/config.go internal/config/config_test.go internal/config/keyval_test.go cmd/roborev/config_cmd_test.go
  go test ./internal/config ./cmd/roborev -run 'Test(LoadGlobalConfigWithFixGuidelines|FixGuidelinesIsGlobalOnly|SetConfigKeyFixGuidelines)'
  go test ./internal/config
  ```

  Expected: all commands pass without warnings.

- [ ] **Step 5: Commit the configuration slice**

  Stage the files named in this task, inspect the staged diff, then run:

  ```bash
  git commit -m "feat(config): add global fix guidelines"
  ```

---

### Task 2: Agent Hook Reminder Policy

**Files:**

- Create: `internal/autofix/guidelines.go`
- Modify: `internal/agenthook/config.go`
- Modify: `internal/agenthook/config_test.go`
- Modify: `internal/agenthook/output.go`
- Modify: `internal/agenthook/output_test.go`
- Modify: `cmd/roborev/agent_hook_handler.go`
- Modify: `cmd/roborev/agent_hook_cmd.go`
- Modify: `cmd/roborev/agent_hook_test.go`

**Interfaces:**

- Produces: `autofix.GuidelinesHeading` and
  `autofix.AppendGuidelines(text, guidelines string) string` in a
  dependency-light package shared by hook output, fix prompts, and skill
  conformance tests.
- Extends: `agenthook.Options` with `FixGuidelines string`; it has no CLI flag or
  environment variable.
- Produces: policy-aware output helpers in `internal/agenthook/output.go` that
  append guidelines after the existing continuation sentence.

- [ ] **Step 1: Add failing option-source tests**

  In `internal/agenthook/config_test.go`, isolate `ROBOREV_DATA_DIR` and write:

  - standard `config.toml` with `fix_guidelines = "Global policy"`;
  - alternate hook config with `[agent_hook] instruction = "Alternate flow"`.

  Resolve both an ordinary profile and Droid with `--config` marked changed.
  Assert profile-specific `Instruction` comes from the alternate file and
  `FixGuidelines` comes from the standard global file in both cases. Add
  empty-policy and malformed-standard-global cases. The malformed case must
  fail before any hook request is sent.

- [ ] **Step 2: Add failing agent-facing output tests**

  In `cmd/roborev/agent_hook_test.go`, use the real run paths and a recording
  `postAgentHook` response to prove:

  - kit-backed Stop output appends a labeled policy after the complete trigger
    reason;
  - PostToolUse additional context does the same;
  - legacy profile-less and Grok output include the policy;
  - Factory Droid, as a kit-backed profile, follows the same handler behavior;
  - an untriggered response never appends policy;
  - empty policy produces the exact prior JSON output.

  For legacy and Grok, decode stdout JSON, extract `reason` for Stop or
  `hookSpecificOutput.additionalContext` for PostToolUse, then assert
  `strings.HasSuffix(strings.TrimSpace(text), guidelines)`. Make the same suffix
  assertion on kit-returned Stop and PostToolUse fields. Do not assert against
  raw encoded JSON, where the policy is escaped inside a string. This prevents
  daemon trigger details or roborev's continuation sentence from landing inside
  the policy section.

  In `internal/agenthook/config_test.go`, update `clearAgentHookEnv` to set
  `ROBOREV_DATA_DIR` to an empty temporary directory, then add calls to that
  helper in the existing tests which currently bypass it:
  `TestResolveOptionsEnvOverridesGlobalConfig`,
  `TestResolveOptionsFlagsOverrideEnv`, and
  `TestResolveOptionsForAgentGrokUsesSelfContainedInstruction`. In
  `cmd/roborev/agent_hook_test.go`, set `ROBOREV_DATA_DIR` in
  `TestAgentHookRunSupportsLegacyProfilelessRegistration` and every new Cobra
  run-path test. This prevents a user's installed policy from changing exact
  output assertions.

- [ ] **Step 3: Run focused tests and confirm policy is absent**

  Run:

  ```bash
  go test ./internal/agenthook ./cmd/roborev -run 'Test(ResolveOptions.*FixGuidelines|AgentHook.*FixGuidelines|RunAgentHook.*FixGuidelines)'
  ```

  Expected: compile or behavior failures because options and output composition
  do not yet carry the policy.

- [ ] **Step 4: Implement global option loading and conditional formatting**

  Add the formatter in the dependency-light `internal/autofix` package so the
  hook path does not import the storage-backed prompt package:

  ```go
  const GuidelinesHeading = "## Autofix Guidelines"

  func AppendGuidelines(text, guidelines string) string {
      guidelines = strings.TrimSpace(guidelines)
      if guidelines == "" {
          return text
      }
      return text + "\n\n" + GuidelinesHeading + "\n\n" + guidelines
  }
  ```

  Refactor `ResolveOptionsForAgent` to capture the result of either
  `resolveAgentOptions` or `resolveDroidOptions`, then perform one shared
  standard-global load and assign `Options.FixGuidelines` after the profile
  resolver succeeds. Reuse the already decoded config when `opts.ConfigPath`
  is the standard path; otherwise load the standard global separately. Never
  read guidelines from an alternate `opts.ConfigPath`. This shared
  post-dispatch step must also cover legacy profile-less and Grok resolution.
  An alternate hook file may contain other top-level global keys, but those
  remain outside this lookup and do not create override semantics.

- [ ] **Step 5: Compose only complete triggered output**

  Keep `StopReason`, `PostToolUseAdditionalContext`, and `BuildOutput` unchanged
  for existing callers. Add policy-aware counterparts which first call the
  current helper and only then append policy:

  ```go
  func StopReasonWithFixGuidelines(reason, guidelines string) string {
      return autofix.AppendGuidelines(StopReason(reason), guidelines)
  }
  ```

  Add the parallel PostToolUse helper and
  `BuildOutputWithFixGuidelines(input, resp, guidelines)`. The build-output
  helper must still return `{}` for untriggered responses. Use these helpers in
  kit Stop/PostToolUse and legacy/Grok encoding. Keep the daemon request
  instruction unchanged so deferred reminders and trigger counters retain
  current behavior.

- [ ] **Step 6: Format and verify Agent Hook**

  Run:

  ```bash
  gofmt -w internal/autofix/guidelines.go internal/agenthook/config.go internal/agenthook/config_test.go internal/agenthook/output.go internal/agenthook/output_test.go cmd/roborev/agent_hook_handler.go cmd/roborev/agent_hook_cmd.go cmd/roborev/agent_hook_test.go
  go test ./internal/agenthook ./cmd/roborev -run 'Test(ResolveOptions|AgentHook|RunAgentHook)'
  go test ./internal/agenthook ./cmd/roborev
  ```

  Expected: all commands pass and empty-policy output remains unchanged.

- [ ] **Step 7: Commit the Agent Hook slice**

  Stage the files named in this task, inspect the staged diff, then run:

  ```bash
  git commit -m "feat(agent-hook): append global fix guidelines"
  ```

---

### Task 3: Foreground Fix Prompts

**Files:**

- Modify: `cmd/roborev/fix.go`
- Modify: `cmd/roborev/fix_test.go`
- Test: `cmd/roborev/analyze_test.go`
- Test: `cmd/roborev/refine_test.go`

**Interfaces:**

- Consumes: `Config.FixGuidelines` and `autofix.AppendGuidelines`.
- Extends: `fixJobParams` with `FixGuidelines string` for commit retries.
- Extends: `batchSplitOptions` with `FixGuidelines string` for exact size
  accounting.
- Changes `buildGenericFixPromptWithMetadata`,
  `buildBatchFixPromptWithMetadata`, `buildGenericCommitPromptWithMetadata`,
  `buildGenericCommitPrompt`, `batchPromptOverhead`, and
  `buildBatchPromptFooter` to accept guidelines.
- Adds: `buildBatchPromptHeader(fixGuidelines string) string`.
- Changes the test convenience wrappers `buildGenericFixPrompt`,
  `buildBatchFixPrompt`, and `buildGenericCommitPrompt` to forward guidelines.

- [ ] **Step 1: Confirm the complete caller set**

  Run:

  ```bash
  rg -n 'buildGenericFixPromptWithMetadata|buildBatchFixPromptWithMetadata|buildGenericCommitPromptWithMetadata|buildGenericCommitPrompt\(|batchPromptOverhead|buildBatchPromptFooter|fixJobParams\{' cmd/roborev --glob '*.go'
  ```

  Confirm production calls are confined to `cmd/roborev/fix.go`, while
  `cmd/roborev/analyze.go` uses its separate `buildFixPromptWithMetadata` path.
  This keeps signature changes and the feature boundary explicit.

- [ ] **Step 2: Add failing direct and batch prompt tests**

  Call both metadata-bearing production builders through policy cases and
  require the exact policy, evaluation wording, and a request to record
  intentionally skipped findings. Require the policy path to omit unconditional
  “apply/address all” wording. Keep baseline empty-policy cases that require the
  existing wording and no guidelines heading. The convenience wrappers have
  these signatures:

  ```go
  buildGenericFixPrompt(output, minSeverity, responses, fixGuidelines)
  buildBatchFixPrompt(entries, minSeverity, fixGuidelines)
  ```

  Add direct assertions against `buildGenericFixPromptWithMetadata` and
  `buildBatchFixPromptWithMetadata` so production call sites cannot drift from
  the wrappers.

- [ ] **Step 3: Add failing retry and size-boundary tests**

  Extend the recording-agent commit-retry test with
  `fixJobParams.FixGuidelines`. Assert the retry receives the policy from
  `buildGenericCommitPromptWithMetadata(metadata, fixGuidelines)` and checks
  pending changes against it before committing.

  Render the real two-entry prompt with non-empty `MinSeverity`, metadata, and
  policy. Set `MaxSize` first to its exact byte length and then one byte less;
  assert the exact boundary fits and the smaller boundary splits. Also use
  three entries to prove the split retains a multi-entry batch. This makes a
  one-byte drift in framing, policy, or the severity separator observable.
  Keep an empty-policy boundary assertion pinned to the legacy grouping so this
  feature does not silently repair the pre-existing severity-newline undercount
  and change default batching.

- [ ] **Step 4: Add failing command-flow and scope tests**

  First enumerate every existing test that reaches `fixSingleJob` or
  `processFixBatch` and isolate its `ROBOREV_DATA_DIR`; those paths now load a
  global field and must never read the developer's installed config. For new
  command-flow cases, point `ROBOREV_DATA_DIR` at a temporary directory and
  write its `config.toml`. Use existing fake-daemon and recording-agent
  boundaries to prove both direct and batch flows pass the loaded global policy
  to the initial agent prompt.

  In separate direct and batch cases, make the recording agent leave a working
  tree change on its first call, then assert the second call receives the global
  policy and the pending-change policy check. These cases exercise both
  production `fixJobParams` struct literals, so omitting `FixGuidelines` from
  either call site fails independently. Add a malformed temporary global config
  case for both direct and batch flows. Add the config error check in the same
  edit that first dereferences the loaded config. The batch case must assert
  that `processFixBatch` returns an error naming the config path without
  panicking and without invoking the recording agent.

  In `cmd/roborev/analyze_test.go`, configure global `fix_guidelines`, register
  an `agent.FakeAgent` whose `ReviewFn` records the prompt, and use the existing
  mock-daemon/wait boundary to drive `runAnalyzeAndFix`. Configure the fake
  agent in the same temporary global config and assert the recorded
  analysis-fix prompt does not contain the unique policy text. In
  `cmd/roborev/refine_test.go`, drive the address-prompt path with the same kind
  of sentinel and assert it is absent. These tests defend the explicit
  hook/`roborev fix` scope even if prompt helpers are shared later. Do not add a
  repository-resolution test because repository policy is out of scope.

- [ ] **Step 5: Run focused tests and observe failures**

  Run:

  ```bash
  go test ./cmd/roborev -run 'Test(BuildGenericFixPrompt|BuildBatchFixPrompt|SplitIntoBatches|FixJobDirect.*Guidelines|RunFix.*Guidelines)'
  ```

  Expected: compile failures for the new arguments and behavior failures for
  policy-aware framing.

- [ ] **Step 6: Implement conditional framing and routing**

  Preserve current strings exactly when guidelines are empty. With policy,
  append the shared section and replace only unconditional finding instructions
  with:

  ```text
  Evaluate each finding against the autofix guidelines. Apply changes for
  findings that warrant a fix, and record any finding intentionally not applied
  with the reason it was skipped.
  ```

  Pass `cfg.FixGuidelines` through direct and batch prompt construction. In the
  same edits, stop ignoring `config.LoadGlobal()` errors in agent-running paths;
  return an actionable error before dereferencing the config, resolving the
  agent, or invoking it. Rename
  `fixJobDirect`'s string parameter and the local batch prompt variable so they
  do not shadow the imported `prompt` package.

- [ ] **Step 7: Include policy in retry and exact batch sizing**

  Add this instruction to the non-empty commit-retry prompt before staging:

  ```text
  Check the pending changes against the autofix guidelines and revise them if
  needed before committing.
  ```

  Add `buildBatchPromptHeader(fixGuidelines string) string`; it selects only the
  conditional framing sentence. `buildBatchPromptFooter` appends the policy
  body exactly once, after all untrusted review output and existing
  instructions. Assert the trimmed prompt ends with the configured policy and
  `strings.Count(prompt, policy) == 1`. Compute configured-policy batch overhead
  from the same conditional header, footer, severity, metadata, and policy
  fragments used by the real prompt builder. Preserve the legacy empty-policy
  overhead calculation and its one-byte severity behavior so default batch
  grouping remains unchanged.

- [ ] **Step 8: Format and verify fix behavior**

  Run:

  ```bash
  gofmt -w cmd/roborev/fix.go cmd/roborev/fix_test.go cmd/roborev/analyze_test.go cmd/roborev/refine_test.go
  go test ./cmd/roborev -run 'Test(BuildGenericFixPrompt|BuildBatchFixPrompt|SplitIntoBatches|FixJobDirect|RunFix|AnalyzeFixIgnoresGlobalFixGuidelines|RefineIgnoresGlobalFixGuidelines)'
  go test ./cmd/roborev
  ```

  Expected: prompt, retry, routing, and size tests pass.

- [ ] **Step 9: Commit the fix-prompt slice**

  Stage the files named in this task, inspect the staged diff, then run:

  ```bash
  git commit -m "feat(fix): honor global fix guidelines"
  ```

---

### Task 4: Shipped Skills And Documentation

**Files:**

- Modify: `internal/skills/codex/roborev-fix/SKILL.md`
- Regenerate: `internal/skills/claude/roborev-fix/SKILL.md`
- Regenerate: `internal/skills/droid/roborev-fix/SKILL.md`
- Regenerate: `internal/skills/grok/roborev-fix/SKILL.md`
- Modify: `skills/roborev-fix.md`
- Modify: `internal/skills/skills_test.go`
- Modify: `docs/agent-hook.md`
- Modify: `docs/configuration.md`
- Modify: `docs/guides/assisted-refactoring.md`
- Modify: `docs/changelog.md`

**Interfaces:**

- Consumes: the runtime `autofix.GuidelinesHeading` section from Tasks 2 and 3.
- Preserves: Codex as the source for generated Claude, Droid, and Grok skills.

- [ ] **Step 1: Add a failing embedded-skill behavior test**

  Add `TestFixSkillsRespectAutofixGuidelines` in
  `internal/skills/skills_test.go`. Load every embedded provider variant and
  require it to reference the exported runtime heading exactly. Existing
  generated-file freshness tests independently ensure every generated variant
  matches the Codex source. This defends runtime/skill vocabulary drift rather
  than mirroring several prose literals.

- [ ] **Step 2: Run the focused skill test and observe failure**

  Run:

  ```bash
  go test ./internal/skills -run TestFixSkillsRespectAutofixGuidelines
  ```

  Expected: current provider variants lack the conditional rule.

- [ ] **Step 3: Update the source skill and regenerate variants**

  Update the skill summary and any frontmatter that unconditionally promises to
  fix all findings. Retitle “Fix all findings” to “Evaluate and fix findings”
  and add the conditional policy rule before sorting. Keep the existing
  no-policy rule that actionable findings are addressed and false positives or
  intentional decisions are reported. Apply the same user-facing wording to
  `skills/roborev-fix.md`, then run:

  ```bash
  go generate ./internal/skills
  ```

- [ ] **Step 4: Document global-only configuration**

  Add this example to the global config documentation:

  ```toml
  fix_guidelines = """
  Treat review findings as hypotheses. Verify each one against the code and
  project requirements. Explain findings that are intentionally not applied.
  """
  ```

  State that `.roborev.toml` cannot override it, it reaches every Agent Hook
  profile and foreground `roborev fix`, and it is separate from full-replacement
  `[agent_hook].instruction`. Explain that hook-only `--config` does not redirect
  it. Add one concise changelog entry.

- [ ] **Step 5: Format and verify skills and docs**

  Run:

  ```bash
  go test ./internal/skills
  nix run 'nixpkgs#uv' -- run --directory docs --frozen python scripts/format_markdown.py
  nix run 'nixpkgs#uv' -- run --directory docs --frozen python -m unittest scripts/test_format_markdown.py
  nix run 'nixpkgs#uv' -- run --directory docs --frozen python scripts/format_markdown.py --check
  ```

  Expected: generated-skill freshness, embedded policy, and Markdown checks all
  pass.

- [ ] **Step 6: Commit the skills and docs slice**

  Stage the files named in this task, inspect the staged diff, then run:

  ```bash
  git commit -m "docs: explain global fix guidelines"
  ```

---

### Task 5: End-To-End Verification

**Files:**

- Verify: all files changed by Tasks 1-4

**Interfaces:**

- Consumes: every prior task deliverable.
- Produces: a clean branch whose empty and configured policy paths pass all
  repository gates.

- [ ] **Step 1: Run cross-package tests**

  Run:

  ```bash
  go test ./internal/autofix ./internal/config ./internal/agenthook ./internal/prompt ./internal/skills ./cmd/roborev
  ```

- [ ] **Step 2: Run repository quality gates**

  Run:

  ```bash
  go test ./...
  go build ./...
  make lint-ci
  prek run --all-files
  ```

- [ ] **Step 3: Audit the complete diff**

  Confirm `fix_guidelines` appears only in global config, every triggered hook
  output and fix-agent prompt gets the policy, `analyze --fix` and `refine` stay
  unchanged, untriggered and empty-policy behavior is unchanged, configured
  batch size accounting matches real prompts, generated skills are current,
  and no unrelated surface changed.

- [ ] **Step 4: Commit any verification correction**

  If a gate required a correction, stage only that logical correction, inspect
  it, and run:

  ```bash
  git commit -m "fix: complete global fix guidelines integration"
  ```

  If no correction was required, do not create an empty commit.
