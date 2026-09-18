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
          garden = pkgs.buildGoModule {
            pname = "garden";
            version = "0.1.0";
            src = ./tools/garden;
            # Placeholder: run `nix build .#garden` once and paste in the hash
            # nix reports.
            vendorHash = "sha256-TUbaUoqDZoQTkcOMtoE/FlAiqkWN+x49JeGkDguh2UU=";
          };

          gamer = pkgs.buildGoModule {
            pname = "gamer";
            version = "0.1.0";
            src = ./tools/gamer;
            # Placeholder: run `nix build .#gamer` once and paste in the hash
            # nix reports.
            vendorHash = "sha256-uwBJAqN4sIepiiJf9lCDumLqfKJEowQO2tOiSWD3Fig=";
          };

          talc = pkgs.buildGoModule {
            pname = "talc";
            version = "0.2.0";
            src = ./tools/talc;
            vendorHash = "sha256-P3iFBhlDRS+bTfGRwy2bTPmi83HgIOMPKI364SRUouI=";
          };

          default = self.packages.${system}.talc;
        };

        apps = {
          garden = {
            type = "app";
            program = "${self.packages.${system}.garden}/bin/garden";
          };

          gamer = {
            type = "app";
            program = "${self.packages.${system}.gamer}/bin/gamer";
          };

          talc = {
            type = "app";
            program = "${self.packages.${system}.talc}/bin/talc";
          };

          default = self.apps.${system}.talc;
        };
      });
}
