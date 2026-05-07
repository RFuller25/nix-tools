package main

import (
	"testing"
)

var sampleConfig = []byte(`
{ config, pkgs, ... }:
{
  environment.systemPackages = with pkgs; [
    vim
    wget
    # a comment
    (python3.withPackages (python-pkgs: with python-pkgs; [
      flask
    ]))
    git
  ];
}
`)

func TestParseConfigPackages_SimpleEntries(t *testing.T) {
	pkgs := parseConfigPackages(sampleConfig)
	names := make(map[string]bool)
	for _, p := range pkgs {
		names[p.Name] = true
	}
	for _, want := range []string{"vim", "wget", "git"} {
		if !names[want] {
			t.Errorf("want %q in result, got %v", want, pkgs)
		}
	}
}

func TestParseConfigPackages_ComplexReadOnly(t *testing.T) {
	pkgs := parseConfigPackages(sampleConfig)
	for _, p := range pkgs {
		if p.Name == "python3" {
			if !p.ReadOnly {
				t.Errorf("python3 entry should be ReadOnly")
			}
			return
		}
	}
	t.Error("python3 complex entry not found")
}

func TestParseConfigPackages_CommentNotIncluded(t *testing.T) {
	pkgs := parseConfigPackages(sampleConfig)
	for _, p := range pkgs {
		if p.Name == "a comment" || p.Name == "#" {
			t.Errorf("comment line should not appear as package: %q", p.Name)
		}
	}
}

func TestAddPackage_AppendsBeforeClosingBracket(t *testing.T) {
	src := []byte(`  environment.systemPackages = with pkgs; [
    vim
  ];`)
	result := addPackage(src, "git")
	pkgs := parseConfigPackages(result)
	names := make(map[string]bool)
	for _, p := range pkgs {
		names[p.Name] = true
	}
	if !names["git"] {
		t.Errorf("git not found after add; packages: %v", pkgs)
	}
	if !names["vim"] {
		t.Errorf("vim disappeared after add; packages: %v", pkgs)
	}
}

func TestAddPackage_EmptyBlock(t *testing.T) {
	src := []byte(`  environment.systemPackages = with pkgs; [
  ];`)
	result := addPackage(src, "htop")
	pkgs := parseConfigPackages(result)
	if len(pkgs) != 1 || pkgs[0].Name != "htop" {
		t.Errorf("want [htop], got %v", pkgs)
	}
}

func TestRemovePackage_RemovesEntry(t *testing.T) {
	src := []byte(`  environment.systemPackages = with pkgs; [
    vim
    wget
    git
  ];`)
	result := removePackage(src, "wget")
	pkgs := parseConfigPackages(result)
	for _, p := range pkgs {
		if p.Name == "wget" {
			t.Error("wget still present after remove")
		}
	}
}

func TestRemovePackage_NoopOnMissing(t *testing.T) {
	src := []byte(`  environment.systemPackages = with pkgs; [
    vim
  ];`)
	result := removePackage(src, "nonexistent")
	if string(result) != string(src) {
		t.Error("src should be unchanged when package not found")
	}
}

func TestRemovePackage_NoopOnComplex(t *testing.T) {
	result := removePackage(sampleConfig, "python3")
	pkgs := parseConfigPackages(result)
	found := false
	for _, p := range pkgs {
		if p.Name == "python3" {
			found = true
		}
	}
	if !found {
		t.Error("complex entry should not be removable")
	}
}

func TestRoundTrip_RemoveThenParse(t *testing.T) {
	src := []byte(`  environment.systemPackages = with pkgs; [
    vim
    wget
    git
  ];`)
	after := removePackage(src, "wget")
	pkgs := parseConfigPackages(after)
	if len(pkgs) != 2 {
		t.Errorf("want 2 packages after remove, got %d: %v", len(pkgs), pkgs)
	}
}

func TestParseConfigPackages_CommentWithBracket(t *testing.T) {
	src := []byte(`  environment.systemPackages = with pkgs; [
    vim # [no bracket issue]
    git
  ];`)
	pkgs := parseConfigPackages(src)
	names := make(map[string]bool)
	for _, p := range pkgs {
		names[p.Name] = true
	}
	if !names["vim"] {
		t.Error("vim not found")
	}
	if !names["git"] {
		t.Error("git not found — bracket in comment corrupted depth counter")
	}
}

func TestAddPackage_NoDuplicate(t *testing.T) {
	src := []byte(`  environment.systemPackages = with pkgs; [
    vim
  ];`)
	result := addPackage(src, "vim")
	pkgs := parseConfigPackages(result)
	count := 0
	for _, p := range pkgs {
		if p.Name == "vim" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 vim, got %d", count)
	}
}
