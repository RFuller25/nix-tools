# pkg-browser: NixOS Config Integration

**Date:** 2026-05-05
**Scope:** Enhance pkg-browser with NixOS config package visibility and in-TUI editing. Does not include merging with nixos-switch (planned follow-on).

---

## Overview

pkg-browser gains a split-panel layout showing user profile packages (left) and NixOS config packages (right) side by side. Users can add packages to their config from search results and remove simple packages from the config panel. After edits, a status message reminds the user to rebuild. Config path is stored on first run.

---

## Layout & Navigation

```
pkg-browser
────────────────────────┬────────────────────────
 Profile                │ Config
 (nix profile list)     │ (/etc/nixos/...)
                        │
 > vim 9.1              │ > wget
   git 2.47             │   discord
   htop 3.3             │   obs-studio
                        │   [python3.withPackages…]
────────────────────────┴────────────────────────
 / search  tab: switch panel  d: remove  q: quit
```

- `tab` moves focus between Profile (left) and Config (right) panels. Focused panel shows `>` cursor in active color.
- `/` opens a search panel at the bottom, shrinking both panels equally.
- In search results: `a` appends the selected package to the config file.
- In Config panel: `d` removes the selected package from the config file. Disabled on complex/read-only entries.
- `esc` closes the search panel; focus returns to the Config panel.
- `q` / `ctrl+c` quits.

---

## Config Parsing & Editing

The parser reads the config file and finds the `environment.systemPackages = with pkgs; [` block.

**Entry classification:**
- **Simple**: bare identifier on its own line (e.g. `wget`, `vim`) — editable, shown in TUI
- **Complex**: contains `(` or spans multiple lines (e.g. `python3.withPackages (...)`) — shown read-only as `[python3.withPackages…]`, `d` key disabled

**Add** (from search, press `a`): appends `    pkgname` on a new line before the closing `]` of the block, writes file atomically.

**Remove** (in Config panel, press `d`): deletes that exact line from the file, writes file.

**File write rules:**
- Comments on the same line as a package are preserved on removal
- Blank lines and standalone comment lines inside the block are not shown in TUI but are preserved in the file
- If write fails (e.g. permission denied), error shown in status line, file left unchanged
- No backup file written

**Post-edit status line:** `Saved. Run nixos-switch to apply.` — shown briefly, then returns to normal help text.

---

## First-Run & Config Persistence

On startup, tool checks `~/.config/pkg-browser/config.json`. If the file is missing or the `configPath` key is absent, a setup screen runs before the main TUI:

```
pkg-browser — first run setup

NixOS config path:
> /etc/nixos/configuration.nix

enter: confirm  ctrl+c: quit
```

- Pre-filled with `/etc/nixos/configuration.nix`
- On confirm: validates file exists and contains `environment.systemPackages`; shows error and re-prompts if not
- On success: writes `{"configPath": "/path/to/config.nix"}` and launches main TUI

`--config <path>` flag skips the first-run screen and uses the provided path for that session (does not overwrite saved config).

---

## Code Structure

```
tools/pkg-browser/
├── main.go       # entry point, first-run gate, tea.NewProgram
├── model.go      # main model struct, Init/Update/View, keybindings
├── layout.go     # split-panel rendering, search panel, viewport math
├── nix.go        # loadInstalled(), searchNixpkgs() — unchanged
├── config.go     # parseConfigPackages(), addPackage(), removePackage() — pure functions
├── setup.go      # first-run tea.Model; exits and returns configPath string to main
├── persist.go    # read/write ~/.config/pkg-browser/config.json
└── nix_test.go   # existing tests + new tests for config.go
```

`config.go` contains only pure functions operating on `[]byte` — no file I/O. File I/O lives in tea.Cmd wrappers in `model.go`. This keeps config logic fully unit-testable.

The setup model and main model are independent: setup runs, returns `configPath`, main model initializes with it.

---

## Error Handling

All errors shown inline — no crashes, no modals:

| Error | Display location |
|-------|-----------------|
| Profile load fail | Profile panel body |
| Config parse fail | Config panel body |
| Write fail | Status line (red) |
| Search fail | Search panel body |

Errors clear on next keypress.

---

## Testing

Unit tests in `nix_test.go` cover `config.go` pure functions:

- Parse: simple packages, complex packages, comments, blank lines
- Add: to empty block, to populated block
- Remove: existing package, nonexistent package (no-op), complex entry (no-op)
- Round-trip: parse → remove → re-parse produces expected list

Existing `nix_test.go` tests remain unchanged. No TUI interaction tests (manual verification per project convention).

---

## Out of Scope

- Merging with nixos-switch (follow-on spec)
- Inline rebuild / password prompt (deferred to merge)
- Editing complex config entries (python3.withPackages etc.)
- Supporting flake-based NixOS configs
- Multiple config files / imports
