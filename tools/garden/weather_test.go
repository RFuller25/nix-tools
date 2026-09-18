package main

import (
	"testing"
	"time"
)

// Weather has to be a pure function of garden seed and date: the catch-up
// simulation re-derives past days every time the program starts.
func TestWeatherIsDeterministic(t *testing.T) {
	day := time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC)
	first := WeatherFor(7, day)
	for i := 0; i < 50; i++ {
		if got := WeatherFor(7, day); got != first {
			t.Fatalf("weather changed between calls: %v then %v", first, got)
		}
	}
	// Any time within the same calendar day gives the same sky.
	if got := WeatherFor(7, day.Add(23*time.Hour)); got != first {
		t.Errorf("weather differs within the same day: %v vs %v", got, first)
	}
	// Different gardens see different weather.
	same := 0
	for seed := int64(0); seed < 40; seed++ {
		if WeatherFor(seed, day) == first {
			same++
		}
	}
	if same == 40 {
		t.Error("every seed produced identical weather")
	}
}

func TestWeatherVariesOverTime(t *testing.T) {
	day := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	seen := map[WeatherKind]int{}
	for i := 0; i < 120; i++ {
		seen[WeatherFor(3, day.AddDate(0, 0, i)).Kind]++
	}
	if len(seen) < 4 {
		t.Errorf("only %d kinds of weather in 120 days", len(seen))
	}
}

func TestSeasonBoundaries(t *testing.T) {
	cases := map[time.Month]Season{
		time.January: Winter, time.February: Winter, time.March: Spring,
		time.May: Spring, time.June: Summer, time.August: Summer,
		time.September: Autumn, time.November: Autumn, time.December: Winter,
	}
	for month, want := range cases {
		got := SeasonOf(time.Date(2026, month, 10, 0, 0, 0, 0, time.UTC))
		if got != want {
			t.Errorf("%v fell in %s, want %s", month, got, want)
		}
	}
}

// Winter must never stop a garden dead, and no season may make growth free.
func TestWeatherGrowthStaysInRange(t *testing.T) {
	for kind := Sunny; kind <= Heatwave; kind++ {
		w := weatherFor(kind)
		if w.Growth < 0.8 || w.Growth > 1.2 {
			t.Errorf("%s has growth multiplier %.2f, want 0.8-1.2", w.Name(), w.Growth)
		}
		if w.Dryness < 0 {
			t.Errorf("%s has negative dryness", w.Name())
		}
		if w.Name() == "" || w.Glyph() == "" {
			t.Errorf("weather kind %d is missing a name or glyph", kind)
		}
	}
}

func TestSnowOnlyInWinter(t *testing.T) {
	day := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 60; i++ { // all of July and August
		w := WeatherFor(11, day.AddDate(0, 0, i))
		if w.Kind == Snow || w.Kind == Frost {
			t.Errorf("%s in the middle of summer", w.Name())
		}
	}
}
