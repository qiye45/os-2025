#!/usr/bin/env python3
"""
系统调用性能可视化工具

用法:
    ./sperf COMMAND [ARG]... | python visualize.py
    
或者从文件读取:
    python visualize.py < output.txt

输入格式示例:
================================================================================
openat (0.48ms)[23.95%]
read (0.36ms)[18.32%]
newfstatat (0.21ms)[10.49%]
execve (0.19ms)[9.34%]
close (0.17ms)[8.43%]
rt_sigaction (0.16ms)[7.98%]
mmap (0.15ms)[7.58%]
mprotect (0.09ms)[4.72%]
fstat (0.07ms)[3.51%]
socket (0.04ms)[1.76%]
================================================================================
"""

import sys
import re
from typing import List, Tuple

# ANSI 颜色代码
COLORS = [
    '\033[91m',  # 红色
    '\033[92m',  # 绿色
    '\033[93m',  # 黄色
    '\033[94m',  # 蓝色
    '\033[95m',  # 紫色
    '\033[96m',  # 青色
    '\033[97m',  # 白色
    '\033[33m',  # 橙色 (深黄)
    '\033[35m',  # 洋红
    '\033[36m',  # 深青
]
RESET = '\033[0m'
BOLD = '\033[1m'


def parse_syscall_line(line: str) -> Tuple[str, float, float]:
    """
    解析系统调用统计行
    
    格式: syscall_name (X.XXms)[XX.XX%]
    返回: (syscall_name, duration_ms, percentage)
    """
    pattern = r'^(\w+)\s+\(([\d.]+)ms\)\[([\d.]+)%\]$'
    match = re.match(pattern, line.strip())
    if match:
        name = match.group(1)
        duration = float(match.group(2))
        percentage = float(match.group(3))
        return (name, duration, percentage)
    return None


def parse_input(lines: List[str]) -> List[List[Tuple[str, float, float]]]:
    """
    解析输入，支持多个命令的输出
    
    返回: 每个命令的系统调用统计列表
    """
    results = []
    current_block = []
    in_block = False
    
    for line in lines:
        line = line.strip()
        
        # 检测分隔符
        if line.startswith('=' * 10):
            if in_block and current_block:
                results.append(current_block)
                current_block = []
            in_block = not in_block
            continue
        
        # 解析系统调用行
        if in_block:
            parsed = parse_syscall_line(line)
            if parsed:
                current_block.append(parsed)
    
    # 处理最后一个块
    if current_block:
        results.append(current_block)
    
    return results


def draw_bar(percentage: float, width: int = 50, color: str = '') -> str:
    """绘制进度条"""
    filled = int(width * percentage / 100)
    bar = '█' * filled + '░' * (width - filled)
    if color:
        return f"{color}{bar}{RESET}"
    return bar


def visualize_syscalls(syscalls: List[Tuple[str, float, float]], title: str = ""):
    """可视化单个命令的系统调用统计"""
    if not syscalls:
        return
    
    # 打印标题
    if title:
        print(f"\n{BOLD}{'=' * 70}{RESET}")
        print(f"{BOLD}{title}{RESET}")
        print(f"{BOLD}{'=' * 70}{RESET}")
    
    # 找出最长的系统调用名称，用于对齐
    max_name_len = max(len(s[0]) for s in syscalls)
    
    print()
    print(f"{'系统调用':<{max_name_len + 4}} {'耗时':>10} {'占比':>8}  {'图表'}")
    print("-" * 80)
    
    for i, (name, duration, percentage) in enumerate(syscalls):
        color = COLORS[i % len(COLORS)]
        bar = draw_bar(percentage, 40, color)
        
        # 格式化输出
        print(f"{color}{name:<{max_name_len + 4}}{RESET} "
              f"{duration:>8.2f}ms "
              f"{percentage:>6.2f}%  "
              f"{bar}")
    
    print()


def visualize_comparison(all_syscalls: List[List[Tuple[str, float, float]]]):
    """可视化多个命令的对比"""
    if len(all_syscalls) <= 1:
        return
    
    print(f"\n{BOLD}{'=' * 70}{RESET}")
    print(f"{BOLD}多命令对比视图{RESET}")
    print(f"{BOLD}{'=' * 70}{RESET}")
    
    # 收集所有系统调用名称
    all_names = set()
    for syscalls in all_syscalls:
        for name, _, _ in syscalls:
            all_names.add(name)
    
    # 创建对比表格
    print()
    header = f"{'系统调用':<20}"
    for i in range(len(all_syscalls)):
        header += f" {'命令' + str(i+1):>12}"
    print(header)
    print("-" * (20 + 13 * len(all_syscalls)))
    
    for name in sorted(all_names):
        row = f"{name:<20}"
        for i, syscalls in enumerate(all_syscalls):
            color = COLORS[i % len(COLORS)]
            found = False
            for n, duration, percentage in syscalls:
                if n == name:
                    row += f" {color}{percentage:>10.2f}%{RESET}"
                    found = True
                    break
            if not found:
                row += f" {'-':>12}"
        print(row)
    
    print()


def print_pie_chart(syscalls: List[Tuple[str, float, float]]):
    """打印简单的饼图表示（使用字符）"""
    if not syscalls:
        return
    
    print(f"\n{BOLD}饼图视图:{RESET}")
    print()
    
    # 使用不同字符表示不同的系统调用
    chars = ['●', '○', '◆', '◇', '■', '□', '▲', '△', '★', '☆']
    
    # 计算每个系统调用在100个字符中占多少
    total_chars = 100
    chart_line = ""
    legend = []
    
    for i, (name, duration, percentage) in enumerate(syscalls):
        color = COLORS[i % len(COLORS)]
        char = chars[i % len(chars)]
        count = int(percentage)
        chart_line += f"{color}{char * count}{RESET}"
        legend.append(f"{color}{char} {name} ({percentage:.1f}%){RESET}")
    
    # 打印饼图
    print(f"[{chart_line}]")
    print()
    
    # 打印图例
    print("图例:")
    for item in legend:
        print(f"  {item}")
    print()


def main():
    """主函数"""
    # 读取输入
    if sys.stdin.isatty():
        print(__doc__)
        print("请通过管道输入数据，例如:")
        print("  ./sperf ps -ef | python visualize.py")
        return
    
    lines = sys.stdin.readlines()
    
    # 解析输入
    all_syscalls = parse_input(lines)
    
    if not all_syscalls:
        print("未能解析到有效的系统调用统计数据")
        return
    
    # 可视化每个命令的结果
    for i, syscalls in enumerate(all_syscalls):
        title = f"命令 {i + 1} 系统调用统计" if len(all_syscalls) > 1 else "系统调用统计"
        visualize_syscalls(syscalls, title)
        print_pie_chart(syscalls)
    
    # 如果有多个命令，显示对比视图
    if len(all_syscalls) > 1:
        visualize_comparison(all_syscalls)


if __name__ == "__main__":
    main()