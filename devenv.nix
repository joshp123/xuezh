{ pkgs, ... }:

{
  # Minimal devenv skeleton for this repo.
  #
  # Policy: if you need a tool, add it here (do not use brew/global installs).
  # Python deps are installed in a project venv (pip install -e .[dev]) inside devenv.

  languages.python = {
    enable = true;
    version = "3.11";
    venv.enable = true;
  };

  packages = with pkgs; [
    git
        gh
    jq

    # Audio / media tools used by the engine wrappers
    ffmpeg
    yt-dlp

    # Optional: if you decide to add local pronunciation tooling later
    # praat
  ];

  enterShell = ''
    echo "Entered devenv shell for xuezh."
    echo "Reminder: do not use brew/global installs; update devenv.nix instead."
  '';
}
