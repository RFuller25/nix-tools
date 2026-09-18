package main

import (
	"fmt"
	"time"
)

// Gentle goals. Never a timer, never a failure: three suggestions sit in the
// journal, and finishing one hands over a few seeds and brings up another.

const activeTasks = 3

// task is one thing worth doing, and how to tell whether it has been done.
type task struct {
	ID     string
	Text   string
	Reward int
	// progress reports how far along the gardener is, and what would finish it.
	progress func(g *Garden) (have, want int)
}

// TaskState is the saved half: which tasks are out, and which are finished.
type TaskState struct {
	ID    string    `json:"id"`
	Given time.Time `json:"given"`
	Done  bool      `json:"done,omitempty"`
}

var taskList = []task{
	{
		ID: "five-species", Text: "Bring five different species into flower", Reward: 6,
		progress: func(g *Garden) (int, int) { return len(g.Herbarium), 5 },
	},
	{
		ID: "fifteen-species", Text: "Fill fifteen pages of the herbarium", Reward: 14,
		progress: func(g *Garden) (int, int) { return len(g.Herbarium), 15 },
	},
	{
		ID: "autumn-three", Text: "Grow three plants that flower in autumn", Reward: 8,
		progress: func(g *Garden) (int, int) {
			n := 0
			for id := range g.Herbarium {
				if sp := SpeciesByID(id); sp != nil && sp.LikesSeason(Autumn) {
					n++
				}
			}
			return n, 3
		},
	},
	{
		ID: "a-tree", Text: "Bring a tree or shrub to maturity", Reward: 10,
		progress: func(g *Garden) (int, int) {
			for id := range g.Herbarium {
				if sp := SpeciesByID(id); sp != nil && sp.Life() == Woody {
					return 1, 1
				}
			}
			return 0, 1
		},
	},
	{
		ID: "pond-life", Text: "Dig a pond and grow something in it", Reward: 10,
		progress: func(g *Garden) (int, int) {
			for i := range g.Plots {
				if g.Plots[i].Pond && !g.Plots[i].Empty() {
					return 1, 1
				}
			}
			return 0, 1
		},
	},
	{
		ID: "a-bee", Text: "Have a bee find the garden", Reward: 5,
		progress: func(g *Garden) (int, int) {
			if _, ok := g.Sightings["bee"]; ok {
				return 1, 1
			}
			return 0, 1
		},
	},
	{
		ID: "moth-night", Text: "Grow a night-flowering plant and wait for a moth", Reward: 12,
		progress: func(g *Garden) (int, int) {
			if _, ok := g.Sightings["moth"]; ok {
				return 1, 1
			}
			return 0, 1
		},
	},
	{
		ID: "good-company", Text: "Put a French marigold beside a tomato", Reward: 8,
		progress: func(g *Garden) (int, int) {
			for i := range g.Plots {
				if g.Plots[i].SpeciesID != "tomato" {
					continue
				}
				for _, n := range g.Neighbours(i) {
					if g.Plots[n].SpeciesID == "marigold" {
						return 1, 1
					}
				}
			}
			return 0, 1
		},
	},
	{
		ID: "volunteer", Text: "Let something sow itself into a spare bed", Reward: 6,
		progress: func(g *Garden) (int, int) { return g.Volunteers, 1 },
	},
	{
		ID: "weed-free", Text: "Have every bed clear of weeds at once", Reward: 5,
		progress: func(g *Garden) (int, int) {
			for i := range g.Plots {
				if g.Plots[i].Weeds > 0.1 {
					return 0, 1
				}
			}
			return 1, 1
		},
	},
	{
		ID: "every-bed", Text: "Have something growing in every bed", Reward: 15,
		progress: func(g *Garden) (int, int) {
			planted := 0
			for i := range g.Plots {
				if !g.Plots[i].Empty() {
					planted++
				}
			}
			return planted, len(g.Plots)
		},
	},
	{
		ID: "good-heart", Text: "Compost a bed into good heart", Reward: 6,
		progress: func(g *Garden) (int, int) {
			for i := range g.Plots {
				if g.Plots[i].Richness >= 0.8 {
					return 1, 1
				}
			}
			return 0, 1
		},
	},
	{
		ID: "gather-twenty", Text: "Gather twenty seeds from your own plants", Reward: 8,
		progress: func(g *Garden) (int, int) { return g.Gathered, 20 },
	},
	{
		ID: "wide-garden", Text: "Break new ground: twenty beds", Reward: 10,
		progress: func(g *Garden) (int, int) { return len(g.Plots), 20 },
	},
}

func taskByID(id string) *task {
	for i := range taskList {
		if taskList[i].ID == id {
			return &taskList[i]
		}
	}
	return nil
}

// Progress is how far along an active task is.
func (g *Garden) Progress(id string) (have, want int) {
	t := taskByID(id)
	if t == nil {
		return 0, 0
	}
	have, want = t.progress(g)
	if have > want {
		have = want
	}
	return have, want
}

// refreshTasks hands out new suggestions up to the usual three, choosing from
// whatever has not been done, in a way that is the same every time for a given
// garden.
func (g *Garden) refreshTasks(now time.Time) {
	done := map[string]bool{}
	active := map[string]bool{}
	live := 0
	for _, t := range g.Tasks {
		if t.Done {
			done[t.ID] = true
			continue
		}
		active[t.ID] = true
		live++
	}

	for live < activeTasks {
		var pool []task
		for _, t := range taskList {
			if done[t.ID] || active[t.ID] {
				continue
			}
			if have, want := t.progress(g); have >= want {
				continue // already true; it would be no fun to hand out
			}
			pool = append(pool, t)
		}
		if len(pool) == 0 {
			return
		}
		pick := pool[int(hashUnit(g.Seed, int64(len(g.Tasks)), 0x7A5C)*float64(len(pool)))%len(pool)]
		g.Tasks = append(g.Tasks, TaskState{ID: pick.ID, Given: now})
		active[pick.ID] = true
		live++
	}
}

// checkTasks settles anything that has been finished since last time.
func (g *Garden) checkTasks(now time.Time) {
	for i := range g.Tasks {
		if g.Tasks[i].Done {
			continue
		}
		t := taskByID(g.Tasks[i].ID)
		if t == nil {
			g.Tasks[i].Done = true // a task that no longer exists
			continue
		}
		if have, want := t.progress(g); have < want {
			continue
		}
		g.Tasks[i].Done = true
		g.Seeds += t.Reward
		g.Log(now, "Done: %s. %d seeds.", t.Text, t.Reward)
	}
	g.refreshTasks(now)
}

// ActiveTasks lists what is currently suggested, with progress.
type ActiveTask struct {
	Text       string
	Have, Want int
	Reward     int
}

func (g *Garden) ActiveTasks() []ActiveTask {
	var out []ActiveTask
	for _, ts := range g.Tasks {
		if ts.Done {
			continue
		}
		t := taskByID(ts.ID)
		if t == nil {
			continue
		}
		have, want := g.Progress(ts.ID)
		out = append(out, ActiveTask{Text: t.Text, Have: have, Want: want, Reward: t.Reward})
	}
	return out
}

// TasksDone is how many suggestions have been seen through.
func (g *Garden) TasksDone() int {
	n := 0
	for _, t := range g.Tasks {
		if t.Done {
			n++
		}
	}
	return n
}

func (a ActiveTask) String() string {
	if a.Want <= 1 {
		return a.Text
	}
	return fmt.Sprintf("%s (%d/%d)", a.Text, a.Have, a.Want)
}
