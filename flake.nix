{
  description = "Super Atlas: see your friends on a map. Dev shell + NixOS module";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-26.05";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    let
      perSystem = flake-utils.lib.eachDefaultSystem (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};

          # $PWD is resolved at runtime, so these scripts must be run from the
          # repo root (always `cd` into super-atlas/ before `nix develop`).
          repoRoot = "$PWD";
          dataDir = "${repoRoot}/.data";
          listenAddr = "127.0.0.1:8080";

          # Dev mode: sign-in codes are printed to the log instead of emailed
          # (unless SMTP is set in server/.env), and "larp" in the add-friend
          # email field toggles a set of simulated friends.
          devEnv = ''
            export SUPERATLAS_LISTEN_ADDR="${listenAddr}"
            export SUPERATLAS_DATA_DIR="${dataDir}"
            export SUPERATLAS_DEV=1
            if [ -d "${repoRoot}/app/dist" ]; then
              export SUPERATLAS_WEB_DIR="${repoRoot}/app/dist"
            fi
          '';

          db-reset = pkgs.writeShellScriptBin "db-reset" ''
            set -e
            rm -rf "${dataDir}"
            echo "removed ${dataDir} (a fresh database is created on next start)"
          '';

          # foreground, hot-reload via air (see server/.air.toml)
          run-superatlas = pkgs.writeShellScriptBin "run-superatlas" ''
            set -e
            mkdir -p "${dataDir}"
            ${devEnv}
            cd "${repoRoot}/server"
            exec ${pkgs.air}/bin/air
          '';

          dev-up = pkgs.writeShellScriptBin "dev-up" ''
            set -e
            mkdir -p "${dataDir}/logs"
            ${devEnv}
            (cd "${repoRoot}/server" && \
              ${pkgs.go}/bin/go run ./cmd/superatlas-server > "${dataDir}/logs/superatlas.log" 2>&1 &)
            echo "superatlas: http://${listenAddr} (log: ${dataDir}/logs/superatlas.log)"
            echo "run 'dev-down' to stop it"
          '';

          dev-down = pkgs.writeShellScriptBin "dev-down" ''
            pkill -f "cmd/superatlas-server" 2>/dev/null || true
            pkill -f "go-build.*superatlas" 2>/dev/null || true
            pkill -f "tmp/superatlas-server" 2>/dev/null || true
            echo "stopped superatlas (best-effort)"
          '';

          run-app-web = pkgs.writeShellScriptBin "run-app-web" ''
            set -e
            cd "${repoRoot}/app"
            exec ${pkgs.nodejs_22}/bin/npm run web
          '';

          run-app-web-build = pkgs.writeShellScriptBin "run-app-web-build" ''
            set -e
            cd "${repoRoot}/app"
            exec ${pkgs.nodejs_22}/bin/npm run build:web
          '';

          # Needs the system-wide Android SDK already on this machine's PATH;
          # nothing Android-related is added to this flake on purpose (as in
          # super-chat). An emulator reaches the dev server at 10.0.2.2:8080,
          # the build's default; for a phone, pass -PapiUrl (see app/README.md).
          run-app-android = pkgs.writeShellScriptBin "run-app-android" ''
            set -e
            cd "${repoRoot}/app"
            exec ${pkgs.nodejs_22}/bin/npm run android -- "$@"
          '';

          # Regenerates the logo data and launcher icons from art/.
          run-appart = pkgs.writeShellScriptBin "run-appart" ''
            set -e
            cd "${repoRoot}/tools"
            exec ${pkgs.go}/bin/go run ./appart "$@"
          '';

          run-tests = pkgs.writeShellScriptBin "run-tests" ''
            set -e
            (cd "${repoRoot}/server" && ${pkgs.go}/bin/go test ./...)
            (cd "${repoRoot}/app" && ${pkgs.nodejs_22}/bin/npx tsc --noEmit)
          '';

          printHelp = ''
            echo "super-atlas dev shell"
            echo "  run-superatlas    # foreground: the server with hot reload (air) on ${listenAddr}"
            echo "  dev-up / dev-down # start/stop the server headless (logs in .data/logs)"
            echo "  db-reset          # delete the local database"
            echo "  run-app-web       # the app's web target via Vite on :5173"
            echo "  run-app-web-build # builds app/dist, which the server then serves at /"
            echo "  run-app-android   # the app on a connected device/emulator (needs the system Android SDK)"
            echo "  run-appart        # regenerate logo data + launcher icons from art/"
            echo "  run-tests         # server tests and the app's typecheck"
            echo "  dev-help          # re-print this list"
          '';

          dev-help = pkgs.writeShellScriptBin "dev-help" printHelp;
        in
        {
          devShells.default = pkgs.mkShell {
            buildInputs = [
              pkgs.go
              pkgs.gopls
              pkgs.air
              pkgs.sqlite
              pkgs.nodejs_22
              pkgs.watchman
              pkgs.gh
              db-reset
              run-superatlas
              dev-up
              dev-down
              run-app-web
              run-app-web-build
              run-app-android
              run-appart
              run-tests
              dev-help
            ];
            shellHook = printHelp;
          };
        }
      );

      # No buildGoModule package on purpose, the same tradeoff as super-chat
      # and super-badger: the module builds the server from source in an
      # activation script instead of keeping a vendorHash in sync by hand.
      nixosModules.default = import ./module.nix { inherit self; };
    in
    perSystem // { inherit nixosModules; };
}
