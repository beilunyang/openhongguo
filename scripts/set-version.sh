#!/bin/sh
# set-version.sh 把所有平台的版本号改成同一个版本，或者检查它们是否一致。
#
#   scripts/set-version.sh X.Y.Z           改版本号；Flutter 的 CHANGELOG 缺这一节时补上
#   scripts/set-version.sh --check X.Y.Z   只检查，有位置不是 X.Y.Z 时失败
#
# 版本号的位置只记在这个脚本里。Package.swift 的 checksum 不在这里改：
# 它要等 xcframework 打包以后才知道，由 prepare-release workflow 写入。
# 需要 npm 和 perl。
set -eu
cd "$(dirname "$0")/.."

check=0
if [ "${1:-}" = --check ]; then
	check=1
	shift
fi
new=${1:?usage: scripts/set-version.sh [--check] X.Y.Z}
if ! echo "$new" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$'; then
	echo "version must look like X.Y.Z: $new" >&2
	exit 2
fi

# 以 Node.js 包的版本为当前版本。
old=$(sed -n 's/^  "version": "\(.*\)",$/\1/p' bindings/node/package.json)
failed=0

# field FILE BEFORE AFTER：FILE 中的 BEFORE<版本>AFTER。改版本时把当前版本换成
# new；检查时确认 BEFORE<new>AFTER 存在，并且没有别的版本留在这个位置。
field() {
	if [ $check = 1 ]; then
		if ! grep -qF "$2$new$3" "$1"; then
			echo "$1: missing $2$new$3" >&2
			failed=1
		fi
		return
	fi
	if ! grep -qF "$2$old$3" "$1"; then
		echo "$1: missing $2$old$3" >&2
		exit 1
	fi
	BEFORE=$2 AFTER=$3 OLD=$old NEW=$new perl -pi -e \
		's/\Q$ENV{BEFORE}$ENV{OLD}$ENV{AFTER}\E/$ENV{BEFORE}$ENV{NEW}$ENV{AFTER}/g' "$1"
}

field bindings/node/package.json '  "version": "' '",'
field bindings/react-native/package.json '  "version": "' '",'
field bindings/react-native/android/build.gradle "version = '" "'"
field bindings/react-native/android/build.gradle 'versionName "' '"'
field bindings/react-native/README.md 'react-native-openhongguo-' '.tgz'
field bindings/flutter/pubspec.yaml 'version: ' ''
field bindings/flutter/android/build.gradle.kts 'version = "' '"'
field bindings/flutter/ios/openhongguo.podspec "s.version          = '" "'"
field bindings/mobile/android/build.gradle.kts 'coordinates("io.github.beilunyang", "openhongguo", "' '")'
field bindings/mobile/README.md 'openhongguo:' '")'
field bindings/mobile/README.md '从 ' ' 开始'
field bindings/mobile/README.md 'from: "' '")'
field Package.swift 'from: "' '")'
field Package.swift 'releases/download/v' '/OpenhongguoCore'
field README.md 'openhongguo:' '")'
field README.md 'from: "' '")'

lock=bindings/react-native/package-lock.json
changelog=bindings/flutter/CHANGELOG.md
if [ $check = 1 ]; then
	lockv=$(sed -n '3s/^  "version": "\(.*\)",$/\1/p' $lock)
	if [ "$lockv" != "$new" ]; then
		echo "$lock: version is $lockv, not $new" >&2
		failed=1
	fi
	if ! grep -qx "## $new" $changelog; then
		echo "$changelog: missing ## $new" >&2
		failed=1
	fi
	exit $failed
fi

# package-lock.json 也记着包自己的版本，交给 npm 改。
(cd bindings/react-native && npm version "$new" --no-git-tag-version --allow-same-version >/dev/null)
# pub.dev 要求 CHANGELOG 有这个版本的一节。没写时补一条指向发布说明的链接。
if ! grep -qx "## $new" $changelog; then
	NOTE="- 见 [v$new 的发布说明](https://github.com/beilunyang/openhongguo/releases/tag/v$new)。" NEW=$new \
		perl -0pi -e 's/^/## $ENV{NEW}\n\n$ENV{NOTE}\n\n/' $changelog
fi
echo "version $old -> $new"
