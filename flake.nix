{
  description = "xuezh - Chinese learning CLI";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
        lib = pkgs.lib;
        pythonPkgs = pkgs.python3Packages;
        ulid-py = pythonPkgs.buildPythonPackage {
          pname = "ulid-py";
          version = "1.1.0";
          format = "setuptools";

          src = pythonPkgs.fetchPypi {
            pname = "ulid-py";
            version = "1.1.0";
            hash = "sha256-3GiEvpFVjfB3wwEbn7DIfRCXy4/GU0sR8xAWGv1XOPA=";
          };

          doCheck = false;
        };
        azureWheelInfo = {
          aarch64-darwin = {
            url = "https://files.pythonhosted.org/packages/86/22/0ca2c59a573119950cad1f53531fec9872fc38810c405a4e1827f3d13a8e/azure_cognitiveservices_speech-1.38.0-py3-none-macosx_11_0_arm64.whl";
            hash = "sha256-ndCAD7xKhDjG39V0emWCUZFP4tIFop6bRhWMraxqs4E=";
          };
          x86_64-darwin = {
            url = "https://files.pythonhosted.org/packages/85/f4/4571c42cb00f8af317d5431f594b4ece1fbe59ab59f106947fea8e90cf89/azure_cognitiveservices_speech-1.38.0-py3-none-macosx_10_14_x86_64.whl";
            hash = "sha256-GNzpFasDJxH2h6uzKX3RkXa5y+pWKzIu5vpzZe9KUJE=";
          };
          x86_64-linux = {
            url = "https://files.pythonhosted.org/packages/4d/96/5436c09de3af3a9aefaa8cc00533c3a0f5d17aef5bbe017c17f0a30ad66e/azure_cognitiveservices_speech-1.38.0-py3-none-manylinux1_x86_64.whl";
            hash = "sha256-HDROim+q2wY86kUfAwHhO0TZck4SQjNwOb/2Aegeb4Y=";
          };
          aarch64-linux = {
            url = "https://files.pythonhosted.org/packages/a9/2d/ba20d05ff77ec9870cd489e6e7a474ba7fe820524bcf6fd202025e0c11cf/azure_cognitiveservices_speech-1.38.0-py3-none-manylinux2014_aarch64.whl";
            hash = "sha256-HgAlladJRx7+rDpUyACXlGVwt2wTBJdguXpLiB2dJK8=";
          };
        }.${system};
        azureSpeechSdk = pythonPkgs.buildPythonPackage {
          pname = "azure-cognitiveservices-speech";
          version = "1.38.0";
          format = "wheel";

          src = pkgs.fetchurl {
            inherit (azureWheelInfo) url hash;
          };

          doCheck = false;
        };
      in
      {
        packages.default = pythonPkgs.buildPythonApplication {
          pname = "xuezh";
          version = "0.1.0";
          format = "pyproject";
          src = ./.;

          nativeBuildInputs = [ pkgs.makeWrapper ];

          propagatedBuildInputs = with pythonPkgs; [
            typer
            pydantic
            pydantic-settings
            ulid-py
            edge-tts
            openai-whisper
            azureSpeechSdk
          ];

          buildInputs = [ pkgs.sqlite ];

          postInstall = ''
            wrapProgram $out/bin/xuezh \
              --prefix PATH : ${lib.makeBinPath [
                pkgs.ffmpeg
                pythonPkgs.edge-tts
                pythonPkgs.openai-whisper
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

        apps.default = flake-utils.lib.mkApp {
          drv = self.packages.${system}.default;
        };

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            python3
            python3Packages.pip
          ];
        };
      }
    );
}
