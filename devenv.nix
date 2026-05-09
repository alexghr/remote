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
      build.exec = "go build ./cmd/remote";
      fmt.exec = "gofmt -w .";
      check.exec = ''
        test -z "$(gofmt -l .)"
        go vet ./...
        staticcheck ./...
        go test ./...
      '';

    };

    outputs = {
      remote = pkgs.buildGoModule {
        pname = "remote";
        version = "0.1.0";
        src = lib.cleanSource ./.;
        vendorHash = null;
        subPackages = [ "cmd/remote" ];
      };
    };
  }
