{
  description = "Bubbletea TUI utilities for NixOS";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachSystem [ "x86_64-linux" "aarch64-linux" ] (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in {
        packages = {
          nixos-switch = pkgs.buildGoModule {
            pname = "nixos-switch";
            version = "0.1.0";
            src = ./tools/nixos-switch;
            vendorHash = "sha256-P3iFBhlDRS+bTfGRwy2bTPmi83HgIOMPKI364SRUouI=";
          };

          qalc = pkgs.buildGoModule {
            pname = "qalc";
            version = "0.1.0";
            src = ./tools/qalc;
            vendorHash = "sha256-P3iFBhlDRS+bTfGRwy2bTPmi83HgIOMPKI364SRUouI=";
          };

          pkg-browser = pkgs.buildGoModule {
            pname = "pkg-browser";
            version = "0.1.0";
            src = ./tools/pkg-browser;
            vendorHash = null;
          };
        };

        apps = {
          nixos-switch = {
            type = "app";
            program = "${self.packages.${system}.nixos-switch}/bin/nixos-switch";
          };
          qalc = {
            type = "app";
            program = "${self.packages.${system}.qalc}/bin/qalc";
          };
          pkg-browser = {
            type = "app";
            program = "${self.packages.${system}.pkg-browser}/bin/pkg-browser";
          };
        };
      });
}
