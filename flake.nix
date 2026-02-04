{
  description = "xuezh - Chinese learning CLI";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    let
      perSystem = flake-utils.lib.eachDefaultSystem (system:
        let
          pkgs = import nixpkgs { inherit system; };
        in
        {
          packages.default = pkgs.buildGoModule {
            pname = "xuezh";
            version = "0.1.0";

            src = ./.;
            subPackages = [ "cmd/xuezh-go" ];

            vendorHash = "sha256-Z5jP92AGNnieIsJamMmoWa09BYWQH9KGGiBm31hBoiY=";

            buildInputs = [ pkgs.sqlite ];

            postInstall = ''
              mv $out/bin/xuezh-go $out/bin/xuezh
            '';

            meta = with pkgs.lib; {
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
              go
              gopls
              sqlite
            ];
          };
        }
      );
    in
    perSystem // {
      openclawPlugin = system:
        let
          pkgs = import nixpkgs { inherit system; };
        in {
          name = "xuezh";
          skills = [ ./skills/xuezh ];
          packages = [ self.packages.${system}.default pkgs.edge-tts ];
          needs = {
            stateDirs = [ ".config/xuezh" ];
            requiredEnv = [
              "XUEZH_AZURE_SPEECH_KEY_FILE"
              "XUEZH_AZURE_SPEECH_REGION"
            ];
          };
        };
    };
}
