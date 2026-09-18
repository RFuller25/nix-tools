package main

import (
	"strings"
	"testing"
	"time"
)

func TestTaskListIsWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, task := range taskList {
		if seen[task.ID] {
			t.Errorf("duplicate task id %q", task.ID)
		}
		seen[task.ID] = true

		if strings.TrimSpace(task.Text) == "" {
			t.Errorf("%s has no wording", task.ID)
		}
		if task.Reward <= 0 {
			t.Errorf("%s pays %d seeds", task.ID, task.Reward)
		}
		if task.progress == nil {
			t.Fatalf("%s has no way of telling whether it is done", task.ID)
		}
		// It must be possible to be at the start of it.
		have, want := task.progress(newTestGarden(testStart()))
		if want <= 0 {
			t.Errorf("%s asks for %d of something", task.ID, want)
		}
		if have > want {
			t.Errorf("%s starts already overshot: %d of %d", task.ID, have, want)
		}
	}
}

func TestANewGardenHasSuggestions(t *testing.T) {
	g := newTestGarden(testStart())
	active := g.ActiveTasks()
	if len(active) != activeTasks {
		t.Fatalf("a new garden suggests %d things, want %d", len(active), activeTasks)
	}
	for _, a := range active {
		if a.Text == "" || a.Reward <= 0 {
			t.Errorf("a suggestion came out empty: %+v", a)
		}
		if a.Have >= a.Want {
			t.Errorf("%q was handed out already finished", a.Text)
		}
	}
	if g.TasksDone() != 0 {
		t.Error("a new garden has already finished something")
	}
}

func TestFinishingATaskPaysAndBringsAnother(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)

	// Force a known task to be the only live one.
	g.Tasks = []TaskState{{ID: "a-bee", Given: now}}
	seeds := g.Seeds

	g.checkTasks(now)
	if g.Seeds != seeds {
		t.Error("an unfinished task paid out")
	}

	g.Sightings = map[string]time.Time{"bee": now}
	g.checkTasks(now)

	reward := taskByID("a-bee").Reward
	if g.Seeds != seeds+reward {
		t.Errorf("seeds = %d, want %d", g.Seeds, seeds+reward)
	}
	if g.TasksDone() != 1 {
		t.Errorf("%d tasks are marked done", g.TasksDone())
	}
	if len(g.ActiveTasks()) != activeTasks {
		t.Errorf("%d suggestions are live after finishing one, want %d", len(g.ActiveTasks()), activeTasks)
	}

	// It pays once, not every tick.
	g.checkTasks(now)
	if g.Seeds != seeds+reward {
		t.Errorf("the same task paid twice: %d seeds", g.Seeds)
	}

	found := false
	for _, e := range g.Journal {
		if strings.Contains(e.Text, "Done:") {
			found = true
		}
	}
	if !found {
		t.Error("finishing a task was not written in the journal")
	}
}

func TestTasksAreNeverHandedOutAlreadyDone(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	// A gardener who has already done a great deal.
	g.Herbarium = map[string]time.Time{}
	for i, sp := range AllSpecies() {
		if i >= 20 {
			break
		}
		g.Herbarium[sp.ID] = now
	}
	g.Tasks = nil
	g.refreshTasks(now)

	for _, a := range g.ActiveTasks() {
		if a.Have >= a.Want {
			t.Errorf("%q was suggested when it was already true", a.Text)
		}
	}
}

func TestProgressIsClamped(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	g.Gathered = 500

	have, want := g.Progress("gather-twenty")
	if have != want {
		t.Errorf("progress = %d/%d, want it capped at the target", have, want)
	}
	if h, w := g.Progress("no-such-task"); h != 0 || w != 0 {
		t.Errorf("an unknown task reported %d/%d", h, w)
	}
}

func TestTaskWordingShowsProgress(t *testing.T) {
	counted := ActiveTask{Text: "Gather twenty seeds", Have: 3, Want: 20, Reward: 8}
	if got := counted.String(); !strings.Contains(got, "3/20") {
		t.Errorf("a counted task reads %q", got)
	}
	single := ActiveTask{Text: "Have a bee find the garden", Have: 0, Want: 1, Reward: 5}
	if got := single.String(); strings.Contains(got, "/") {
		t.Errorf("a one-off task reads %q", got)
	}
}

func TestTasksSurviveASave(t *testing.T) {
	now := testStart()
	path := t.TempDir() + "/garden.json"
	g := newTestGarden(now)
	before := g.ActiveTasks()

	if err := Save(path, g); err != nil {
		t.Fatal(err)
	}
	back, err := Load(path, now)
	if err != nil {
		t.Fatal(err)
	}
	after := back.ActiveTasks()
	if len(after) != len(before) {
		t.Fatalf("%d suggestions came back, want %d", len(after), len(before))
	}
	for i := range before {
		if before[i].Text != after[i].Text {
			t.Errorf("suggestion %d came back as %q, was %q", i, after[i].Text, before[i].Text)
		}
	}
}

func TestTheSameGardenGetsTheSameSuggestions(t *testing.T) {
	now := testStart()
	a, b := newTestGarden(now), newTestGarden(now)
	ta, tb := a.ActiveTasks(), b.ActiveTasks()
	if len(ta) != len(tb) {
		t.Fatal("two identical gardens got different numbers of suggestions")
	}
	for i := range ta {
		if ta[i].Text != tb[i].Text {
			t.Errorf("suggestion %d differs: %q and %q", i, ta[i].Text, tb[i].Text)
		}
	}
}

// Everything in the list has to be reachable, or it sits there forever.
func TestEveryTaskCanBeFinished(t *testing.T) {
	now := testStart()
	for _, task := range taskList {
		g := newTestGarden(now)
		g.Seeds = 100000
		g.Herbarium = map[string]time.Time{}
		g.Sightings = map[string]time.Time{}

		// A thoroughly accomplished garden.
		for _, sp := range AllSpecies() {
			g.Herbarium[sp.ID] = now
		}
		for name := range creatures {
			g.Sightings[name.kind().name] = now
		}
		g.Gathered, g.Volunteers = 100, 5
		for len(g.Plots) < 20 {
			g.Plots = append(g.Plots, Plot{})
		}
		g.layOutSoil()
		for i := range g.Plots {
			g.Plots[i].Weeds = 0
			g.Plots[i].Richness = 0.9
			g.Plots[i].SpeciesID = "basil"
		}
		g.Plots[0] = Plot{SpeciesID: "waterlily", Pond: true, PH: 6.5, Richness: 0.9}
		g.Plots[1] = Plot{SpeciesID: "tomato", PH: 6.5, Richness: 0.9}
		g.Plots[2] = Plot{SpeciesID: "marigold", PH: 6.5, Richness: 0.9}

		if have, want := task.progress(g); have < want {
			t.Errorf("%s is still %d of %d in a garden that has done everything", task.ID, have, want)
		}
	}
}
