# penguin

A looping, colour ASCII rendition of the penguin-walks-into-the-mountains
scene: a colony huddles on the ice at dawn, one emperor penguin looks back,
turns, and waddles off toward the peaks until it is a speck. Then it fades and
starts again.

Everything is drawn live: sky, stars and clouds, three ranges of faceted
mountains (alpenglow, hatched shadows, inked arêtes), sastrugi ice, and
penguins that are small ray-traced 3D models shaded into glyphs, so the walker
can turn, waddle and shrink continuously with perspective.

```sh
go run ./tools/penguin        # or: nix run .#penguin
go run ./tools/penguin -frame 20 -w 120 -h 40   # print one frame
```

Keys: `space` pause, `r` restart, `←`/`→` seek 5s, `q` quit. Wants a terminal of
at least 60x21 with truecolor (falls back to 256 colours, or plain glyphs under
`NO_COLOR`).
