# nix-tools

Bubbletea TUI utilities, packaged as a Nix flake.

| tool | what it is |
| --- | --- |
| [`talc`](tools/talc) | calculator with units, alternative time systems and answer history |
| [`garden`](tools/garden) | a little garden you plant, tend and come back to |
| [`gamer`](tools/gamer) | Tetris, 2048, Snake, Hue and Minesweeper in one menu |

## Running

```sh
nix run .#garden
nix run .#gamer
nix run .#talc
```

Or from a checkout, with Go 1.26:

```sh
go run ./tools/garden
go test ./...
```

Both `garden` and `gamer` make sound by generating samples and piping raw PCM
to whatever player is on `PATH` (`pw-play`, `paplay`, `aplay`, `ffplay` or
sox's `play`). Nothing is linked in, so the builds stay pure Go; with no player
installed both run silently and say so.

New tools get their own module under `tools/`, a `use` line in `go.work`, and a
`buildGoModule` package plus app in `flake.nix`. A new package starts with
`vendorHash = pkgs.lib.fakeHash`; run `nix build .#<tool>` once and paste in the
hash nix reports.
