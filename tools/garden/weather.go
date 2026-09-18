package main

import (
	"hash/fnv"
	"math/rand"
	"time"
)

// Season is the meteorological season, northern-hemisphere flavoured.
type Season int

const (
	Spring Season = iota
	Summer
	Autumn
	Winter
)

func (s Season) String() string {
	switch s {
	case Spring:
		return "spring"
	case Summer:
		return "summer"
	case Autumn:
		return "autumn"
	default:
		return "winter"
	}
}

func (s Season) Glyph() string {
	switch s {
	case Spring:
		return "❀"
	case Summer:
		return "☀"
	case Autumn:
		return "❦"
	default:
		return "❄"
	}
}

// SeasonOf maps a date onto its season.
func SeasonOf(t time.Time) Season {
	switch t.Month() {
	case time.March, time.April, time.May:
		return Spring
	case time.June, time.July, time.August:
		return Summer
	case time.September, time.October, time.November:
		return Autumn
	default:
		return Winter
	}
}

// WeatherKind is a day's sky.
type WeatherKind int

const (
	Sunny WeatherKind = iota
	Clear
	Cloudy
	Overcast
	Showers
	Rain
	Storm
	Fog
	Frost
	Snow
	Heatwave
)

type Weather struct {
	Kind WeatherKind
	// Rainfall is moisture added per hour, 0 for a dry day.
	Rainfall float64
	// Dryness scales how fast soil gives up its water.
	Dryness float64
	// Growth scales how eagerly plants put on size.
	Growth float64
}

func (w Weather) Name() string {
	switch w.Kind {
	case Sunny:
		return "sunny"
	case Clear:
		return "clear"
	case Cloudy:
		return "cloudy"
	case Overcast:
		return "overcast"
	case Showers:
		return "showers"
	case Rain:
		return "rain"
	case Storm:
		return "thunderstorm"
	case Fog:
		return "fog"
	case Frost:
		return "frost"
	case Snow:
		return "snow"
	default:
		return "heatwave"
	}
}

func (w Weather) Glyph() string {
	switch w.Kind {
	case Sunny, Clear:
		return "☀"
	case Cloudy:
		return "⛅"
	case Overcast:
		return "☁"
	case Showers, Rain:
		return "☂"
	case Storm:
		return "⚡"
	case Fog:
		return "≋"
	case Frost:
		return "✳"
	case Snow:
		return "❄"
	default:
		return "☼"
	}
}

// Wet reports whether the sky is watering the garden for you.
func (w Weather) Wet() bool { return w.Rainfall > 0 }

func weatherFor(kind WeatherKind) Weather {
	switch kind {
	case Sunny:
		return Weather{kind, 0, 1.30, 1.12}
	case Clear:
		return Weather{kind, 0, 1.15, 1.08}
	case Cloudy:
		return Weather{kind, 0, 0.85, 1.00}
	case Overcast:
		return Weather{kind, 0, 0.70, 0.96}
	case Showers:
		return Weather{kind, 0.10, 0.45, 1.06}
	case Rain:
		return Weather{kind, 0.18, 0.30, 1.02}
	case Storm:
		return Weather{kind, 0.28, 0.30, 0.92}
	case Fog:
		return Weather{kind, 0.03, 0.45, 0.95}
	case Frost:
		return Weather{kind, 0, 0.80, 0.88}
	case Snow:
		return Weather{kind, 0.06, 0.40, 0.85}
	default: // Heatwave
		return Weather{kind, 0, 1.90, 0.90}
	}
}

// seasonTable lists the weather kinds a season draws from, with repeats
// standing in for likelihood.
var seasonTable = map[Season][]WeatherKind{
	Spring: {Sunny, Clear, Clear, Cloudy, Cloudy, Overcast, Showers, Showers, Rain, Rain, Storm, Fog},
	Summer: {Sunny, Sunny, Sunny, Clear, Clear, Cloudy, Showers, Rain, Storm, Heatwave, Heatwave, Clear},
	Autumn: {Cloudy, Cloudy, Overcast, Overcast, Rain, Rain, Showers, Storm, Fog, Fog, Clear, Frost},
	Winter: {Overcast, Overcast, Cloudy, Frost, Frost, Snow, Snow, Rain, Fog, Clear, Sunny, Snow},
}

// WeatherFor derives a day's weather deterministically from the garden's seed
// and the calendar date. Determinism matters: the garden catches up on time
// that passed while the program was closed, and it must reach the same answer
// every run.
func WeatherFor(seed int64, day time.Time) Weather {
	y, m, d := day.Date()
	h := fnv.New64a()
	var buf [24]byte
	putInt(buf[0:8], seed)
	putInt(buf[8:16], int64(y)*10000+int64(m)*100+int64(d))
	putInt(buf[16:24], goldenRatio)
	_, _ = h.Write(buf[:])
	rng := rand.New(rand.NewSource(int64(h.Sum64())))

	season := SeasonOf(day)
	table := seasonTable[season]
	return weatherFor(table[rng.Intn(len(table))])
}

// goldenRatio is the usual 64-bit mixing constant, kept as a signed value.
const goldenRatio = int64(-0x61C8864680B583EB)

func putInt(b []byte, v int64) {
	for i := 0; i < 8; i++ {
		b[i] = byte(v >> (8 * i))
	}
}
