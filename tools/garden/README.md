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
| `b` | break new ground: one more bed |
| `d` | dig a pond here, or fill it back in |
| `m` | calming music on or off |
| `↑↓` | scroll the info card and the help screen |
| `?` | help |
| `q` | quit (the garden saves itself) |

## The garden itself

Five beds wide, three rows to start, growing downwards to thirty as you break
new ground (`b`). Beds keep their positions — a narrow terminal scrolls across
the garden rather than reflowing it — because what grows next door matters.

Every bed has its own pH and its own richness, derived from the garden's seed.
Mediterranean herbs want chalk, bog and woodland plants want acid, and a plant
in ground it dislikes grows slowly rather than badly. Lifting a plant (`u`)
composts it into the bed it came from. Bigleaf hydrangea reads its bed and
flowers blue in acid soil and pink in lime, which is the one plant here doing
its own chemistry.

Ponds (`d`) never dry out, never weed over, and are the only place the water
lily and the sacred lotus will grow.

## Light

The garden runs on the real sun. Dawn comes up rose, dusk goes amber, night
settles blue and dim, and day length follows the season — a January evening is
dark by five. Crocus, tulip, water lily, lotus, morning glory and chamomile
fold shut for the dark. Moonflower, evening primrose and night-scented stock do
the opposite: shut all day, open at dusk, and scent the garden for the moths.

## Neighbours

What you plant alongside matters, using the relationships gardeners have
actually used. Marigolds guard the nightshades against nematodes, basil sits
beside tomatoes, alliums keep aphids off roses, chives muddle the carrot fly,
legumes feed the ground around them and corn gives beans a frame to climb —
while mint crowds out whatever it is next to, sunflowers sour the ground for
beans, and a birch drinks its neighbours dry. The info card lists what the beds
alongside are doing and why.

## The year

Every species leads the life it really leads. Annuals and biennials flower, set
seed and finish: a spent plant is not dead, it stands there bleached to straw
with a last handful of seed until you lift it. Perennials die back over winter,
deciduous trees stand bare, evergreens carry on, and all of them wake in spring.

Self-seeders — poppies, cosmos, foxgloves, chamomile and a dozen more — drop
volunteers into bare ground beside them, free. That decision is hashed from the
garden's seed rather than rolled at random, so a garden left running and one
catching up on a fortnight it spent closed grow exactly the same plants.

## Visitors

Bees work the flowers on a dry day, butterflies follow the nectar, finches drop
in on seed heads, dragonflies patrol a pond, and after dark moths come to the
night-scented flowers while a fox or a hedgehog crosses the beds. Nothing
visits a garden with nothing in it. The journal notes each one's first visit.

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

## Worth doing

The journal keeps three gentle suggestions — grow three things that flower in
autumn, put a marigold beside a tomato, wait for a moth — each worth a few
seeds. Never a timer, never a failure; finish one and another appears.

## The almanac

88 real species, each with its Latin binomial, family, origin, flowering time,
sun and water needs, eventual height, a gardener's note and a description —
from sweet basil to the sacred lotus, by way of the Venus flytrap and a bonsai
black pine. Bring one into flower and it is pressed into the herbarium, marked
with a tick and the date it first flowered.

## Small terminals

Every screen is built to the window it is given. The garden scrolls both ways
rather than reflowing, so beds keep their neighbours; the info card and the
help screen scroll with `↑↓`; and the seed shed and almanac drop their side
card when the window is too narrow to hold one, leaving the list the full
width. Nothing is ever drawn taller or wider than the terminal, because a view
that overflows scrolls its own header out of reach.

## Saved state

`$XDG_DATA_HOME/garden/garden.json`, or `~/.local/share/garden/garden.json`.
Override with `--save <path>` or `GARDEN_SAVE`. Writes are atomic, so an
interrupted save cannot shred an existing garden.

```sh
garden --species    # list the whole catalogue and exit
garden --postcard   # print the garden as it stands, to share or redirect
garden --status     # one line for a prompt or a status bar
garden --version
```

`--postcard` draws the beds with no cursor and no chrome. The garden is five
beds across, which wants 79 columns; `--width` narrower than that wraps the
beds into blocks rather than cropping them. `--status` prints something like
`❀ 7/15 growing · 3 in flower · 2 thirsty · 21 seeds`.
