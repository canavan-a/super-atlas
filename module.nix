# NixOS module for the Super Atlas server, following super-chat's.
#
# The server is built by an activation script rather than a buildGoModule
# derivation: keeping a vendorHash in sync by hand on every go.sum change
# costs more than it's worth for a binary with no reproducibility
# requirement. It rebuilds from ${self}/server on every activation; goCache
# keeps that fast after the first run. The optional web client follows the
# same tradeoff via npmCache, and the server itself serves it at /.
{ self }:
{ config, lib, pkgs, ... }:
let
  cfg = config.services.superatlas;
  stateDir = "/var/lib/superatlas";
  serverBin = "${stateDir}/bin/superatlas-server";
  goCache = "/var/cache/superatlas-go";
  npmCache = "/var/cache/superatlas-npm";
  webDir = "${stateDir}/web";
in
{
  options.services.superatlas = {
    enable = lib.mkEnableOption "the Super Atlas server";

    listenAddr = lib.mkOption {
      type = lib.types.str;
      default = "127.0.0.1:8080";
      description = ''
        Address for the API (and web client) to listen on. Loopback by
        default: put a tunnel (cloudflared) or reverse proxy in front of it
        for TLS.
      '';
    };

    dataDir = lib.mkOption {
      type = lib.types.str;
      default = "${stateDir}/data";
      description = "Where the SQLite database lives.";
    };

    publicOrigin = lib.mkOption {
      type = lib.types.str;
      default = "";
      example = "https://atlas.example.com";
      description = ''
        The public https origin invite links point at. Build the app with the
        same URL (release/app.env) so those links open it directly.
      '';
    };

    androidCertSha256 = lib.mkOption {
      type = lib.types.listOf lib.types.str;
      default = [ ];
      example = [ "AB:CD:…" ];
      description = ''
        SHA-256 fingerprints of the APK signing key, served in
        /.well-known/assetlinks.json so invite links open straight in the app.
        release/gen-keystore.sh prints it.
      '';
    };

    trustProxy = lib.mkOption {
      type = lib.types.bool;
      default = true;
      description = ''
        Rate-limit by the client address a local tunnel reports: the
        rightmost X-Forwarded-For entry, believed only on loopback
        connections, so clients can't forge it. Turn off if clients connect
        to the server directly.
      '';
    };

    environmentFile = lib.mkOption {
      type = lib.types.nullOr lib.types.str;
      default = null;
      example = "/run/secrets/superatlas.env";
      description = ''
        File with the SMTP account sign-in codes are sent from:
        SUPERATLAS_SMTP_EMAIL=… and SUPERATLAS_SMTP_APP_PASSWORD=… (a Gmail
        app password). Without it nobody can sign in. Give a runtime path as
        a string, not a Nix path literal (which would copy it into the store).
      '';
    };

    fcmCredentialsFile = lib.mkOption {
      type = lib.types.nullOr lib.types.str;
      default = null;
      example = "/run/secrets/superatlas-fcm.json";
      description = ''
        Firebase service account key (Firebase console → Project settings →
        Service accounts → Generate new private key). When set, the Google
        Play build gets friend notifications through FCM. Passed via
        LoadCredential: give a runtime path as a string.
      '';
    };

    web.enable = lib.mkEnableOption ''
      building the app's web client (view only: it shows friends but never
      shares a location) and serving it at / next to the API
    '';
  };

  config = lib.mkIf cfg.enable {
    users.users.superatlas = {
      isSystemUser = true;
      group = "superatlas";
      home = stateDir;
    };
    users.groups.superatlas = { };

    systemd.tmpfiles.rules = [
      "d ${stateDir} 0750 superatlas superatlas -"
      "d ${stateDir}/bin 0755 root root -"
      "d ${cfg.dataDir} 0700 superatlas superatlas -"
      "d ${goCache} 0755 root root -"
    ] ++ lib.optionals cfg.web.enable [
      "d ${npmCache} 0755 root root -"
    ];

    system.activationScripts.superatlasBuild = {
      deps = [ ];
      text = ''
        export PATH=${pkgs.go}/bin:${pkgs.bash}/bin:${pkgs.nodejs_22}/bin:$PATH
        export HOME=${goCache}
        export GOCACHE=${goCache}/build
        export GOPATH=${goCache}/path
        # modernc sqlite is pure Go; no C toolchain needed.
        export CGO_ENABLED=0
        mkdir -p ${stateDir}/bin ${goCache}
        cd ${self}/server
        ${pkgs.go}/bin/go build -buildvcs=false -o ${serverBin}.new ./cmd/superatlas-server
        mv -f ${serverBin}.new ${serverBin}
      '' + lib.optionalString cfg.web.enable ''
        # ${self}/app is in the read-only store and npm ci writes
        # node_modules into the project, so build from a scratch copy.
        export npm_config_cache=${npmCache}
        appBuild=${stateDir}/app-build
        rm -rf "$appBuild"
        mkdir -p "$appBuild"
        cp -r ${self}/app/. "$appBuild"/
        chmod -R u+w "$appBuild"
        rm -rf "$appBuild/android"
        cd "$appBuild"
        ${pkgs.nodejs_22}/bin/npm ci
        ${pkgs.nodejs_22}/bin/npm run build:web
        rm -rf ${webDir}.new
        cp -r dist ${webDir}.new
        chown -R superatlas:superatlas ${webDir}.new
        rm -rf ${webDir}
        mv -f ${webDir}.new ${webDir}
      '' + ''
        ${pkgs.systemd}/bin/systemctl try-restart superatlas.service || true
      '';
    };

    systemd.services.superatlas = {
      description = "Super Atlas server";
      after = [ "network.target" ];
      wantedBy = [ "multi-user.target" ];
      environment = {
        SUPERATLAS_LISTEN_ADDR = cfg.listenAddr;
        SUPERATLAS_DATA_DIR = cfg.dataDir;
        SUPERATLAS_PUBLIC_ORIGIN = cfg.publicOrigin;
        SUPERATLAS_ANDROID_CERT_SHA256 = lib.concatStringsSep "," cfg.androidCertSha256;
        SUPERATLAS_TRUST_PROXY = if cfg.trustProxy then "1" else "0";
      } // lib.optionalAttrs (cfg.fcmCredentialsFile != null) {
        SUPERATLAS_FCM_CREDENTIALS_FILE = "%d/fcm.json";
      } // lib.optionalAttrs cfg.web.enable {
        SUPERATLAS_WEB_DIR = webDir;
      };
      serviceConfig = {
        ExecStart = serverBin;
        Restart = "always";
        RestartSec = 2;
        User = "superatlas";
        Group = "superatlas";
        WorkingDirectory = cfg.dataDir;
        NoNewPrivileges = true;
        PrivateTmp = true;
        ProtectSystem = "strict";
        ProtectHome = true;
        ReadWritePaths = [ cfg.dataDir ];
        ProtectKernelTunables = true;
        ProtectKernelModules = true;
        ProtectControlGroups = true;
        RestrictSUIDSGID = true;
        LockPersonality = true;
        MemoryDenyWriteExecute = true;
        RestrictAddressFamilies = [ "AF_INET" "AF_INET6" "AF_UNIX" ];
        CapabilityBoundingSet = "";
        UMask = "0077";
      } // lib.optionalAttrs (cfg.environmentFile != null) {
        EnvironmentFile = cfg.environmentFile;
      } // lib.optionalAttrs (cfg.fcmCredentialsFile != null) {
        LoadCredential = [ "fcm.json:${cfg.fcmCredentialsFile}" ];
      };
    };
  };
}
