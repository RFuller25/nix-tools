# nix-tools

Bubbletea TUI utilities, packaged as a Nix flake.

| tool | what it is |
| --- | --- |
| [`talc`](tools/talc) | calculator with units, alternative time systems and answer history |
| [`garden`](tools/garden) | a little garden you plant, tend and come back to |

## Running

```sh
nix run .#garden
nix run .#talc
```

Or from a checkout, with Go 1.26:

```sh
go run ./tools/garden
go test ./...
```

New tools get their own module under `tools/`, a `use` line in `go.work`, and a
`buildGoModule` package plus app in `flake.nix`. A new package starts with
`vendorHash = pkgs.lib.fakeHash`; run `nix build .#<tool>` once and paste in the
hash nix reports.
