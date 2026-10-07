package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func sourceFiles(t *testing.T) map[string]string {
	t.Helper()
	out := map[string]string{}
	files, _ := filepath.Glob("*.go")
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		out[f] = string(b)
	}
	return out
}

// Every flag the program defines is documented in the almanac's guide and the README.
func TestEveryFlagIsDocumented(t *testing.T) {
	re := regexp.MustCompile(`flag\.(?:String|Bool|Int)\("([a-z-]+)"`)
	m := re.FindAllStringSubmatch(sourceFiles(t)["main.go"], -1)
	if len(m) < 6 {
		t.Fatalf("found only %d flags in main.go", len(m))
	}
	docs := ""
	for _, d := range cliDocs() {
		docs += d.Name + "\n"
	}
	readme, _ := os.ReadFile("README.md")
	for _, f := range m {
		if !strings.Contains(docs, "--"+f[1]) {
			t.Errorf("flag --%s is missing from the guide's command-line chapter", f[1])
		}
		if !strings.Contains(string(readme), "--"+f[1]) {
			t.Errorf("flag --%s is missing from the README", f[1])
		}
	}
}

// Every environment variable the program reads is documented.
func TestEveryEnvironmentVariableIsDocumented(t *testing.T) {
	re := regexp.MustCompile(`os\.Getenv\("([A-Z_]+)"\)`)
	have := map[string]bool{}
	for _, src := range sourceFiles(t) {
		for _, m := range re.FindAllStringSubmatch(src, -1) {
			have[m[1]] = true
		}
	}
	// Constants naming variables.
	have["GARDEN_COLOR"], have["NO_COLOR"] = true, true
	docs := ""
	for _, d := range envDocs() {
		docs += d.Name + " " + d.Desc + "\n"
	}
	readme, _ := os.ReadFile("README.md")
	for name := range have {
		if name == "XDG_DATA_HOME" || name == "HOME" {
			continue // described in the saved-state paragraph
		}
		if !strings.Contains(docs, name) {
			t.Errorf("environment variable %s is not in the guide", name)
		}
		if !strings.Contains(string(readme), name) {
			t.Errorf("environment variable %s is not in the README", name)
		}
	}
}

// The README's counts are the catalogue's counts.
func TestReadmeCountsMatchTheCatalogue(t *testing.T) {
	readme, _ := os.ReadFile("README.md")
	forms := 0
	for _, sp := range AllSpecies() {
		forms += len(sp.Varieties())
	}
	want := regexp.MustCompile(`(\d+) real species in (\d+) varieties`).FindStringSubmatch(string(readme))
	if want == nil {
		t.Fatal("the README should say \"N real species in M varieties\"")
	}
	if want[1] != itoa(len(AllSpecies())) || want[2] != itoa(forms) {
		t.Errorf("README says %s species in %s varieties; the catalogue has %d in %d", want[1], want[2], len(AllSpecies()), forms)
	}
}

func itoa(n int) string { return strconv.Itoa(n) }

// Every journal-worthy thing the guide names exists: each task's text is in the guide.
func TestEveryTaskIsListedInTheGuide(t *testing.T) {
	m := demoModel(t, 100, 40)
	var text string
	for _, ch := range guideChapters() {
		if ch.Title == "Worth doing" {
			text = strings.Join(ch.Build(m, 300), "\n")
		}
	}
	for _, task := range taskList {
		if !strings.Contains(text, task.Text) {
			t.Errorf("task %q is not in the guide", task.ID)
		}
	}
}

// Every fair class and every companion rule is described in the guide.
func TestGuideNamesEveryClassAndRule(t *testing.T) {
	m := demoModel(t, 100, 40)
	all := map[string]string{}
	for _, ch := range guideChapters() {
		all[ch.Title] = strings.Join(ch.Build(m, 400), "\n")
	}
	for _, c := range fairCategories {
		if !strings.Contains(all["The fair"], c.Name) {
			t.Errorf("fair class %q is not in the guide", c.Name)
		}
	}
	for _, c := range companions {
		if !strings.Contains(all["Neighbours"], c.note) {
			t.Errorf("companion rule %q is not in the guide", c.note)
		}
	}
	for _, tpl := range builtinTemplates() {
		if !strings.Contains(all["Layout planner and layouts"], tpl.Name) {
			t.Errorf("built-in layout %q is not in the guide", tpl.Name)
		}
	}
}

// Every chapter title is unique and the guide leads the almanac.
func TestAlmanacOpensWithTheGuide(t *testing.T) {
	m := demoModel(t, 100, 40)
	vis := m.almanacVisible()
	if len(vis) < 2 || vis[0].Kind != rowHeader || vis[1].Kind != rowGuide {
		t.Fatal("the almanac should open on the guide")
	}
	if cur, _ := m.almanacCur(); cur.Kind != rowGuide {
		t.Errorf("the cursor starts on a %v row, want the first chapter", cur.Kind)
	}
	seen := map[string]bool{}
	for _, ch := range guideChapters() {
		if seen[ch.Title] {
			t.Errorf("two chapters are called %q", ch.Title)
		}
		seen[ch.Title] = true
	}
}
