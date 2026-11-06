#!/bin/sh
# git commit 之前先跑測試。Claude Code 只有在 hook 回 2 時才會擋下工具呼叫，
# go test 失敗回的是 1，直接接在 settings.json 裡擋不住，所以由這支轉換。
# matcher 比對的是工具名稱，看不到指令內容，是不是 commit 在這裡判斷。
input=$(cat)
printf '%s' "$input" | grep -Eq '"command"[[:space:]]*:[[:space:]]*"([^"\\]|\\.)*git[[:space:]]+([^"\\]|\\.)*commit' || exit 0
cd "${CLAUDE_PROJECT_DIR:-.}" || exit 2
if ! go test ./... >&2; then
	echo 'go test 沒過，這次 commit 已擋下。' >&2
	exit 2
fi
