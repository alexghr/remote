{ pkgs, lib, config, inputs, ... }:
let
  pkgs-unstable = import inputs.nixpkgs-unstable { system = pkgs.stdenv.system; };
in
  {
    packages = with pkgs;
      [
        git
        watch
        pkgs-unstable.codex
      ];

    languages.go = {
      enable = true;
    };

    scripts = {
      run.exec = "go run ./cmd/remote";
      dev.exec = "watch -x run";
    };
    env = {};
  }
