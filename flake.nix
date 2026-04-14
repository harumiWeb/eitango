{
  description = "eitango - English vocabulary learning CLI";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages = {
          eitango = pkgs.buildGoModule {
            pname = "eitango";
            version = self.shortRev or self.dirtyShortRev or "dev";

            src = self;

            vendorHash = "sha256-fVqXrhz2wMWvWiPL8X/M0Oqz6LIRSLocKNZmEVhQLtY=";

            subPackages = [ "cmd/eitango" ];

            env.CGO_ENABLED = 0;

            ldflags = [
              "-s"
              "-w"
              "-X main.version=${self.shortRev or self.dirtyShortRev or "dev"}"
              "-X main.commit=${self.rev or "dirty"}"
              "-X main.date=1970-01-01T00:00:00Z"
            ];

            meta = {
              description = "English vocabulary learning CLI with spaced repetition";
              homepage = "https://github.com/harumiWeb/eitango";
              mainProgram = "eitango";
            };
          };
          default = self.packages.${system}.eitango;
        };

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gopls
            golangci-lint
          ];
        };
      }
    );
}
