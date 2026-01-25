{ pkgs ? (
    let
      inherit (builtins) fetchTree fromJSON readFile;
      inherit ((fromJSON (readFile ./flake.lock)).nodes) nixpkgs;
    in
    import (fetchTree nixpkgs.locked) {}
  )
, sdflow ? (
    let
      inherit (builtins) fromJSON readFile;
      lock = (fromJSON (readFile ./flake.lock)).nodes.sdflow.locked;
      url = "github:${lock.owner}/${lock.repo}/${lock.rev}";
    in
    (builtins.getFlake url).packages.${pkgs.system}.default
  )
}:

let
  # provides "echo-shortcuts"
  nix_shortcuts = import (pkgs.fetchurl {
    url = "https://raw.githubusercontent.com/whacked/setup/ce9fe9be8e42db9ce003772099d08395358efe8c/bash/nix_shortcuts.nix.sh";
    hash = "sha256-uK+Fgwr6iWXbfi/itJGELzkWqGZsQ8HFpfc+ztGSF98=";
  }) { inherit pkgs; };

in pkgs.mkShell {
  buildInputs = [
    pkgs.go
    pkgs.gopls
    pkgs.jsonnet
    pkgs.yq-go
    pkgs.nodejs
    sdflow
  ];  # join lists with ++

  nativeBuildInputs = [
  ];

  shellHook = nix_shortcuts.shellHook + ''
    update-gomod2nix() {
      # can't figure out how to include it from the flake
      # like we do in sdflow/shell.nix
      nix run github:nix-community/gomod2nix
    }
  '' + ''
    export PATH=$PATH:$PWD/bin
    eval "$(sdflow --completions bash)"
    echo-shortcuts ${__curPos.file}
  '';  # join strings with +
}
