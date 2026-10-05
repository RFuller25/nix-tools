# ledger

A betting board for friends. Everyone starts with 100 Betting Bucks (BBs) and
gets 10 more for every day that passes. Anyone can put a bet on the board;
the pool is shared out among whoever backed the winning outcome, in proportion
to what they put in. The house takes nothing.

```sh
nix run .#ledger          # or: go run ./tools/ledger
ledger -u alice           # use a different username for this run
```

First run asks for an API key and a username (the same flow as
`wordguesser-term`), checks them against the server, and only then saves them to
`~/.config/ledger/config.json`, readable by you alone.

## Keys

| screen | keys |
| --- | --- |
| board | `↑↓` move · `enter` open a bet · `n` new bet · `r` refresh · `q` quit |
| bet | `b` bet · `r` resolve (creator only, while open) · `esc` back |

A bet page shows every outcome's money, its share of the pool, what a BB on it
returns right now (`x2.50`), and a graph of those shares over time.

## Rules

* A new bet needs at least two outcomes and an opening stake of 5 BBs or more.
  Any later stake is 1 BB or more. A stake can't be taken back.
* Only the creator resolves a bet, picking the outcome that happened or VOID.
* Winners split the whole pool pro rata. If nobody backed the winning outcome,
  or the bet is voided, every stake is refunded.

## Network footprint

The point is to leave a very small trace.

* One request when it starts, then a request only when you press a key that
  needs one (open a bet, bet, resolve, create, `r`). There is no timer and no
  background refresh; stakes you place update the screen locally.
* Each request waits a random 0–1.2 s first, so key presses don't make a beat.
* Plain HTTPS `POST`s with a browser-style user agent and short, opaque paths
  and JSON keys. The legend lives in the website repo, `docs/bb-api.md`.
* The last board is cached in `~/.cache/ledger/` so something shows while the
  first request is in flight.

`LEDGER_URL`, `LEDGER_CONFIG` and `LEDGER_CACHE` override the server and file
locations (used by the tests).
