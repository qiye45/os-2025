package main

import (
	"container/list"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

const VersionInfo = "pstree (Go implementation)"

type Process struct {
	PID      int64
	PPID     int64
	Name     string
	Children []*Process
}

type Node struct {
	count  int
	isLast bool
}

func (n Node) String() string {
	return fmt.Sprintf("%d,%v", n.count, n.isLast)
}

var (
	row int64 = 0
)

type Deque struct {
	list list.List
	m    map[int]*Node
}

func (d *Deque) PushBack(x int, isLast bool) {
	_, ok := d.m[x]
	if !ok {
		d.m[x] = &Node{
			count:  0,
			isLast: isLast,
		}
	}
	d.m[x].count += 1
	d.list.PushBack(x)
}

// PopBack 注意传指针
// 使用值接收者时，它修改的是 deque 的副本，而不是原始对象
func (d *Deque) PopBack() {
	if d.list.Len() == 0 {
		return
	}
	tmp := d.list.Back()
	d.list.Remove(tmp)
	value, ok := tmp.Value.(int)
	if !ok {
		return
	}
	d.m[value].count -= 1
	if d.m[value].count == 0 {
		delete(d.m, value)
	}
}
func (d *Deque) Get(x int) *Node {
	value, ok := d.m[x]
	if !ok {
		return &Node{}
	}
	return value
}

func (d *Deque) GetMax() int {
	tmp := -1
	for k, v := range d.m {
		if v.count > 0 {
			tmp = max(tmp, k)
		}
	}
	return tmp
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
		_, err := fmt.Fprintf(os.Stderr, "Error reading processes: %v\n", err)
		if err != nil {
			return
		}
		os.Exit(1)
	}

	tree := BuildTree(processes)
	symbolList := Deque{list: list.List{}, m: make(map[int]*Node)}
	PrintTree(tree.Children[0], 0, &symbolList, showPid, sortByPid, true, true, false)
	os.Exit(0)
}

// ReadProcesses 读取系统中所有进程信息
func ReadProcesses() (map[int64]*Process, error) {
	// 提示：
	// 1. 遍历 /proc 目录
	processDirs, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	// 2. 对于每个数字命名的目录，读取 /proc/[pid]/stat
	// 3. 解析 stat 文件获取 PID, PPID, Name
	// pid到进程的映射
	processes := make(map[int64]*Process)
	processes[0] = &Process{
		PID:  0,
		PPID: 0,
		Name: "init",
	}
	// 未处理的进程
	unhandled := make(map[int64]*Process)
	orphanProcessCount := 0
	for _, dir := range processDirs {
		if !dir.IsDir() {
			continue
		}
		pid := dir.Name()
		statPath := fmt.Sprintf("/proc/%s/stat", pid)
		statFile, err := os.Open(statPath)
		if err != nil {
			continue
		}
		stat, err := os.ReadFile(statPath)
		if err != nil {
			continue
		}
		process, err := ParseStat(stat)
		if err != nil {
			continue
		}
		processes[process.PID] = process
		if _, ok := processes[process.PPID]; ok {
			processes[process.PPID].Children = append(processes[process.PPID].Children, process)
		} else {
			unhandled[process.PID] = process
		}
		err = statFile.Close()
		if err != nil {
			return nil, err
		}
	}
	fmt.Println("unhandled processes:", len(unhandled))
	for _, process := range unhandled {
		if _, ok := processes[process.PPID]; ok {
			processes[process.PPID].Children = append(processes[process.PPID].Children, process)
		} else {
			// 孤儿进程
			processes[0] = process
			orphanProcessCount += 1
		}
	}
	fmt.Println("orphanProcessCount:", orphanProcessCount)
	// 4. 返回 map[pid]*Process
	return processes, nil
}

// ParseStat 解析进程stat
func ParseStat(stat []byte) (*Process, error) {
	fields := strings.Fields(string(stat))
	pid, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		return nil, err
	}
	ppid, err := strconv.ParseInt(fields[3], 10, 64)
	if err != nil {
		return nil, err
	}
	name := strings.TrimLeft(fields[1], "(")
	name = strings.TrimRight(name, ")")
	return &Process{
		PID:  pid,
		PPID: ppid,
		Name: name,
	}, nil
}

// BuildTree 构建进程树，剪枝
func BuildTree(processes map[int64]*Process) *Process {
	// 提示：
	//vis := make(map[int64]bool)
	//root := processes[0]
	//vis[0]= true
	// 1. 遍历所有进程
	// 2. 将每个进程添加到其父进程的 Children 列表
	// 3. 找到所有根进程（PPID 为 0 或父进程不存在）
	// 4. 返回根进程列表
	return processes[0]
}

func SortByPid(processes []*Process) []*Process {
	sort.SliceIsSorted(processes, func(i, j int) bool {
		return processes[i].PID < processes[j].PID
	})
	return processes
}

// PrintTree 打印进程树，DFS
func PrintTree(root *Process, prefix int, symbolList *Deque, showPid, isSort, isFront, isTreeStart, isLeafLast bool) {
	// 提示：
	// 1. 遍历根进程列表
	// 2. 打印当前进程（使用 prefix 控制缩进）
	// 3. 递归打印子进程
	// 4. 使用树形字符：─ ├ └ │
	// 5. 如果 showPid 为 true，显示 PID
	prefixSpace := 0
	newPrefix := 0
	var text string
	if showPid {
		text = fmt.Sprintf("%s(%d)", root.Name, root.PID)
	} else {
		text = fmt.Sprintf("%s", root.Name)
	}
	prefixSpace = len(text)
	if isFront {
		if isTreeStart {
			fmt.Printf("%s%s", strings.Repeat(" ", 0), text)
		} else {
			fmt.Printf("%s%s", strings.Repeat("─", 4), text)
		}
		newPrefix = prefix + prefixSpace + 4
	} else {
		fmt.Println("")
		//fmt.Println(symbolList.m)
		for i := 0; i < prefix; i++ {
			if i == symbolList.GetMax()-2 {
				if isLeafLast {
					fmt.Printf("└")
				} else {
					fmt.Printf("├")
				}
			} else if i > symbolList.GetMax()-2 {
				fmt.Printf("─")
			} else if node := symbolList.Get(i + 2); node.count > 0 && node.isLast == false {
				fmt.Printf("│")
			} else {
				fmt.Printf(" ")
			}
		}
		fmt.Printf("%s", text)
		row += 1
		newPrefix = prefix + prefixSpace + 4
	}
	if isSort {
		root.Children = SortByPid(root.Children)
	}
	for i, child := range root.Children {
		isLast := i == len(root.Children)-1
		symbolList.PushBack(newPrefix, isLast)
		PrintTree(child, newPrefix, symbolList, showPid, isSort, i == 0, false, isLast)
		symbolList.PopBack()
	}
}
