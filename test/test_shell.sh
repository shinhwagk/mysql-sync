#!/bin/bash

# 初始 JSON 对象
json='{"abc":1}'

# 增加 abc 的值
increment_value() {
    local json=$1
    local add_value=$2  # 这是你想要增加的值
    echo "$json" | jq ".abc += $add_value"
}

# 使用函数增加 abc 的值
new_json=$(increment_value "$json" 5)  # 增加 5
echo "New JSON: $new_json"

# 再次增加 abc 的值
new_json=$(increment_value "$new_json" 3)  # 再增加 3
echo "New JSON: $new_json"
