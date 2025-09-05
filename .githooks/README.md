# Git Hooks for Kitsunium SDK

This directory contains Git hooks to ensure code quality and prevent common issues.

## Setup

Run the setup script to configure Git to use these hooks:

```bash
./scripts/setup-git-hooks.sh
```

## Available Hooks

### pre-commit

The pre-commit hook runs before each commit and performs the following checks:

#### 1. **t.Skip() Detection** 🚫

- **Purpose**: Prevents commits containing `t.Skip()` in test files
- **Why**: Skipping tests is a bad practice that hides problems and reduces confidence in the codebase
- **Philosophy**: Tests should either:
  - ✅ Pass - indicating the code works correctly
  - ❌ Fail - indicating a real issue that needs fixing

**Instead of skipping tests, you should:**

- Fix the package if the test logic is correct
- Adjust the test expectations without compromising its purpose
- Remove the test if it's no longer relevant

#### 2. **Auto-formatting** 🎨

- Automatically formats Go files using `gofmt` and `goimports`
- Updates BUILD.bazel files using Gazelle
- Formats YAML/JSON/Markdown files if prettier is available

#### 3. **Main Branch Protection** 🛡️

- Prevents direct commits to the main branch
- Encourages proper branching and PR workflow

### pre-push

Runs additional checks before pushing to remote repository.

### commit-msg

Validates commit message format.

## Bypass Hooks (Emergency Only)

If you absolutely need to bypass the hooks (not recommended):

```bash
git commit --no-verify
```

⚠️ **Warning**: Bypassing hooks should only be done in emergency situations. The hooks are there to maintain code quality.

## Disable Hooks

To temporarily disable hooks:

```bash
git config --unset core.hooksPath
```

To re-enable:

```bash
./scripts/setup-git-hooks.sh
```

## Philosophy

### Why We Don't Allow t.Skip()

`t.Skip()` is considered harmful because:

1. **Hidden Problems**: Skipped tests don't run, so problems they would catch remain hidden
2. **False Confidence**: A test suite with skipped tests gives a false sense of security
3. **Technical Debt**: Skipped tests tend to stay skipped forever, accumulating technical debt
4. **Unclear Intent**: It's not clear why a test is skipped - is it broken? Is the feature removed?

### Proper Test Management

- **Broken Tests**: Fix them or fix the code they're testing
- **Flaky Tests**: Make them deterministic
- **Slow Tests**: Optimize them or move to a separate test suite
- **Obsolete Tests**: Remove them cleanly
- **Platform-specific Tests**: Use build tags instead of t.Skip()

Remember: Every test should provide value. If it doesn't, it shouldn't exist.
