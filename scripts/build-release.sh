#!/usr/bin/env bash
# Cross-platform release tarball builder for a Polar plugin (or polar-dock).
#
# Compiles the plugin's main binary for {darwin,linux,freebsd}/{amd64,arm64}
# and packages each into a tarball under dist/ with the OS-appropriate
# service file (launchd plist / systemd unit / rc.d script), nginx
# snippet, schema migration, env sample, and an install.sh that picks
# the right one for the current OS.
#
# Usage:
#   ./scripts/build-release.sh <plugin-repo-path> [<version>]
#
#   plugin-repo-path  Absolute path to the plugin's git checkout.
#                     Must contain go.mod, cmd/<svc>-svc/main.go,
#                     scripts/launchd/, scripts/migrate/, scripts/nginx/.
#   version           Defaults to git describe --tags (or "dev" if no tag).
#
# Examples:
#   ./scripts/build-release.sh /tmp/polar-modules/polar-wg
#   ./scripts/build-release.sh /tmp/polar-modules/polar-expense v0.2.0
#
# Output: dist/<svc>-<version>-<os>-<arch>.tar.gz  (one per target)
#
# Each tarball lays out:
#   <svc>-<version>/
#     bin/<svc>-svc
#     etc/launchd/polar.<svc>-svc.plist           (macOS)
#     etc/launchd/polar-<svc>-svc-launch.sh       (macOS)
#     etc/systemd/polar-<svc>-svc.service         (linux)
#     etc/rc.d/polar_<svc>_svc                    (freebsd)
#     etc/nginx/<svc>-svc-snippet.conf
#     migrations/<svc>-schema.sql
#     env/<svc>-svc.env.sample
#     install.sh
#     README.md

set -euo pipefail

REPO="${1:?usage: build-release.sh <plugin-repo-path> [version]}"
REPO="$(cd "$REPO" && pwd)"   # canonicalize
VERSION="${2:-$(cd "$REPO" && git describe --tags --always 2>/dev/null || echo dev)}"

# Discover the plugin name from cmd/<svc>-svc/ — there should be exactly one.
CMD_DIR=$(ls "$REPO/cmd" | head -1)
SVC="${CMD_DIR%-svc}"           # "wg-svc" -> "wg"
[ -n "$SVC" ] || { echo "could not infer plugin name from $REPO/cmd/" >&2; exit 1; }

UMBRELLA="$(cd "$(dirname "$0")/.." && pwd)"
TEMPLATES="$UMBRELLA/scripts/templates"
OUT="$REPO/dist"

# Wipe and re-create dist for a clean slate.
rm -rf "$OUT"
mkdir -p "$OUT"

echo "[build-release] plugin=$SVC version=$VERSION repo=$REPO"

# Targets — six combinations.  Each entry: GOOS GOARCH human-name.
TARGETS=(
    "darwin  arm64  macos-apple-silicon"
    "darwin  amd64  macos-intel"
    "linux   amd64  linux-amd64"
    "linux   arm64  linux-arm64"
    "freebsd amd64  freebsd-amd64"
    "freebsd arm64  freebsd-arm64"
)

for entry in "${TARGETS[@]}"; do
    # shellcheck disable=SC2086
    set -- $entry
    GOOS=$1; GOARCH=$2; LABEL=$3

    STAGE_NAME="${SVC}-${VERSION}-${LABEL}"
    STAGE="$OUT/$STAGE_NAME"
    rm -rf "$STAGE"
    mkdir -p "$STAGE"/{bin,etc/launchd,etc/systemd,etc/rc.d,etc/nginx,migrations,env}

    echo "[build-release] -> $GOOS/$GOARCH"
    (cd "$REPO" && CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" \
        go build -ldflags "-s -w -X main.version=$VERSION" \
        -o "$STAGE/bin/${SVC}-svc" "./cmd/${SVC}-svc")

    # macOS launchd assets (plist + wrapper + env sample)
    if [ -d "$REPO/scripts/launchd" ]; then
        cp "$REPO/scripts/launchd/polar.${SVC}-svc.plist"       "$STAGE/etc/launchd/" 2>/dev/null || true
        cp "$REPO/scripts/launchd/polar-${SVC}-svc-launch.sh"  "$STAGE/etc/launchd/" 2>/dev/null || true
    fi

    # Linux systemd — generate from template if a per-plugin one doesn't exist
    if [ -f "$REPO/scripts/systemd/polar-${SVC}-svc.service" ]; then
        cp "$REPO/scripts/systemd/polar-${SVC}-svc.service" "$STAGE/etc/systemd/"
    else
        sed -e "s/{{SVC}}/${SVC}/g" "$TEMPLATES/systemd-unit.service.tmpl" \
            > "$STAGE/etc/systemd/polar-${SVC}-svc.service"
    fi

    # FreeBSD rc.d — generate from template
    if [ -f "$REPO/scripts/rc.d/polar_${SVC}_svc" ]; then
        cp "$REPO/scripts/rc.d/polar_${SVC}_svc" "$STAGE/etc/rc.d/"
    else
        sed -e "s/{{SVC}}/${SVC}/g" "$TEMPLATES/rc.d-script.tmpl" \
            > "$STAGE/etc/rc.d/polar_${SVC}_svc"
        chmod +x "$STAGE/etc/rc.d/polar_${SVC}_svc"
    fi

    # Nginx snippet
    if [ -f "$REPO/scripts/nginx/${SVC}-svc-snippet.conf" ]; then
        cp "$REPO/scripts/nginx/${SVC}-svc-snippet.conf" "$STAGE/etc/nginx/"
    fi

    # Schema migration
    if [ -f "$REPO/scripts/migrate/${SVC}-schema.sql" ]; then
        cp "$REPO/scripts/migrate/${SVC}-schema.sql" "$STAGE/migrations/"
    fi

    # Env sample
    if [ -f "$REPO/scripts/launchd/${SVC}-svc.env.sample" ]; then
        cp "$REPO/scripts/launchd/${SVC}-svc.env.sample" "$STAGE/env/"
    fi

    # install.sh — generic per-OS picker
    sed -e "s/{{SVC}}/${SVC}/g; s/{{VERSION}}/${VERSION}/g" \
        "$TEMPLATES/install.sh.tmpl" > "$STAGE/install.sh"
    chmod +x "$STAGE/install.sh"

    # Stub README pointing at the umbrella deploy doc
    cat > "$STAGE/README.md" <<EOF
# polar-${SVC} ${VERSION} (${LABEL})

Cross-platform install bundle. Run \`./install.sh\` as root (or via sudo).
The installer detects the OS and copies the right service file + binary
into place. You still need:

  - Postgres reachable (default DSN points at localhost; override via env).
  - polar-dock running and the plugin_modules row minted (see umbrella
    deployment doc).
  - nginx route in front (snippet provided under etc/nginx/).

Full guide: https://github.com/networkextension/Polar/blob/main/doc/deploy.md
EOF

    # Tarball
    (cd "$OUT" && tar -czf "${STAGE_NAME}.tar.gz" "$STAGE_NAME" && rm -rf "$STAGE_NAME")
    echo "[build-release]    -> dist/${STAGE_NAME}.tar.gz"
done

echo ""
echo "[build-release] done. ${#TARGETS[@]} tarballs in $OUT/"
ls -lh "$OUT"
