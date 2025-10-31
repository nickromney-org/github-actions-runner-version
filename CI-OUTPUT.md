# CI Output Examples

This document shows what the `--ci` flag outputs look like in different scenarios.

## ✅ Scenario 1: Current Version

### Command

```bash
runner-version-check -c 2.329.0 --ci
```

### GitHub Actions Log Output

```text
2.329.0

::group::📊 Runner Version Check
Latest version: v2.329.0
Your version: v2.329.0
Status: Current
::endgroup::

::notice title=Runner Version Current::✅ Version 2.329.0 is up to date
```

### Job Summary (Markdown)

```markdown
## ✅ Runner Version Status: Current

| Metric          | Value      |
| --------------- | ---------- |
| Current Version | v2.329.0   |
| Latest Version  | v2.329.0   |
| Status          | ✅ Current |
| Releases Behind | 0          |

---
```

---

## ⚠️ Scenario 2: Behind (Warning)

### Command

```bash
runner-version-check -c 2.328.0 --ci
```

### GitHub Actions Log Output

```
2.329.0

::group::📊 Runner Version Check
Latest version: v2.329.0
Your version: v2.328.0
Status: Behind
::endgroup::

::notice title=Runner Version Behind::ℹ️  Version 2.328.0 is 1 releases behind
::notice::Latest version: v2.329.0

::group::📋 Available Updates
  • v2.329.0 (2024-10-14, 3 days ago) [Latest]
::endgroup::
```

### Job Summary (Markdown)

```markdown
## ℹ️ Runner Version Status: Behind

| Metric            | Value     |
| ----------------- | --------- |
| Current Version   | v2.328.0  |
| Latest Version    | v2.329.0  |
| Status            | ℹ️ Behind |
| Releases Behind   | 1         |
| Days Until Expiry | 27        |

### ℹ️ Update Available

A newer version (v2.329.0) is available.

### 📦 Available Updates

- [v2.329.0](https://github.com/actions/runner/releases/tag/v2.329.0) - Released Oct 14, 2024 (3 days ago)

---
```

---

## 🔶 Scenario 3: Critical (Approaching Expiry)

### Command

```bash
runner-version-check -c 2.328.0 --ci
```

_(Assuming v2.328.0 was released 25 days ago)_

### GitHub Actions Log Output

```
2.329.0

::group::📊 Runner Version Check
Latest version: v2.329.0
Your version: v2.328.0
Status: Critical
::endgroup::

::warning title=Runner Version Critical::⚠️  Version 2.328.0 expires in 5 days! (1 releases behind)
::warning::Update available: v2.329.0 (released 3 days ago)
::warning::Latest version: v2.329.0

::group::📋 Available Updates
  • v2.329.0 (2024-10-14, 3 days ago) [Latest]
::endgroup::
```

### Job Summary (Markdown)

```markdown
## 🔶 Runner Version Status: Critical

| Metric            | Value       |
| ----------------- | ----------- |
| Current Version   | v2.328.0    |
| Latest Version    | v2.329.0    |
| Status            | 🔶 Critical |
| Releases Behind   | 1           |
| Days Until Expiry | 5           |

### ⚠️ Update Soon

Version expires in **5 days**. Update to v2.329.0 or later.

### 📦 Available Updates

- [v2.329.0](https://github.com/actions/runner/releases/tag/v2.329.0) - Released Oct 14, 2024 (3 days ago)

---
```

---

## 🚨 Scenario 4: Expired

### Command

```bash
runner-version-check -c 2.327.1 --ci
```

### GitHub Actions Log Output

```
2.329.0

::group::📊 Runner Version Check
Latest version: v2.329.0
Your version: v2.327.1
Status: Expired
::endgroup::

::error title=Runner Version Expired::🚨 Version 2.327.1 EXPIRED! (2 releases behind AND 35 days overdue)
::error::Update required: v2.328.0 was released 65 days ago
::error::Latest version: v2.329.0

::group::📋 Available Updates
  • v2.329.0 (2024-10-14, 3 days ago) [Latest]
  • v2.328.0 (2024-08-13, 65 days ago) [First newer release]
::endgroup::
```

### Job Summary (Markdown)

```markdown
## 🚨 Runner Version Status: Expired

| Metric          | Value      |
| --------------- | ---------- |
| Current Version | v2.327.1   |
| Latest Version  | v2.329.0   |
| Status          | 🚨 Expired |
| Releases Behind | 2          |
| Days Overdue    | 35         |

### ⚠️ Action Required

**Update to v2.328.0 or later immediately.** GitHub will not queue jobs to runners with expired versions.

### 📦 Available Updates

- [v2.329.0](https://github.com/actions/runner/releases/tag/v2.329.0) - Released Oct 14, 2024 (3 days ago)
- [v2.328.0](https://github.com/actions/runner/releases/tag/v2.328.0) - Released Aug 13, 2024 (65 days ago)

---
```

---

## 📸 What It Looks Like in GitHub Actions

### In the Workflow Log

The workflow commands (`::group::`, `::error::`, etc.) create collapsible sections and annotations:

![GitHub Actions Log Example]

```
┌─────────────────────────────────────────┐
│ ▼ 📊 Runner Version Check               │  ← Collapsible section
│   Latest version: v2.329.0              │
│   Your version: v2.327.1                │
│   Status: Expired                       │
└─────────────────────────────────────────┘

🚨 Runner Version Expired                  ← Red error annotation
   Version 2.327.1 EXPIRED!
   (2 releases behind AND 35 days overdue)

┌─────────────────────────────────────────┐
│ ▼ 📋 Available Updates                  │  ← Collapsible section
│   • v2.329.0 (2024-10-14, 3 days ago)   │
│   • v2.328.0 (2024-08-13, 65 days ago)  │
└─────────────────────────────────────────┘
```

### In the Job Summary

At the bottom of your workflow run, you'll see a beautiful markdown summary:

![Job Summary Example]

```
┌─────────────────────────────────────────────────┐
│ Summary                                         │
├─────────────────────────────────────────────────┤
│                                                 │
│ ## 🚨 Runner Version Status: Expired           │
│                                                 │
│ | Metric | Value |                             │
│ |--------|-------|                             │
│ | Current Version | v2.327.1 |                 │
│ | Latest Version | v2.329.0 |                  │
│ | Status | 🚨 Expired |                        │
│ | Releases Behind | 2 |                        │
│ | Days Overdue | 35 |                          │
│                                                 │
│ ### ⚠️ Action Required                         │
│                                                 │
│ Update to v2.328.0 or later immediately.       │
│ GitHub will not queue jobs to runners with     │
│ expired versions.                               │
│                                                 │
│ ### 📦 Available Updates                       │
│                                                 │
│ - v2.329.0 - Released Oct 14, 2024 (3 days)   │
│ - v2.328.0 - Released Aug 13, 2024 (65 days)  │
└─────────────────────────────────────────────────┘
```

---

## 🎯 Benefits of `--ci` Flag

1. **Collapsible Sections** - Uses `::group::` for clean logs
2. **Annotations** - Errors/warnings show up prominently
3. **Job Summary** - Beautiful markdown table at the bottom
4. **Links** - Clickable links to GitHub releases
5. **Structured** - Easy to parse and understand
6. **Compatible** - Works with GitHub Actions natively

## 💡 Usage Tips

### Basic Usage

```yaml
- name: Check runner version
  run: runner-version-check -c $RUNNER_VERSION --ci
```

### With Continue on Error

```yaml
- name: Check runner version
  run: runner-version-check -c $RUNNER_VERSION --ci
  continue-on-error: true # Don't fail the workflow
```

### Fail on Expired

```yaml
- name: Check runner version
  run: |
    runner-version-check -c $RUNNER_VERSION --ci
    # Exit code is always 0, but we can parse the output
    # to decide if we want to fail
```

### Store Result for Later Steps

```yaml
- name: Check runner version
  id: check
  run: |
    OUTPUT=$(runner-version-check -c $RUNNER_VERSION --json)
    echo "status=$(echo $OUTPUT | jq -r .status)" >> $GITHUB_OUTPUT

    # Also show CI output
    runner-version-check -c $RUNNER_VERSION --ci

- name: Fail if expired
  if: steps.check.outputs.status == 'expired'
  run: exit 1
```

---

## 🔍 Comparison with Other Flags

| Flag     | Use Case            | Output Style                    |
| -------- | ------------------- | ------------------------------- |
| _none_   | Local terminal use  | Colorized text with emojis      |
| `--ci`   | GitHub Actions      | Workflow commands + Job summary |
| `--json` | Automation/scripts  | Structured JSON                 |
| `-v`     | Debug/investigation | Detailed verbose output         |

## 📝 Notes

- The first line of output is **always** the latest version number (for backwards compatibility with scripts)
- Exit code is always `0` - use JSON output if you need to fail based on status
- Job summary is only written if `$GITHUB_STEP_SUMMARY` environment variable exists
- Workflow commands only work in GitHub Actions (they're ignored in other environments)
