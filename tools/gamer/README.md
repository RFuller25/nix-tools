# gamer

Five small games for a terminal, in one menu.

```sh
nix run .#gamer          # or: go run ./tools/gamer
gamer --play tetris      # skip the menu
```

| game | what it is | scored by |
| --- | --- | --- |
| **Tetris** | stack the falling pieces, clear the lines | points |
| **2048** | slide the tiles, double the numbers | points |
| **Snake** | eat, grow, do not bite yourself | points |
| **Hue** | put the scrambled shades of a colour gradient back in order | fewest moves |
| **Minesweeper** | clear the field without standing on a mine | fastest time |

Menu: `↑↓` to choose, `enter` to play, `q` to quit. In a game, `esc` returns to
the menu and `r` starts a fresh round. `m` mutes and unmutes from anywhere. Each
game prints its own keys along the bottom.

## Notes on each

* **Tetris** — seven-bag randomiser, ghost piece, hold (once per piece), hard
  drop, wall kicks, and gravity that speeds up every ten lines.
* **2048** — the usual rules, including the one people get wrong: a tile that
  has just merged cannot merge again in the same move.
* **Snake** — turns are queued, so a fast double-tap around a corner does what
  you meant rather than folding the snake into its own neck. The snake may
  follow its own tail into the square it is leaving.
* **Hue** — a two-way colour gradient is cut into a 7×5 grid and shuffled, with
  the corners and a few other tiles pinned as reference points. Pick a tile up
  with `space` and press `space` on another to swap them. A tile that lands in
  its own slot settles there and drops out of play, marked `✓`, so progress
  only ever accumulates — and that can never strand a puzzle, because the tiles
  still loose always include a pair whose swap settles one of them. Solve it in
  as few swaps as you can: the finished board drops every mark and cursor and
  rolls a wave of hue across the bare gradient until you press `r` or leave.
* **Minesweeper** — 16×14 with 40 mines. The first square you open is always
  safe, along with everything touching it. `f` flags; `space` on a revealed
  number opens the rest of its neighbours once the flags add up.

## Sound

A looping chiptune theme plus a sound effect for everything that happens:
pieces locking, lines clearing, tiles merging, mines going off. All of it is
generated as samples at run time — there are no audio files in the repository
and nothing is linked into the binary.

`m` mutes and unmutes, and the setting is remembered between sessions; `--mute`
starts silent. Muting is not a volume of zero: it shuts the player down, so a
muted game holds no sound device and burns no CPU.

Playback works by piping raw PCM to the first of these it finds on `PATH`:
`pw-play`, `paplay`, `aplay`, `ffplay`, or sox's `play`. With none of them
installed the games run exactly as before, silently, and the menu says so.
`GAMER_AUDIO=off` disables sound entirely.

To hear the music and effects without a sound card:

```sh
RENDER_DIR=/tmp go test ./tools/gamer -run TestRenderDemos
```

## Scores

Best score and games played per game, in
`$XDG_DATA_HOME/gamer/scores.json` (or `~/.local/share/gamer/scores.json`).
Override with `--scores <path>` or `GAMER_SCORES`. Writes are atomic. Hue and
Minesweeper are scored the other way round: fewer moves and fewer seconds win.
