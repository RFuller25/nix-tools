# garden

A garden that grows in real time. Sow a seed, name the plant, water it, pull
the weeds, and come back tomorrow to something taller.

```sh
nix run .#garden      # or: go run ./tools/garden
```

## How it grows

* **Five drawn stages** per plant — seed, sprout, seedling, budding, mature —
  each one hand-drawn for that species and coloured from its own palette.
* **A day to adulthood.** A watered, weeded plant reaches its mature stage
  within a day of real time, faster for radishes than for magnolias.
* **Time passes while the program is closed.** On startup the garden replays
  the hours you were away, weather and all.
* **Nothing dies.** Dry soil and weeds slow a plant down and make it sulk; they
  never kill it. Come back after a fortnight and your garden is overgrown, not
  gone.
* **Weather and seasons** are derived from the calendar and your garden's seed,
  so they are the same every time the missing hours are replayed. Rain waters
  the beds for you; a plant out of season takes its time.
* **Seeds** are the currency: gather ripe pods from mature plants, earn one for
  clearing a properly overgrown bed, and collect a few from the shed on your
  first visit each day. Rarer species unlock as more of your plants reach
  maturity.

## Keys

| key | action |
| --- | --- |
| `←↑↓→` / `hjkl` | move between beds |
| `p` / `enter` | sow a seed in the selected bed |
| `w` / `W` | water this bed / every bed |
| `c` / `C` | clear weeds here / everywhere |
| `f` / `F` | gather ripe seed here / everywhere |
| `n` | name the plant in this bed |
| `u` | lift a plant and turn the soil |
| `i` / `space` | open the plant's info card |
| `s` | seed shed |
| `a` | almanac — every species, every stage |
| `tab` | cycle garden → shed → almanac → journal |
| `m` | calming music on or off |
| `?` | help |
| `q` | quit (the garden saves itself) |

## Wind

Gusts blow through the garden at random, more often in a storm than in fog.
Each one starts off to the left and crosses the beds, so plants lean one after
another rather than all together — and they bend, with the tip of a plant
moving furthest and the base staying rooted. A tall, mature plant catches more
of the wind than a seedling does.

## Music

`m` plays a slow ambient piece: a low drone with single notes from a D major
pentatonic scale drifting over it, generated as samples at run time rather than
loaded from a file. Each garden's seed gives it its own drift. The setting is
saved, so a garden left humming is humming when you return.

Playback pipes raw PCM to the first of these found on `PATH`: `pw-play`,
`paplay`, `aplay`, `ffplay`, or sox's `play`. With none installed the garden
says so and stays quiet. `GARDEN_AUDIO=off` disables it entirely.

To hear the piece without a sound card:

```sh
RENDER_DIR=/tmp go test ./tools/garden -run TestRenderCalmMusic
```

## The almanac

85 real species, each with its Latin binomial, family, origin, flowering time,
sun and water needs, eventual height, a gardener's note and a description —
from sweet basil to the sacred lotus, by way of the Venus flytrap and a
bonsai black pine.

## Saved state

`$XDG_DATA_HOME/garden/garden.json`, or `~/.local/share/garden/garden.json`.
Override with `--save <path>` or `GARDEN_SAVE`. Writes are atomic, so an
interrupted save cannot shred an existing garden.

```sh
garden --species   # list the whole catalogue and exit
garden --version
```
