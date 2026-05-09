# Agent Notes

## Project

This project aims to provide a way to remotely control coding agents that are running on my development machine while
on the go. My workflow revolves around tmux, one session, multiple windows and multiple panes per window. Generally
one (or more) panes per window will be dedicated to coding agents. This project will aim to detect the correct panes
and using the tmux API will send key strokes for the agent to continue working while I'm away from my desk.

The reason this project is tied to tmux session is so that I can drive agent session from both the terminal and the
web session in the browser. Both views of the coding agent session would be in sync.

The project assumes a Unix-like environment. It is expected to run on both Linux and MacOS so when in doube we will prefer standard Unix API.

## Tech Stack

Devenv to manage the development tools. See `devenv.nix` 

## Commands

First off check if you are inside the Nix dev shell or enter using the helper `./devenv shell --no-reload`. The helper
`./devenv` script loads devenv from a pinned nixpkgs reference. Prefer it over using the global devenv.

`devenv.nix` defines helper scritps:

- Build Go: `build`
- Build production package: `./devenv build --quiet outputs.remote`
- Test, lint and check format: `check`
- Format: `fmt`
- Run locally: `run` and `dev` to watch src files and re-run the binary

If you are outside the devenv shell then you can enter it and invoke a script using
`./devenv shell --no-reload -- <command>`.

## Constraints

Your objective is to help me achieve the project's goals. I do all the coding myself, you will assist me in designing
features and researching things for the project. Do _not_ edit any files without explicit permission from myself.
Only edit files when I explicitly ask you to implement, change, or fix something.

I want to use zero or almost zero dependencies in this project so a lot of the code will have to be written by me directly. If you want to add a dependency, let me know, we can discuss it but I will lean towards saying 'no' unless it's a high quality, well maintained package.

I repeat: *do not edit* any of the files in the project without explicit permission. The point is for me to learn Go in my spare time, not to test your capabilities.

## Architecture

- `cmd/...`: entrypoints
- `internal/...`: core code

## Gotchas

The tmux test I wrote fails because it tries to open a temp tmux session. I suspect it's because tmux wants a PTY.
You can ignore tests for now and the `check` helper script as it is expected to fail.
