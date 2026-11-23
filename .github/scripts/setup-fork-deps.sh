#!/usr/bin/env bash
set -euo pipefail

WORKSPACE="${GITHUB_WORKSPACE:-$(pwd)}"
PARENT="$(dirname "${WORKSPACE}")"

QUIC_GO_REPO="${FORK_QUIC_GO_REPO:-https://github.com/BanYeHanFeng/quic-go.git}"
SING_QUIC_REPO="${FORK_SING_QUIC_REPO:-https://github.com/BanYeHanFeng/sing-quic.git}"
QUIC_GO_REF="${1:-${FORK_QUIC_GO_REF:-udpx-dev}}"
SING_QUIC_REF="${2:-${FORK_SING_QUIC_REF:-udpx-dev}}"

echo "::group::检出 fork 依赖仓库"
echo "quic-go   -> ${QUIC_GO_REPO} @ ${QUIC_GO_REF}"
echo "sing-quic -> ${SING_QUIC_REPO} @ ${SING_QUIC_REF}"

git clone --depth=1 -b "${QUIC_GO_REF}" "${QUIC_GO_REPO}" "${PARENT}/quic-go"
git clone --depth=1 -b "${SING_QUIC_REF}" "${SING_QUIC_REPO}" "${PARENT}/sing-quic"

echo "::endgroup::"

echo "::group::清理 fork 依赖的 replace"
for fork in "${PARENT}/quic-go" "${PARENT}/sing-quic"; do
	[[ -f "${fork}/go.mod" ]] || continue
	while IFS= read -r module; do
		[[ -n "${module}" ]] || continue
		go mod edit -dropreplace="${module}" "${fork}/go.mod"
		echo "${fork}: 移除 replace ${module}"
	done < <(go mod edit -json "${fork}/go.mod" | python3 -c 'import json,sys; [print(r["Old"]["Path"]) for r in (json.load(sys.stdin).get("Replace") or [])]')
done
echo "::endgroup::"

echo "::group::生成 go.work"
(
	cd "${WORKSPACE}"
	go work init .
	go work use ../quic-go ../sing-quic
)

echo "go.work 内容："
cat "${WORKSPACE}/go.work"
echo "::endgroup::"

echo "✅ fork 依赖已就绪"
