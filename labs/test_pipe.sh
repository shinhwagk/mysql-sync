#!/usr/bin/env bash

# 设置脚本以在管道命令失败时返回相应的失败状态
set -o pipefail

# 模拟 cmd1
cmd1() {
    while true; do
        echo "cmd1 output"
        sleep 1  # 短暂睡眠以减缓输出
    done
}

# 模拟 cmd2
cmd2() {
    local start_time=$(date +%s)
    while read line; do
        local current_time=$(date +%s)
        local elapsed_time=$((current_time - start_time))

        if [ $elapsed_time -ge 5 ]; then
            echo "cmd2 error after 5 seconds" >&2
            return 1
        else
            echo "cmd2 processes $line"
        fi
        sleep 1  # 短暂睡眠以减缓输出
    done
}

# 模拟 cmd3
cmd3() {
    while read line; do
        echo "cmd3 processes $line"
        sleep 1  # 短暂睡眠以减缓输出
    done
}

# 将函数连接在一起模拟管道
cmd1 | cmd2 | cmd3

# 检查管道命令的退出状态
status=$?
echo "Exit status: $status"
if [ $status -eq 0 ]; then
    echo "All commands executed successfully"
else
    echo "cmd2 failed after 5 seconds"
fi
