package main

import "strings"

func (m model) viewHelp() string {
	sections := []struct {
		title string
		keys  [][2]string
	}{
		{"tending", [][2]string{
			{"←↑↓→ / hjkl", "move between beds"},
			{"p / enter", "sow a seed in the selected bed"},
			{"w / W", "water this bed / every bed"},
			{"c / C", "clear weeds here / everywhere"},
			{"f / F", "gather ripe seed here / everywhere"},
			{"n", "name the plant in this bed"},
			{"u", "lift a plant and turn the soil"},
			{"i / space", "open the plant's info card"},
		}},
		{"places", [][2]string{
			{"s", "seed shed"},
			{"a", "almanac of every species"},
			{"tab", "cycle garden → shed → almanac → journal"},
			{"esc", "back to the garden"},
			{"? ", "this help"},
			{"q", "quit (the garden saves itself)"},
		}},
		{"how it grows", [][2]string{
			{"time", "plants grow in real time, even while closed"},
			{"a day", "well-tended seeds reach maturity within a day"},
			{"water", "dry soil slows growth — nothing ever dies here"},
			{"weeds", "creep in slowly and hold plants back"},
			{"weather", "rain waters for you; frost and snow slow things"},
			{"seeds", "gathered from mature plants and earned by weeding"},
		}},
	}

	var lines []string
	lines = append(lines, titleStyle.Render("❀ garden — help"), "")
	for _, sec := range sections {
		lines = append(lines, labelStyle.Render(strings.ToUpper(sec.title)))
		for _, k := range sec.keys {
			lines = append(lines, "  "+okStyle.Render(pad(k[0], 14))+valueStyle.Render(k[1]))
		}
		lines = append(lines, "")
	}
	lines = append(lines, subtleStyle.Render("saved to "+m.path))
	lines = append(lines, helpStyle.Render("any key returns to the garden"))
	return strings.Join(lines, "\n")
}
