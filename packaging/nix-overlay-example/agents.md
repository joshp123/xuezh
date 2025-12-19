This folder contains an example Nix overlay for packaging the `xuezh` CLI.
It is **not** applied automatically. Another agent should copy/adapt it into
`~/code/nixos-config/overlays/` and wire it into the bot's package set.

Quick steps (do these in `~/code/nixos-config`, not here):
1) Copy `xuezh-overlay.nix` into `overlays/95-xuezh.nix`.
2) Edit `xuezhSrcPath` inside the overlay to point at your local repo
   (e.g., `/Users/josh/code/xuezh`).
3) Add `xuezh` to the bot's package list (e.g., `modules/shared/packages/ai-tooling.nix`
   or the Clawdis service package set).
4) Rebuild (darwin-rebuild switch) and confirm `xuezh` is on PATH for the bot.

Notes:
- This overlay bundles `ffmpeg`, `edge-tts`, and `openai-whisper` so `audio.process-voice`
  works out of the box (local and Azure backends). Adjust package names if your nixpkgs
  differs (some channels use `python3Packages.openai-whisper`).
- If you want a devenv-only path instead, you can add `pkgs.xuezh` to the bot's
  `devenv.nix` packages list, but the overlay route is preferred for system-wide
  availability without global installs.
