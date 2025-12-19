self: super:
let
  lib = super.lib;

  # TODO: adjust this path to your local clone.
  xuezhSrcPath = /Users/josh/code/xuezh;

  src = builtins.path {
    name = "xuezh-src";
    path = xuezhSrcPath;
    filter = path: type:
      let
        abs = toString path;
        rel =
          if abs == toString xuezhSrcPath then ""
          else lib.removePrefix (toString xuezhSrcPath + "/") abs;
        parts = lib.filter (p: p != "") (lib.splitString "/" rel);
        hasPart = p: lib.elem p parts;
      in
        !(
          hasPart ".git"
          || hasPart ".devenv"
          || hasPart ".direnv"
          || hasPart ".pytest_cache"
          || hasPart ".mypy_cache"
          || hasPart ".ruff_cache"
          || hasPart ".DS_Store"
          || rel == "devenv.lock"
        );
  };
in
{
  xuezh = super.python3Packages.buildPythonApplication {
    pname = "xuezh";
    version = "0.1.0";
    format = "pyproject";
    inherit src;

    nativeBuildInputs = [
      super.makeWrapper
    ];

    propagatedBuildInputs = with super.python3Packages; [
      typer
      pydantic
      pydantic-settings
      ulid-py
      azure-cognitiveservices-speech
      openai-whisper
    ];

    postInstall = ''
      wrapProgram $out/bin/xuezh \
        --prefix PATH : ${lib.makeBinPath [
          super.ffmpeg
          super.edge-tts
          super.python3Packages.openai-whisper
        ]}
    '';

    meta = with lib; {
      description = "Local Chinese learning engine (xuezh)";
      homepage = "https://github.com/joshp123/xuezh";
      license = licenses.mit;
      platforms = platforms.unix;
      mainProgram = "xuezh";
    };
  };
}
