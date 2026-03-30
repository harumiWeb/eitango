#!/usr/bin/env sh
set -eu

REPO="${EITANGO_INSTALL_REPO:-harumiWeb/eitango}"
API_BASE="${EITANGO_INSTALL_API_BASE:-https://api.github.com}"
DOWNLOAD_BASE="${EITANGO_INSTALL_DOWNLOAD_BASE:-https://github.com}"
APP_NAME="eitango"
INSTALL_ROOT="${HOME}/.eitango"
BIN_DIR="${INSTALL_ROOT}/bin"
BIN_PATH="${BIN_DIR}/${APP_NAME}"
SHARE_DIR="${INSTALL_ROOT}/share"
VERSION_FILE="${INSTALL_ROOT}/version"

MODE="install"
REQUESTED_VERSION=""
PURGE_DATA=0

DOWNLOAD_DIR=""
STAGE_DIR=""
BACKUP_DIR=""

usage() {
	cat <<'EOF'
Usage:
  install.sh [--version vX.Y.Z]
  install.sh --uninstall [--purge-data]
  install.sh --help

Options:
  --version TAG   Install a specific GitHub Release tag.
  --uninstall     Remove the installer-managed ~/.eitango tree.
  --purge-data    With --uninstall, also remove the current data directory.
  --help          Show this help.

Notes:
  - This installer supports macOS and Linux only.
  - Downloaded archives are verified against release checksums.txt.
  - PATH is never modified automatically.
EOF
}

say() {
	printf '%s\n' "$*"
}

die() {
	printf '%s\n' "Error: $*" >&2
	exit 1
}

cleanup() {
	if [ -n "${STAGE_DIR}" ] && [ -d "${STAGE_DIR}" ]; then
		rm -rf "${STAGE_DIR}"
	fi
	if [ -n "${DOWNLOAD_DIR}" ] && [ -d "${DOWNLOAD_DIR}" ]; then
		rm -rf "${DOWNLOAD_DIR}"
	fi
}

trap cleanup EXIT INT TERM

require_cmd() {
	command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

normalize_version() {
	raw="$1"
	raw="$(printf '%s' "$raw" | tr -d '\r\n')"
	[ -n "$raw" ] || die "--version requires a value"
	case "$raw" in
		v*|V*) printf '%s\n' "v${raw#?}" ;;
		*) printf 'v%s\n' "$raw" ;;
	esac
}

resolve_os() {
	raw="$(uname -s)"
	case "$raw" in
		Darwin) printf '%s\n' "darwin" ;;
		Linux) printf '%s\n' "linux" ;;
		*) die "unsupported OS: $raw (use GitHub Releases manually on this platform)" ;;
	esac
}

resolve_arch() {
	raw="$(uname -m)"
	case "$raw" in
		x86_64|amd64) printf '%s\n' "x86_64" ;;
		arm64|aarch64) printf '%s\n' "arm64" ;;
		*) die "unsupported architecture: $raw" ;;
	esac
}

default_data_dir() {
	if [ -n "${EITANGO_DATA_DIR:-}" ]; then
		printf '%s\n' "${EITANGO_DATA_DIR}"
		return
	fi
	case "$1" in
		darwin) printf '%s\n' "${HOME}/Library/Application Support/eitango-cli" ;;
		linux) printf '%s\n' "${HOME}/.local/share/eitango-cli" ;;
		*) die "unsupported OS for data dir: $1" ;;
	esac
}

resolve_latest_version() {
	url="${API_BASE}/repos/${REPO}/releases/latest"
	response="$(curl -fsSL "$url")" || die "failed to fetch latest release metadata from $url"
	version="$(printf '%s\n' "$response" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)"
	[ -n "$version" ] || die "failed to parse tag_name from latest release metadata"
	printf '%s\n' "$version"
}

archive_name() {
	version="$1"
	version="${version#v}"
	version="${version#V}"
	printf '%s\n' "${APP_NAME}_${version}_$2_$3.tar.gz"
}

checksum_file_name() {
	printf '%s\n' "checksums.txt"
}

release_url() {
	printf '%s\n' "${DOWNLOAD_BASE}/${REPO}/releases/download/$1/$2"
}

try_sha256sum() {
	sha256sum "$1" 2>/dev/null | awk 'NR==1 { print $1 }'
}

try_shasum() {
	shasum -a 256 "$1" 2>/dev/null | awk 'NR==1 { print $1 }'
}

try_openssl() {
	openssl dgst -sha256 "$1" 2>/dev/null | awk 'NR==1 { print $NF }'
}

compute_sha256() {
	for tool in sha256sum shasum openssl; do
		if command -v "$tool" >/dev/null 2>&1; then
			case "$tool" in
				sha256sum) sum="$(try_sha256sum "$1")" ;;
				shasum) sum="$(try_shasum "$1")" ;;
				openssl) sum="$(try_openssl "$1")" ;;
			esac
			if [ -n "${sum:-}" ]; then
				printf '%s\n' "$sum"
				return 0
			fi
		fi
	done
	return 1
}

expected_checksum() {
	result="$(awk -v name="$2" '$2 == name { print $1 }' "$1")"
	count="$(printf '%s\n' "$result" | sed '/^$/d' | wc -l | tr -d ' ')"
	[ "$count" = "1" ] || die "checksum entry for $2 was not found exactly once"
	printf '%s\n' "$result"
}

verify_archive() {
	expected="$(expected_checksum "$1" "$2")"
	actual="$(compute_sha256 "$3")" || die "no usable SHA256 tool found (need sha256sum, shasum, or openssl)"
	[ "$expected" = "$actual" ] || die "checksum mismatch for $2"
}

prepare_stage() {
	parent="$(dirname "$INSTALL_ROOT")"
	mkdir -p "$parent"
	STAGE_DIR="$(mktemp -d "${parent}/.eitango-install.XXXXXX")" || die "failed to create install staging directory"
	mkdir -p "${STAGE_DIR}/bin" "${STAGE_DIR}/share"
}

copy_required_file() {
	src="$1"
	dst="$2"
	[ -f "$src" ] || die "release archive is missing required file: $src"
	cp "$src" "$dst"
}

build_stage() {
	extracted="$1"
	copy_required_file "${extracted}/${APP_NAME}" "${STAGE_DIR}/bin/${APP_NAME}"
	chmod 755 "${STAGE_DIR}/bin/${APP_NAME}"
	copy_required_file "${extracted}/LICENSE" "${STAGE_DIR}/share/LICENSE"
	copy_required_file "${extracted}/README.md" "${STAGE_DIR}/share/README.md"
	copy_required_file "${extracted}/README.en.md" "${STAGE_DIR}/share/README.en.md"
	copy_required_file "${extracted}/THIRD_PARTY_NOTICES.md" "${STAGE_DIR}/share/THIRD_PARTY_NOTICES.md"
	[ -d "${extracted}/third_party/licenses" ] || die "release archive is missing third_party/licenses"
	mkdir -p "${STAGE_DIR}/share/third_party"
	cp -R "${extracted}/third_party/licenses" "${STAGE_DIR}/share/third_party/licenses"
	printf '%s\n' "$2" > "${STAGE_DIR}/version"
}

replace_install_root() {
	if [ -e "${INSTALL_ROOT}" ]; then
		BACKUP_DIR="${INSTALL_ROOT}.backup.$$"
		rm -rf "${BACKUP_DIR}"
		mv "${INSTALL_ROOT}" "${BACKUP_DIR}" || die "failed to back up existing install root"
	fi
	if mv "${STAGE_DIR}" "${INSTALL_ROOT}"; then
		STAGE_DIR=""
		if [ -n "${BACKUP_DIR}" ] && [ -d "${BACKUP_DIR}" ]; then
			rm -rf "${BACKUP_DIR}"
			BACKUP_DIR=""
		fi
		return
	fi
	if [ -n "${BACKUP_DIR}" ] && [ -d "${BACKUP_DIR}" ]; then
		if mv "${BACKUP_DIR}" "${INSTALL_ROOT}"; then
			BACKUP_DIR=""
			die "failed to replace ${INSTALL_ROOT}; restored previous install"
		fi
		die "failed to replace ${INSTALL_ROOT}; previous install kept at ${BACKUP_DIR}"
	fi
	die "failed to replace ${INSTALL_ROOT}"
}

path_contains_bin_dir() {
	case ":${PATH}:" in
		*":${BIN_DIR}:"*) return 0 ;;
		*) return 1 ;;
	esac
}

maybe_warn_shadowed_binary() {
	found="$(command -v "${APP_NAME}" 2>/dev/null || true)"
	if [ -n "$found" ] && [ "$found" != "${BIN_PATH}" ]; then
		say "Warning: another ${APP_NAME} is earlier on PATH: ${found}"
	fi
}

run_uninstall() {
	os_name="$(resolve_os)"
	data_dir="$(default_data_dir "$os_name")"
	if [ -d "${INSTALL_ROOT}" ]; then
		rm -rf "${INSTALL_ROOT}"
		say "Removed ${INSTALL_ROOT}"
	else
		say "No installer-managed files found at ${INSTALL_ROOT}"
	fi
	if [ "${PURGE_DATA}" -eq 1 ]; then
		if [ -d "$data_dir" ]; then
			rm -rf "$data_dir"
			say "Removed learning data at ${data_dir}"
		else
			say "No learning data found at ${data_dir}"
		fi
	else
		say "Learning data was kept."
		say "Run again with --uninstall --purge-data to remove the current data directory."
	fi
}

while [ "$#" -gt 0 ]; do
	case "$1" in
		--version)
			[ "$MODE" = "install" ] || die "--version cannot be used with --uninstall"
			shift
			[ "$#" -gt 0 ] || die "--version requires a value"
			REQUESTED_VERSION="$(normalize_version "$1")"
			;;
		--uninstall)
			[ -z "$REQUESTED_VERSION" ] || die "--uninstall cannot be combined with --version"
			MODE="uninstall"
			;;
		--purge-data)
			PURGE_DATA=1
			;;
		--help|-h)
			usage
			exit 0
			;;
		*)
			die "unknown argument: $1 (use --help)"
			;;
	esac
	shift
done

[ "$MODE" = "uninstall" ] || [ "${PURGE_DATA}" -eq 0 ] || die "--purge-data requires --uninstall"

require_cmd curl
require_cmd tar
require_cmd mktemp
require_cmd cp
require_cmd mv
require_cmd rm
require_cmd mkdir
require_cmd chmod
require_cmd awk
require_cmd sed
require_cmd grep
require_cmd head
require_cmd wc
require_cmd tr

if [ "$MODE" = "uninstall" ]; then
	run_uninstall
	exit 0
fi

OS_NAME="$(resolve_os)"
ARCH_NAME="$(resolve_arch)"
VERSION="${REQUESTED_VERSION}"
if [ -z "${VERSION}" ]; then
	VERSION="$(resolve_latest_version)"
fi

ARCHIVE_NAME="$(archive_name "${VERSION}" "${OS_NAME}" "${ARCH_NAME}")"
CHECKSUM_NAME="$(checksum_file_name)"
ARCHIVE_URL="$(release_url "${VERSION}" "${ARCHIVE_NAME}")"
CHECKSUM_URL="$(release_url "${VERSION}" "${CHECKSUM_NAME}")"

say "Installing ${APP_NAME} ${VERSION} for ${OS_NAME}/${ARCH_NAME}..."

DOWNLOAD_DIR="$(mktemp -d "${TMPDIR:-/tmp}/eitango-download.XXXXXX")" || die "failed to create download directory"
ARCHIVE_PATH="${DOWNLOAD_DIR}/${ARCHIVE_NAME}"
CHECKSUM_PATH="${DOWNLOAD_DIR}/${CHECKSUM_NAME}"
EXTRACT_DIR="${DOWNLOAD_DIR}/extract"
mkdir -p "${EXTRACT_DIR}"

curl -fsSL "${ARCHIVE_URL}" -o "${ARCHIVE_PATH}" || die "failed to download ${ARCHIVE_URL}"
curl -fsSL "${CHECKSUM_URL}" -o "${CHECKSUM_PATH}" || die "failed to download ${CHECKSUM_URL}"
verify_archive "${CHECKSUM_PATH}" "${ARCHIVE_NAME}" "${ARCHIVE_PATH}"

tar -xzf "${ARCHIVE_PATH}" -C "${EXTRACT_DIR}" || die "failed to extract ${ARCHIVE_NAME}"

prepare_stage
build_stage "${EXTRACT_DIR}" "${VERSION}"
replace_install_root

say "Installed to ${BIN_PATH}"
say "Bundled notices copied to ${SHARE_DIR}"
say "Installed version recorded in ${VERSION_FILE}"
maybe_warn_shadowed_binary

if path_contains_bin_dir; then
	say "PATH already contains ${BIN_DIR}"
else
	say "Add this to your shell config:"
	say "  export PATH=\"${BIN_DIR}:\$PATH\""
fi

say "Run:"
say "  ${APP_NAME}"
