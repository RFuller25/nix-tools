# garden

A garden that grows in real time, and that you breed. Sow a seed, tend it, cross
it with its neighbour, keep the seedlings you like best, and come back tomorrow
to something taller, bluer or quicker than anything the shop sells.

```sh
nix run .#garden      # or: go run ./tools/garden
```

Everything below is also in the game: open the **almanac** (`a`) and read the
**guide**, which is generated from the same tables the game runs on.

## What it is

* **Plants with genes.** Every plant has seven blended genes: a colour (hue,
  saturation and lightness, so any shade a terminal can show), height, shape,
  speed and yield. A seed is the average of its two parents plus a little noise,
  and the noise widens the more unlike the parents are, so crossing different
  plants gives something to choose between and breeding near-identical siblings
  steadies a line.
* **Real time.** A watered, weeded plant of a bought form reaches its mature
  stage within a day. Time passes while the program is closed: on startup the
  garden replays the hours you were away, weather and all.
* **Nothing dies.** Dry soil and weeds slow a plant and make it sulk; they never
  kill it.
* **Gold, and your own seed shed.** Gold buys seed, beds and ponds. Seed lives
  in packets in your shed: buy it, or gather it from your own plants, sow it and
  sell what you do not need.
* **Pollination.** Bees by day and moths by night cross neighbouring flowers of
  one species on their own; `x` crosses two by hand, with a brush.
* **Cultivars.** A plant that flowers unlike any named form is written down in
  the almanac as a new hybrid. Breed it true three generations running and it is
  a stable line you can name.
* **Neighbours.** The old companion-planting rules (marigolds guard tomatoes,
  mint crowds everything) plus rules from the genes: a giant shades a sun-lover,
  a bushy plant crowds a smaller one, a heavy cropper takes the water. `g` shows
  how every bed gets on with the ones around it.
* **A planner.** `P` lays ghosts of your seed over the beds and scores the
  layout as though it were grown; `V` saves a block of beds as a layout and `T`
  stamps one down: three sisters, a tomato guild, a cottage border, or your own.
* **Orders and a fair.** Three orders a day ask for a plant with a height or a
  colour; fill one and the plant is used up and you are paid. Once a week the
  fair judges one plant of yours against a field that rises as you win.
* **Weather, light and the year.** Weather and seasons come from the calendar and
  your garden's seed, so they are the same every time the missing hours are
  replayed. The garden runs on the real sun and some flowers close for the dark.
  Annuals go to seed, perennials sleep through winter.
* **Visitors.** Bees, butterflies, finches, dragonflies, moths, foxes and
  hedgehogs call, and the journal notes each first visit.

## Keys

<!-- keys:start -->

**In the garden**

| key | action |
| --- | --- |
| `←↑↓→ / hjkl` | move between beds |
| `home / end / G` | first bed / last bed |
| `p / enter` | sow in the selected bed (opens the seed shed), or open the card of what is growing |
| `i / space` | open the plant's info card |
| `w / W` | water this bed / every bed |
| `c / C` | clear weeds here / everywhere |
| `f / F` | gather ripe seed here / everywhere |
| `n / r` | name the plant in this bed |
| `u` | lift a plant and compost it into the bed |
| `b` | break new ground: one more bed |
| `d` | dig a pond here, or fill it back in |
| `g` | neighbour overlay: colour every bed by how well it gets on with the beds around it, green for good company and red for crowding |
| `x` | pollinate by hand: brush pollen from another flower of the same species onto this one, so its next seed is that cross |
| `a` | the almanac |
| `s` | the seed shed (shop and your own seeds) |
| `P` | plan mode: lay ghosts of your seed over the beds and see how the layout would get on before sowing anything |
| `V` | select a block of beds, then save what is growing in them as a layout |
| `T` | layouts: the built-in ones and your own, to stamp down anywhere |
| `o` | fill an order with the plant in this bed, if it fits one: the plant is used up and you are paid |
| `O` | the order board |
| `e` | enter the plant in this bed in this week's show |
| `E` | the fair: this week's class, your entry and your ribbons |

**Choosing a pollen donor (after x)**

| key | action |
| --- | --- |
| `←↑↓→ / hjkl` | move to the flower to take pollen from (flowers that will do are outlined in gold) |
| `enter / x / space` | take pollen from the selected flower |
| `esc` | cancel |

**In plan mode (P)**

| key | action |
| --- | --- |
| `←↑↓→ / hjkl` | move to a bed |
| `[ / ]` | choose which seed to place: yours first, then anything the shop sells |
| `{ / }` | jump ten along the seed list |
| `enter / space / p` | place the chosen seed in this bed as a ghost |
| `backspace / x / delete` | take the ghost out of this bed |
| `C` | sow the whole plan: your own seed where you have it, bought seed where you do not |
| `g` | the neighbour overlay is always on in plan mode; g is not needed |
| `esc` | leave plan mode without sowing anything |

**Placing a layout (T, then enter)**

| key | action |
| --- | --- |
| `←↑↓→ / hjkl` | move the layout over the garden |
| `enter / space` | sow the layout here: your own seed where you have it, bought seed where you do not |
| `esc` | put the layout away |

**Selecting beds to save as a layout (V)**

| key | action |
| --- | --- |
| `←↑↓→ / hjkl` | stretch the selection |
| `enter / space` | save what is growing in the selected beds as a layout, and name it |
| `esc` | cancel |

**On the layouts screen (T)**

| key | action |
| --- | --- |
| `↑↓ / jk` | browse layouts |
| `enter / p / space` | place the layout: choose where in the garden it goes, then enter again |
| `x / delete / backspace` | delete a layout of your own (the built-in ones stay) |

**On the order board**

| key | action |
| --- | --- |
| `↑↓ / jk` | browse orders |
| `enter / o / space` | deliver the first of your plants that fits the order; it is used up and you are paid |

**At the fair**

| key | action |
| --- | --- |
| `↑↓ / jk` | choose a plant |
| `enter / e / space` | enter the chosen plant in this week's show, replacing any earlier entry |

**In the shed: the shop shelf**

| key | action |
| --- | --- |
| `↑↓ / jk` | browse |
| `←→ / hl` | choose a variety |
| `pgup / pgdn` | read a long card |
| `home / g, end / G` | first / last |
| `t` | show only what is happy in this season, or the whole rack |
| `enter / p / space` | buy one seed and sow it in the selected bed |
| `b` | buy one seed into your shed without sowing it |
| `s` | switch to your own seeds |

**In the shed: my seeds shelf**

| key | action |
| --- | --- |
| `↑↓ / jk` | browse your packets |
| `pgup / pgdn` | read a long card |
| `home / g, end / G` | first / last packet |
| `enter / p / space` | sow one seed from the packet in the selected bed |
| `$ / S` | sell one seed from the packet / the whole packet |
| `r / n` | give the packet a name of your own |
| `s` | switch to the shop |

**On a plant's card**

| key | action |
| --- | --- |
| `w` | water the bed |
| `c` | clear the weeds |
| `f` | gather ripe seed |
| `n / r` | name the plant |
| `u` | lift the plant and compost it |
| `←→ / hl` | the previous / next bed |
| `↑↓ / jk, pgup / pgdn` | scroll the card |
| `home` | back to the top of the card |

**In the almanac**

| key | action |
| --- | --- |
| `↑↓ / jk` | browse the guide, species and visitors |
| `←→ / hl, space` | step through a plant's five stages |
| `v` | flick through a species' varieties |
| `n` | name a cultivar you have bred (on a cultivar's page) |
| `pgup / pgdn` | read a long page |
| `home / g, end / G` | first / last entry |

**In the journal**

| key | action |
| --- | --- |
| `↑↓ / jk, pgup / pgdn` | scroll back through the log |
| `home / g` | newest entry |

**On this screen**

| key | action |
| --- | --- |
| `↑↓ / jk, pgup / pgdn` | scroll |
| `home / g` | back to the top |
| `any other key` | close help |

**Anywhere**

| key | action |
| --- | --- |
| `ctrl+c` | quit at once (the garden saves itself) |
| `m` | calming music on or off |
| `q` | back to the garden; from the garden, quit (it saves itself) |
| `?` | this help, from anywhere; again to close it |
| `tab` | cycle garden → shed → orders → fair → almanac → journal |
| `esc` | back to the garden |

<!-- keys:end -->

## Colour

Colours are full 24-bit RGB. If your terminal advertises true colour
(`COLORTERM=truecolor`) you see exactly the shade a plant carries; otherwise
they are snapped to the nearest of 256 colours. Force a mode with
`--color truecolor|256|16|off` or `GARDEN_COLOR`, which helps under tmux or ssh
when the terminal under-reports. `NO_COLOR` turns colour off.

## Genes

| gene | what it does |
| --- | --- |
| colour | hue, saturation and lightness of the flowers (or the fruit, or the leaves on a lettuce) |
| height | dwarf to giant within the species' own range; giants grow up to 20% slower |
| shape | slim to bushy; changes the drawing and how much wind and room a plant takes |
| speed | ×0.80 to ×1.25 growth, and quick annuals finish their year sooner |
| yield | how fast pods ripen and how many a plant holds, at the price of thirstier ground |

Every roll in the garden is a hash of the garden's seed rather than a random
number, so a garden left running and one catching up on a fortnight it spent
closed grow exactly the same plants, make the same crosses and judge the same
show.

## The garden itself

Five beds wide, three rows to start, growing downwards to thirty as you break
new ground (`b`). Beds keep their positions, so a narrow terminal scrolls
across the garden rather than reflowing it: what grows next door matters.

Every bed has its own pH and its own richness. Mediterranean herbs want chalk,
bog and woodland plants want acid, and a plant in ground it dislikes grows
slowly rather than badly. Lifting a plant (`u`) composts it into its bed. Ponds
(`d`) never dry out and are the only place the water lily and the sacred lotus
will grow.

## Species

88 real species in 264 varieties, each with its Latin binomial, family, origin,
flowering time, sun and water needs, height, a gardener's note and a
description, from sweet basil to the sacred lotus by way of the Venus flytrap
and a bonsai black pine. Each variety is a real cultivar or colour form, and
has a genome of its own. `garden --species` lists them.

## If you played the first version

The first version used seeds as both money and planting stock. On first load of
an old save, once and once only:

* every old seed became one gold;
* every plant standing in a bed became a packet of its own form in your shed
  (plus one seed for each ripe pod it held);
* the beds were cleared, keeping the ground itself (beds, ponds, pH, richness)
  and every record: herbarium, sightings, journal, tasks and tallies;
* the old file was copied to `garden.json.v1.bak` beside the new one.

## Music

`m` plays a slow ambient piece: a low drone with single notes from a D major
pentatonic scale drifting over it, generated as samples at run time rather than
loaded from a file. Each garden's seed gives it its own drift. The setting is
saved.

Playback pipes raw PCM to the first of these found on `PATH`: `pw-play`,
`paplay`, `aplay`, `ffplay`, or sox's `play`. With none installed the garden says
so and stays quiet. `GARDEN_AUDIO=off` disables it entirely.

```sh
RENDER_DIR=/tmp go test ./tools/garden -run TestRenderCalmMusic
```

## Small terminals

Every screen is built to the window it is given. The garden scrolls both ways
rather than reflowing; the info card, the almanac and the help screen scroll
with `↑↓`; and the shed, almanac and layouts drop their side card when the
window is too narrow for one. Nothing is ever drawn taller or wider than the
terminal.

## Command line and files

The garden is saved to `$XDG_DATA_HOME/garden/garden.json`, or
`~/.local/share/garden/garden.json`. Override with `--save <path>` or
`GARDEN_SAVE`. Writes are atomic, so an interrupted save cannot shred an
existing garden.

```sh
garden --species      # list the whole catalogue and exit
garden --cultivars    # list the hybrid lines you have found and exit
garden --postcard     # print the garden as it stands, to share or redirect
garden --status       # one line for a prompt or a status bar
garden --keys         # the key tables, as markdown
garden --color 256    # truecolor, 256, 16 or off (also GARDEN_COLOR)
garden --version
```

`--postcard` draws the beds with no cursor and no chrome. The garden is five
beds across, which wants 79 columns; `--width` narrower than that wraps the beds
into blocks rather than cropping them. `--status` prints something like
`❀ 7/15 growing · 3 in flower · 2 thirsty · 21 gold · 5 seeds`.

## Development

```sh
go test ./tools/garden
UPDATE_README=1 go test ./tools/garden -run TestReadmeKeyTablesAreCurrent
```

The key tables above are generated from the binding table in `keys.go`, and
tests fail if a key, a flag, an environment variable, a task, a fair class or a
companion rule is missing from the almanac's guide or the README.
