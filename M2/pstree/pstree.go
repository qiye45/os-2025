package main

import (
	"flag"
	"fmt"
	"os"
)

const VersionInfo = "pstree (Go implementation)"

type Process struct {
	PID      int
	PPID     int
	Name     string
	Children []*Process
}

func main() {
	showPids := flag.Bool("p", false, "Show PIDs")
	showPidsLong := flag.Bool("show-pids", false, "Show PIDs")
	numericSort := flag.Bool("n", false, "Sort by PID")
	numericSortLong := flag.Bool("numeric-sort", false, "Sort by PID")
	version := flag.Bool("V", false, "Show version")
	versionLong := flag.Bool("version", false, "Show version")

	flag.Parse()

	if *version || *versionLong {
		fmt.Println(VersionInfo)
		os.Exit(0)
	}

	if flag.NArg() > 0 {
		fmt.Println("Usage: pstree [-p|--show-pids] [-n|--numeric-sort] [-V|--version]")
		os.Exit(1)
	}

	showPid := *showPids || *showPidsLong
	sortByPid := *numericSort || *numericSortLong

	processes, err := ReadProcesses()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading processes: %v\n", err)
		os.Exit(1)
	}

	tree := BuildTree(processes)
	if sortByPid {
		SortTree(tree)
	}

	PrintTree(tree, "", true, showPid)
	os.Exit(0)
}

// ReadProcesses 读取系统中所有进程信息
// TODO: 实现从 /proc 读取进程信息
func ReadProcesses() (map[int]*Process, error) {
	// 提示：
	// 1. 遍历 /proc 目录
	// 2. 对于每个数字命名的目录，读取 /proc/[pid]/stat
	// 3. 解析 stat 文件获取 PID, PPID, Name
	// 4. 返回 map[pid]*Process
	return nil, nil
}

// BuildTree 构建进程树
// TODO: 实现进程树构建
func BuildTree(processes map[int]*Process) []*Process {
	// 提示：
	// 1. 遍历所有进程
	// 2. 将每个进程添加到其父进程的 Children 列表
	// 3. 找到所有根进程（PPID 为 0 或父进程不存在）
	// 4. 返回根进程列表
	return nil
}

// SortTree 按 PID 排序进程树
// TODO: 实现进程树排序
func SortTree(roots []*Process) {
	// 提示：
	// 1. 对根进程列表按 PID 排序
	// 2. 递归对每个进程的子进程排序
}

// PrintTree 打印进程树
// TODO: 实现进程树打印
func PrintTree(roots []*Process, prefix string, isLast bool, showPid bool) {
	// 提示：
	// 1. 遍历根进程列表
	// 2. 打印当前进程（使用 prefix 控制缩进）
	// 3. 递归打印子进程
	// 4. 使用树形字符：─ ├ └ │
	// 5. 如果 showPid 为 true，显示 PID
}
