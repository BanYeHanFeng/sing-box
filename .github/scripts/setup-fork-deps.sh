#!/usr/bin/env bash
# 在 GitHub Actions 中检出 fork 依赖仓库并生成 go.work
# 使用方法: bash .github/scripts/setup-fork-deps.sh [quic_go_ref] [sing_quic_ref]
#
# 默认引用分支:
#   quic-go   -> udpx-dev
#   sing-quic -> udpx-dev
#
# 环境变量:
#   FORK_QUIC_GO_REPO   (默认: https://github.com/BanYeHanFeng/quic-go.git)
#   FORK_SING_QUIC_REPO (默认: https://github.com/BanYeHanFeng/sing-quic.git)
#   FORK_QUIC_GO_REF    (默认: udpx-dev)
#   FORK_SING_QUIC_REF  (默认: udpx-dev)
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
# workspace 会合并各 use 模块 go.mod 的 replace，fork 里锁的版本和主模块不一致会直接冲突，
# 统一以主模块 go.mod 的 replace 为准
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
cat > "${WORKSPACE}/go.work" <<EOF
go 1.25.5

use (
	.
	../quic-go
	../sing-quic
)
EOF

echo "go.work 内容："
cat "${WORKSPACE}/go.work"
echo "::endgroup::"

echo "✅ fork 依赖已就绪"
