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
          talc = pkgs.buildGoModule {
            pname = "talc";
            version = "0.2.0";
            src = ./tools/talc;
            vendorHash = "sha256-P3iFBhlDRS+bTfGRwy2bTPmi83HgIOMPKI364SRUouI=";
          };

          default = self.packages.${system}.talc;
        };

        apps = {
          talc = {
            type = "app";
            program = "${self.packages.${system}.talc}/bin/talc";
          };

          default = self.apps.${system}.talc;
        };
      });
}
