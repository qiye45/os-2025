package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// SyscallStat 系统调用统计信息
type SyscallStat struct {
	Name     string
	Duration time.Duration
}

func main() {
	// 检查参数数量，如果少于2个则打印用法并退出
	// 获取命令名称和参数列表
	_, cmdArgs := filepath.Base(os.Args[0]), os.Args[1:]
	if len(cmdArgs) < 1 {
		printUsage()
		os.Exit(1)
	}
	fmt.Println(cmdArgs)
	cmdName := cmdArgs[0]
	cmdArgs = cmdArgs[1:]
	// 使用 syscall.Pipe 创建管道
	// pipefd[0] 为读端，pipefd[1] 为写端
	r, w, err := os.Pipe()
	if err != nil {
		return
	}
	// 查找 strace 的绝对路径
	// 尝试常见路径: /usr/bin/strace, /bin/strace
	// 或从 PATH 环境变量中搜索
	stracePath, err := findCommandPath("strace")
	if err != nil {
		return
	}
	cmdPath, err := findCommandPath(cmdName)
	if err != nil {
		return
	}
	if isExecutable(stracePath) == false {
		return
	}
	if isExecutable(cmdPath) == false {
		return
	}
	// exec_argv: ["strace", "-T", COMMAND, ARG1, ARG2, ...]
	// exec_envp: 需要传入 PATH 环境变量，否则 strace 无法找到命令
	// 示例: char *exec_envp[] = { "PATH=/bin:/usr/bin", NULL }
	args := append([]string{"-T", cmdPath}, cmdArgs...)

	// 使用 syscall.ForkExec 或手动 fork
	// 子进程：
	//   - 关闭管道读端
	//   - 将管道写端复制到 stderr (strace 输出到 stderr)
	//   - 使用 execve 执行 strace -T COMMAND ARG...
	//   - strace -T 选项会输出每个系统调用的耗时，格式如 <0.000011>
	// 父进程：
	//   - 关闭管道写端
	//   - 从管道读端读取 strace 输出
	cmd := exec.Command(stracePath, args...)
	fmt.Println("cmd:", cmd)
	cmd.Stderr = w
	err = cmd.Start() // 阻塞启动
	if err != nil {
		return
	}
	err = w.Close() // 父进程关闭写端
	if err != nil {
		return
	}

	// 使用 bufio.Scanner 或逐行读取
	// 解析每行输出，提取系统调用名称和耗时
	// strace 输出格式示例:
	//   read(3, "...", 4096) = 1024 <0.000123>
	//   mmap(NULL, 4096, ...) = 0x7f... <0.000045>
	scanner := bufio.NewScanner(r)
	lastPrintTime := time.Now()
	syscallStatMap := make(map[string]time.Duration)
	for scanner.Scan() {
		line := scanner.Text()
		// 提取系统调用名称: 行首到第一个 '(' 之间的字符串
		// 提取耗时: 行尾 <...> 中的数字，转换为 time.Duration
		// 注意: 需要处理特殊情况，如程序输出可能干扰解析
		// 建议使用正则表达式: `^(\w+)\(.*<(\d+\.\d+)>$`
		syscallName, duration, ok := parseStraceLine(line)
		if !ok {
			continue
		}

		// map[string]time.Duration 用于存储每个系统调用的总耗时
		// 每次解析到一行，累加对应系统调用的耗时
		syscallStatMap[syscallName] += duration

		// 记录上次打印时间 lastPrintTime
		// 如果距离上次打印超过 100ms，调用 printStats 打印当前统计
		// 打印后更新 lastPrintTime
		if time.Now().Sub(lastPrintTime) > 100*time.Millisecond {
			printStats(syscallStatMap)
			lastPrintTime = time.Now()
		}
	}

	// 使用 syscall.Wait4 等待 strace 进程结束
	// 调用 printStats 打印最终统计结果
	printStats(syscallStatMap)

}

// findCommandPath 在 PATH 中搜索命令的绝对路径
func findCommandPath(cmd string) (string, error) {
	// 如果命令包含 '/'，直接返回
	// 获取 PATH 环境变量
	// 遍历 PATH 中的每个目录
	// 拼接完整路径并检查文件是否存在且可执行
	// 返回找到的第一个可执行文件路径
	path, err := exec.LookPath(cmd)
	if err != nil {
		return "", err
	}
	return path, nil
}

// parseStraceLine 解析 strace 输出行
func parseStraceLine(line string) (syscallName string, duration time.Duration, ok bool) {
	// 使用正则表达式解析 strace 输出
	// 格式: syscall_name(...) = result <time>
	// 示例: read(3, "...", 4096) = 1024 <0.000123>
	//
	// 正则表达式建议: `^(\w+)\(.*<(\d+\.\d+)>$`
	// 或更精确的: `^(\w+)\(.*\)\s*=\s*.*<(\d+\.\d+)>$`
	//
	// 注意边界情况:
	// 1. 未完成的系统调用可能没有 <time>
	// 2. 程序输出可能干扰解析 (如 echo '", 1) = 100 <99999.9>')
	// 3. 信号中断的系统调用格式可能不同
	re := regexp.MustCompile("^(\\w+)\\(.*<(\\d+\\.\\d+)>$")
	match := re.FindStringSubmatch(line)
	if len(match) == 3 {
		timeDuration, _ := strconv.ParseFloat(match[2], 64)
		return match[1], time.Duration(float64(time.Second) * timeDuration), true
	}
	return "", 0, false
}

// printStats 打印系统调用统计信息
func printStats(stats map[string]time.Duration) {
	// 计算总耗时
	// 将 map 转换为切片并按耗时排序
	// 打印 Top 5 系统调用
	// 格式: printf("%s (%d%%)\n", syscall_name, ratio)
	// 打印 80 个 \0 作为分隔符
	syscallStatList := make([]SyscallStat, len(stats))
	totalDuration := time.Duration(0)
	for syscallName, duration := range stats {
		syscallStatList = append(syscallStatList, SyscallStat{syscallName, duration})
		totalDuration += duration
	}
	sort.Slice(syscallStatList, func(i, j int) bool {
		return syscallStatList[i].Duration > syscallStatList[j].Duration
	})
	fmt.Print(strings.Repeat("=", 80) + "\n")
	for i, syscallStat := range syscallStatList {
		if totalDuration == 0 {
			continue
		}
		if i >= 10 {
			break
		}
		fmt.Printf("%s (%.2fms)[%.2f%%]\n", syscallStat.Name, float64(syscallStat.Duration)/float64(time.Millisecond), float64(syscallStat.Duration)/float64(totalDuration)*100)
	}
	fmt.Print(strings.Repeat("=", 80) + "\n")
}

// isExecutable 检查文件是否可执行
func isExecutable(path string) bool {
	// 使用 syscall.Access 检查文件是否可执行
	// 或使用 os.Stat 检查文件权限
	return syscall.Access(path, 0x1) == nil
}

// 辅助函数: 打印用法信息
func printUsage() {
	fmt.Println("Usage: sperf COMMAND [ARG]...")
}
