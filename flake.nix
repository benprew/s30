{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = {nixpkgs, ...}: let
    system = "x86_64-linux";
    pkgs = nixpkgs.legacyPackages.${system};
  in {
    devShells.${system}.default = pkgs.mkShell {
      packages = with pkgs; [
        pkg-config
        go
        gcc

        libGL
        libX11
        libxcursor
        libxinerama
        libXrandr
        libXxf86vm
        xinput
        xorg.libXi.dev

        alsa-lib

        imagemagick
        pngquant
      ];

      LD_LIBRARY_PATH = pkgs.lib.makeLibraryPath [
        pkgs.libglvnd
      ];
    };
  };
}
