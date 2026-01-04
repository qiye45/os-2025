package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
)

const (
	MaxRows     = 100
	MaxCols     = 100
	VersionInfo = "Labyrinth Game"
)

// Labyrinth 迷宫结构
type Labyrinth struct {
	Map  [][]rune
	Rows int
	Cols int
}

// Position 位置结构
type Position struct {
	Row int
	Col int
}

func main() {
	// 定义命令行参数
	mapFile := flag.String("map", "", "Map file path")
	mapFileShort := flag.String("m", "", "Map file path (short)")
	playerID := flag.String("player", "", "Player ID (0-9)")
	playerIDShort := flag.String("p", "", "Player ID (short)")
	moveDir := flag.String("move", "", "Move direction (up/down/left/right)")
	version := flag.Bool("version", false, "Show version information")

	flag.Parse()

	// 处理 --version
	if *version {
		if flag.NArg() > 0 || len(os.Args) > 2 {
			printUsage()
			os.Exit(1)
		}
		fmt.Println(VersionInfo)
		os.Exit(0)
	}

	// 合并短参数和长参数
	if *mapFileShort != "" {
		mapFile = mapFileShort
	}
	if *playerIDShort != "" {
		playerID = playerIDShort
	}

	// 检查未知参数
	if flag.NArg() > 0 {
		printUsage()
		os.Exit(1)
	}

	// TODO: 实现主逻辑
	// 1. 验证参数
	// 2. 加载地图
	// 3. 处理玩家查询或移动
	// 4. 保存地图（如果有移动）

	os.Exit(0)
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  labyrinth --map map.txt --player id")
	fmt.Println("  labyrinth -m map.txt -p id")
	fmt.Println("  labyrinth --map map.txt --player id --move direction")
	fmt.Println("  labyrinth --version")
}

// IsValidPlayer 检查玩家ID是否有效（0-9）
func IsValidPlayer(playerID rune) bool {
	// TODO: 实现此函数
	// 提示：玩家ID应该是 '0' 到 '9' 之间的字符
	return false
}

// LoadMap 从文件加载地图
func LoadMap(labyrinth *Labyrinth, filename string) bool {
	// TODO: 实现此函数
	// 提示：
	// 1. 打开文件
	// 2. 逐行读取
	// 3. 将每行转换为 rune 切片
	// 4. 更新 labyrinth.Map, labyrinth.Rows, labyrinth.Cols
	return false
}

// FindPlayer 在地图中查找指定玩家的位置
func FindPlayer(labyrinth *Labyrinth, playerID rune) Position {
	// TODO: 实现此函数
	// 提示：遍历地图，找到与 playerID 匹配的位置
	// 如果找不到，返回 Position{-1, -1}
	return Position{-1, -1}
}

// FindFirstEmptySpace 找到地图中第一个空位置
func FindFirstEmptySpace(labyrinth *Labyrinth) Position {
	// TODO: 实现此函数
	// 提示：遍历地图，找到第一个 '.' 字符
	return Position{-1, -1}
}

// IsEmptySpace 检查指定位置是否为空
func IsEmptySpace(labyrinth *Labyrinth, row, col int) bool {
	// TODO: 实现此函数
	// 提示：
	// 1. 检查边界
	// 2. 检查该位置是否为 '.'
	return false
}

// MovePlayer 移动玩家到指定方向
func MovePlayer(labyrinth *Labyrinth, playerID rune, direction string) bool {
	// TODO: 实现此函数
	// 提示：
	// 1. 找到玩家当前位置
	// 2. 根据方向计算新位置
	// 3. 检查新位置是否有效（在边界内且为空）
	// 4. 移动玩家（更新地图）
	// 5. 检查移动后的地图连通性
	// 方向：up, down, left, right
	return false
}

// SaveMap 保存地图到文件
func SaveMap(labyrinth *Labyrinth, filename string) bool {
	// TODO: 实现此函数
	// 提示：
	// 1. 创建或覆盖文件
	// 2. 逐行写入地图内容
	return false
}

// DFS 深度优先搜索，用于检查连通性
func DFS(labyrinth *Labyrinth, row, col int, visited [][]bool) {
	// TODO: 实现此函数
	// 提示：
	// 1. 检查边界和访问状态
	// 2. 标记当前位置为已访问
	// 3. 递归访问四个方向的邻居
}

// IsConnected 检查所有空位置是否连通
func IsConnected(labyrinth *Labyrinth) bool {
	// TODO: 实现此函数
	// 提示：
	// 1. 找到第一个空位置作为起点
	// 2. 使用 DFS 标记所有可达的空位置
	// 3. 检查是否所有空位置都被访问过
	return false
}

// 辅助函数：读取文件内容
func readFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

// 辅助函数：写入文件内容
func writeFile(filename string, lines []string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			return err
		}
	}

	return writer.Flush()
}
