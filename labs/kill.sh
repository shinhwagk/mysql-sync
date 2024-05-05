#!/bin/bash

# 定义信号处理函数
cleanup() {
    echo "Received kill signal. Cleaning up..."
    # 在这里添加清理逻辑，比如关闭文件描述符、清理临时文件等
    # 在脚本退出时，这个函数将会被调用
    # exit 0
}

# 注册信号处理函数
trap cleanup SIGINT SIGTERM

# 脚本主体部分
echo "Script is running. PID: $$"

# 模拟脚本工作
while true; do
    sleep 1
done
