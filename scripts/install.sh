#!/usr/bin/env sh
set -eu

REPO="${YEELIGHT_HOME_REPO:-yeelight/yeelight-home}"
VERSION="${YEELIGHT_HOME_VERSION:-latest}"
INSTALL_DIR="${YEELIGHT_HOME_INSTALL_DIR:-/usr/local/bin}"
BIN_NAME="yeelight-home"

detect_os() {
  case "$(uname -s)" in
    Darwin) printf '%s' darwin ;;
    Linux) printf '%s' linux ;;
    *) echo "unsupported OS: $(uname -s)" >&2; exit 1 ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) printf '%s' amd64 ;;
    arm64|aarch64) printf '%s' arm64 ;;
    *) echo "unsupported architecture: $(uname -m)" >&2; exit 1 ;;
  esac
}

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

download() {
  url="$1"
  output="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$output"
    return
  fi
  if command -v wget >/dev/null 2>&1; then
    wget -q "$url" -O "$output"
    return
  fi
  echo "missing required command: curl or wget" >&2
  exit 1
}

need_cmd tar
os="$(detect_os)"
arch="$(detect_arch)"
asset="yeelight-home-${os}-${arch}.tar.gz"
base_url="https://github.com/${REPO}/releases"
if [ "$VERSION" = "latest" ]; then
  url="${base_url}/latest/download/${asset}"
else
  url="${base_url}/download/${VERSION}/${asset}"
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

archive="$tmp_dir/$asset"
download "$url" "$archive"
checksums="$tmp_dir/checksums.txt"
if [ "$VERSION" = "latest" ]; then
  checksums_url="${base_url}/latest/download/checksums.txt"
else
  checksums_url="${base_url}/download/${VERSION}/checksums.txt"
fi
download "$checksums_url" "$checksums"
if command -v sha256sum >/dev/null 2>&1; then
  expected="$(awk -v file="$asset" '$2 == file {print $1}' "$checksums")"
  actual="$(sha256sum "$archive" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
  expected="$(awk -v file="$asset" '$2 == file {print $1}' "$checksums")"
  actual="$(shasum -a 256 "$archive" | awk '{print $1}')"
else
  echo "missing required command: sha256sum or shasum" >&2
  exit 1
fi
if [ -z "$expected" ] || [ "$expected" != "$actual" ]; then
  echo "checksum verification failed for $asset" >&2
  exit 1
fi
tar -xzf "$archive" -C "$tmp_dir"

binary="$tmp_dir/$BIN_NAME"
if [ ! -x "$binary" ]; then
  echo "release archive does not contain executable $BIN_NAME" >&2
  exit 1
fi

mkdir -p "$INSTALL_DIR"
target="$INSTALL_DIR/$BIN_NAME"
if [ -w "$INSTALL_DIR" ]; then
  cp "$binary" "$target"
  chmod 755 "$target"
else
  need_cmd sudo
  sudo cp "$binary" "$target"
  sudo chmod 755 "$target"
fi

"$target" version
echo "installed $BIN_NAME to $target"
