{
  description = "konvert";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
      ...
    }:
    let
      konvert =
        pkgs:
        pkgs.buildGoModule rec {
          name = "konvert";
          version = self.shortRev or "dirty";
          src = ./.;
          # this needs to be changed any time there is a change in go.mod
          # dependencies
          vendorHash = "sha256-fDa/jX1nmGA9y/ACQYaZegRQOQa8seKcjL5N/BnNtA4=";
          nativeBuildInputs = [ ];
          CGO_ENABLED = 0;
          doCheck = false;
          ldflags = [
            "-s"
            "-w"
            "-X github.com/kumorilabs/konvert/cmd.Version=${version}"
            "-X github.com/kumorilabs/konvert/cmd.GitCommit=${version}"
          ];
        };
      flakeForSystem =
        nixpkgs: system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          kf = konvert pkgs;
        in
        {
          packages = {
            konvert = kf;
          };
          devShell = pkgs.mkShell {
            packages = with pkgs; [
              curl
              kfilt
            ];
          };
        };
    in
    flake-utils.lib.eachDefaultSystem (system: flakeForSystem nixpkgs system);
}
