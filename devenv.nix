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

    # Load Azure Speech creds from nix-secrets (agenix) without touching global nixos-config
    if command -v agenix >/dev/null 2>&1; then
      if [[ -f "$HOME/code/nix-secrets/xuezh-azure-speech-key.age" ]]; then
        export AZURE_SPEECH_KEY="$(cd "$HOME/code/nix-secrets" && RULES=./secrets.nix agenix -d xuezh-azure-speech-key.age)"
      fi
    fi
    export AZURE_SPEECH_REGION="westeurope"
  '';
}
