#!/usr/bin/env sh
set -eu

if [ "$#" -ne 3 ]; then
  echo "usage: sh scripts/package-deb.sh <version> <linux-amd64-binary> <linux-arm64-binary>" >&2
  exit 2
fi

version="$1"
amd64_binary="$2"
arm64_binary="$3"
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
root_dir=$(CDPATH= cd -- "$script_dir/.." && pwd)
output_dir="$root_dir/dist"

command -v ar >/dev/null 2>&1 || { echo "missing ar" >&2; exit 1; }
command -v tar >/dev/null 2>&1 || { echo "missing tar" >&2; exit 1; }
command -v gzip >/dev/null 2>&1 || { echo "missing gzip" >&2; exit 1; }

mkdir -p "$output_dir"

package_one() {
  arch="$1"
  binary="$2"
  work_dir="$output_dir/deb-$arch"
  data_dir="$work_dir/data"
  control_dir="$work_dir/control"
  package_file="$output_dir/yeelight-home_${version}_${arch}.deb"

  rm -rf "$work_dir" "$package_file"
  mkdir -p "$data_dir/usr/local/bin" "$control_dir"
  cp "$binary" "$data_dir/usr/local/bin/yeelight-home"
  chmod 0755 "$data_dir/usr/local/bin/yeelight-home"

  size_kb=$(du -sk "$data_dir/usr/local/bin/yeelight-home" | awk '{print $1}')
  cat > "$control_dir/control" <<EOF
Package: yeelight-home
Version: $version
Section: utils
Priority: optional
Architecture: $arch
Installed-Size: $size_kb
Maintainer: Yeelight <support@yeelight.com>
Homepage: https://github.com/Yeelight/yeelight-home
Description: Yeelight Home local Runtime CLI
 Local Runtime CLI used by Yeelight smart-home Skills.
EOF

  printf '2.0\n' > "$work_dir/debian-binary"
  (cd "$control_dir" && tar --format=ustar --owner=0 --group=0 -czf "$work_dir/control.tar.gz" .)
  (cd "$data_dir" && tar --format=ustar --owner=0 --group=0 -czf "$work_dir/data.tar.gz" .)
  (cd "$work_dir" && ar -q -S "$package_file" debian-binary control.tar.gz data.tar.gz)
  rm -rf "$work_dir"
  echo "$package_file"
}

package_one amd64 "$amd64_binary"
package_one arm64 "$arm64_binary"
