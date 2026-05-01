package main

import (
	"testing"
)

func TestParseProfileBlock(t *testing.T) {
	block := `Index:              0
Flake attribute:    legacyPackages.x86_64-linux.firefox
Original URL:       flake:nixpkgs
Locked URL:         github:NixOS/nixpkgs/abc123
Store paths:        /nix/store/abc123-firefox-124.0`

	p := parseProfileBlock(block)
	if p == nil {
		t.Fatal("got nil, want a package")
	}
	if p.Name != "firefox" {
		t.Errorf("Name = %q, want %q", p.Name, "firefox")
	}
	if p.Version != "124.0" {
		t.Errorf("Version = %q, want %q", p.Version, "124.0")
	}
}

func TestParseProfileBlockNoFlakeAttr(t *testing.T) {
	block := `Index:              0
Original URL:       flake:nixpkgs`
	p := parseProfileBlock(block)
	if p != nil {
		t.Fatalf("expected nil for block with no Flake attribute, got %+v", p)
	}
}

func TestExtractNameVersionFromStorePath(t *testing.T) {
	name, version := extractNameVersion("/nix/store/abc123-firefox-124.0")
	if name != "firefox" {
		t.Errorf("name = %q, want %q", name, "firefox")
	}
	if version != "124.0" {
		t.Errorf("version = %q, want %q", version, "124.0")
	}
}

func TestExtractNameVersionNoVersion(t *testing.T) {
	name, version := extractNameVersion("/nix/store/abc123-bash")
	if name != "bash" {
		t.Errorf("name = %q, want %q", name, "bash")
	}
	if version != "" {
		t.Errorf("version = %q, want empty", version)
	}
}

func TestParseSearchJSON(t *testing.T) {
	raw := `{
  "legacyPackages.x86_64-linux.firefox": {
    "pname": "firefox",
    "version": "124.0",
    "description": "A web browser"
  },
  "legacyPackages.x86_64-linux.firefox-esr": {
    "pname": "firefox-esr",
    "version": "115.0",
    "description": "Firefox ESR"
  }
}`
	pkgs, err := parseSearchJSON([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 2 {
		t.Fatalf("len = %d, want 2", len(pkgs))
	}
	found := false
	for _, p := range pkgs {
		if p.Name == "firefox" && p.Version == "124.0" && p.Description == "A web browser" {
			found = true
		}
	}
	if !found {
		t.Errorf("firefox not found in results: %+v", pkgs)
	}
}
