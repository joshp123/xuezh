{ pkgs, ... }:

{
  # Minimal devenv skeleton for this repo.
  #
  # Policy: if you need a tool, add it here (do not use brew/global installs).
  # Python deps are installed in a project venv (pip install -e .[dev]) inside devenv.

  packages = with pkgs; [
    git
    gh
    jq
    go
    gopls
    buf
    protobuf
    protoc-gen-go
    protoc-gen-connect-go
    nodejs_22
    pnpm

    # Azure provisioning + IaC
    azure-cli
    opentofu

    # Audio / media tools used by the engine wrappers
    ffmpeg
    yt-dlp
    python313Packages.edge-tts

    # Optional: if you decide to add local pronunciation tooling later
    # praat
  ];

  enterShell = ''
    echo "Entered devenv shell for xuezh."
    echo "Reminder: do not use brew/global installs; update devenv.nix instead."
  '';
}
