# penguin

A looping, colour ASCII rendition of the penguin-walks-into-the-mountains
scene: a colony huddles on the ice at dawn, one emperor penguin looks back,
turns, and waddles off toward the peaks until it is a speck. From there it never
stops: the walker becomes two symbols trading places at the foot of the
mountains, for as long as you leave it running. Only `r` starts the film over.

Everything is drawn live: sky, stars and clouds, three ranges of faceted
mountains (alpenglow, hatched shadows, inked arêtes), sastrugi ice, and
penguins that are small ray-traced 3D models shaded into glyphs, so the walker
can turn, waddle and shrink continuously with perspective.

## Scenes

Pick one with `-scene` (default `penguin`); `-list` shows them all. In the
player, `tab` / `shift+tab` flip to the next or previous scene.

| scene | what you get |
| --- | --- |
| `penguin` | the walk into the mountains, with the soundtrack |
| `torii` | a vermilion torii in still water at dusk, Fuji, cherry blossom and falling petals |
| `aurora` | a winter range under the northern lights, mirrored in a frozen lake, a cabin with smoke |
| `river` | a wide river in perspective, sun glitter, drifting leaves and a canoe |
| `lighthouse` | a storm at night: rain, lightning, spray, and a turning beam |
| `campfire` | a fire in a pine forest, embers rising, the Milky Way, a shooting star |
| `deepsea` | jellyfish drifting through light shafts, swaying kelp, fish, marine snow |

Every scene runs forever; only `r` starts one over.

```sh
go run ./tools/penguin        # or: nix run .#penguin
go run ./tools/penguin -scene aurora
go run ./tools/penguin -frame 20 -w 120 -h 40   # print one frame
```

Keys: `tab` next scene, `space` pause, `r` restart (the only way it begins again), `m` mute, `←`/`→` seek 5s, `q` quit. Wants a terminal of
at least 60x21 with truecolor (falls back to 256 colours, or plain glyphs under
`NO_COLOR`).

Sound is an 8-bit tune, a 16-bar loop, synthesised in-process and piped to whichever
player is on `PATH` (`pw-play`, `paplay`, `aplay`, `ffplay`, sox `play`); set
`PENGUIN_AUDIO=off` to disable it. The score is written as note names in `music.go`.
