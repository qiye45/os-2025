#!/bin/bash

# =================配置区域=================
# 脚本名称（用于进程名前缀）
SCRIPT_NAME=$(basename "$0")
# 基础名称，用于过滤和显示
BASE_NAME="${SCRIPT_NAME%.*}"

# 运行总时长：30分钟
DURATION=1800

# 目标设定
MAX_DEPTH=3           # 最大递归深度
MIN_BRANCHES=3        # 每个节点最少分支数
MAX_BRANCHES=5        # 每个节点最多分支数

# 临时文件用于跨进程原子计数
COUNTER_FILE="/tmp/${BASE_NAME}_counter.tmp"
echo "0" > "$COUNTER_FILE"

# =================函数定义=================

# 获取下一个唯一的 Index (原子操作)
get_next_index() {
    (
        flock -x 200
        local current=$(<"$COUNTER_FILE")
        echo $((current + 1)) > "$COUNTER_FILE"
        echo "$current"
    ) 200>"$COUNTER_FILE.lock"
}

# 递归生成进程树的核心函数
# 参数 $1: 当前深度
spawn_node() {
    local current_depth=$1
    local my_index
    
    # 获取唯一编号
    my_index=$(get_next_index)
    
    # 构造自定义进程名
    local proc_name="${BASE_NAME}_${my_index}"

    # 如果未达到最大深度，尝试生成子进程
    if [ "$current_depth" -lt "$MAX_DEPTH" ]; then
        
        # 随机决定当前节点产生多少个分支
        # 为了保证总数 > 50，我们在前两层强制产生较多分支
        local branches
        if [ "$current_depth" -lt 2 ]; then
            branches=$(( RANDOM % 2 + 3 )) # 3 到 4 个分支
        else
            # 随机 1 到 MAX_BRANCHES
            branches=$(( RANDOM % MAX_BRANCHES + MIN_BRANCHES ))
        fi

        for ((i=0; i<branches; i++)); do
            # 【关键】在后台子shell中递归调用，形成父子关系
            (
                spawn_node $((current_depth + 1))
            ) & 
            # 稍微错开启动时间，避免瞬间系统负载过高
            sleep 0.1
        done
    fi

    # 【核心逻辑】
    # 使用 exec -a 替换当前 Shell 进程为 sleep 进程
    # 这样该进程的 PID 保持不变，它生成的子进程依然挂在它名下
    # 从而在 pstree 中显示为正确的树状结构
    exec -a "$proc_name" sleep "$DURATION"
}

# 清理函数
cleanup() {
    echo ""
    echo ">>> 正在清理进程树..."
    # 杀掉所有包含脚本基础名称的 sleep 进程
    pkill -f "${BASE_NAME}_" 2>/dev/null
    rm -f "$COUNTER_FILE" "$COUNTER_FILE.lock"
    echo ">>> 清理完毕，退出。"
    exit 0
}

# =================主逻辑=================

# 捕获 Ctrl+C 和 退出信号
trap cleanup SIGINT SIGTERM EXIT

echo "----------------------------------------"
echo "启动随机进程树生成器"
echo "基础名称: $BASE_NAME"
echo "持续时间: $DURATION 秒"
echo "----------------------------------------"

# 启动根节点 (在后台启动，这样主脚本可以作为控制器)
(
    spawn_node 0
) &
ROOT_PID=$!

echo "根进程已启动 (PID: $ROOT_PID)"
echo "正在生成子节点，请稍候..."

# 等待几秒让树生长
sleep 5

# 统计生成的进程数
CURRENT_COUNT=$(pgrep -f "${BASE_NAME}_" | wc -l)
echo "----------------------------------------"
echo "进程树生成完毕！"
echo "当前存活子进程数: $CURRENT_COUNT"
echo "----------------------------------------"
echo "请在另一个终端执行以下命令查看效果："
echo "pstree -p $ROOT_PID"
echo "或者："
echo "ps -ef | grep $BASE_NAME | grep -v grep"
echo "----------------------------------------"
echo "脚本将运行至 $(date -d "+${DURATION} seconds" "+%Y-%m-%d %H:%M:%S")"
echo "按 Ctrl+C 可提前结束。"

# 主循环：保持主脚本运行，直到时间结束
END_TIME=$(( $(date +%s) + DURATION ))
while [ $(date +%s) -lt $END_TIME ]; do
    sleep 10
    # 检查根进程是否还活着
    if ! kill -0 $ROOT_PID 2>/dev/null; then
        echo "根进程意外退出，终止脚本。"
        break
    fi
done

# 退出时会触发 trap cleanup