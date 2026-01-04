package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
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
	mapFile := flag.String("map", "/Users/qiye/home/2025/github/GolandProjects/os-2025/M1/labyrinth/maps/map.txt", "Map file path")
	mapFileShort := flag.String("m", "", "Map file path (short)")
	playerID := flag.String("player", "1", "Player ID (0-9)")
	playerIDShort := flag.String("p", "", "Player ID (short)")
	moveDir := flag.String("move", "up", "Move direction (up/down/left/right)")
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

	// 1. 验证参数
	if *mapFile == "" || *playerID == "" || len(*playerID) != 1 || *moveDir == "" {
		printUsage()
		os.Exit(1)
	}
	if !IsValidPlayer(*playerID) {
		printUsage()
		os.Exit(1)
	}
	// 2. 加载地图
	labyrinth := &Labyrinth{}
	err := LoadMap(labyrinth, *mapFile)
	if err != nil {
		fmt.Println("Error loading map:", err)
		os.Exit(1)
	}
	err = IsConnected(labyrinth)
	if err != nil {
		fmt.Println("Error checking connectivity:", err)
		os.Exit(1)
	}
	// 3. 处理玩家查询或移动
	playerid := rune((*playerID)[0])
	postion, err := FindPlayer(labyrinth, playerid)
	if err != nil {
		fmt.Println("Error finding player:", err)
		os.Exit(1)
	}
	fmt.Printf("Player found at (%d, %d)", postion.Row, postion.Col)
	err = MovePlayer(labyrinth, playerid, *moveDir)
	if err != nil {
		fmt.Println("Error moving player:", err)
		os.Exit(1)
	}
	postion, err = FindPlayer(labyrinth, playerid)
	if err != nil {
		fmt.Println("Error finding player:", err)
		os.Exit(1)
	}
	fmt.Printf("player new position at (%d,%d)\n", postion.Row, postion.Col)
	// 4. 保存地图（如果有移动）
	err = SaveMap(labyrinth, *mapFile)
	if err != nil {
		fmt.Println("Error saving map:", err)
		os.Exit(1)
	}
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
func IsValidPlayer(playerID string) bool {
	// 提示：玩家ID应该是 '0' 到 '9' 之间的字符
	num, _ := strconv.Atoi(playerID)
	if 0 <= num && num <= 9 {
		return true
	}
	return false
}

// LoadMap 从文件加载地图
func LoadMap(labyrinth *Labyrinth, filename string) error {
	// 提示：
	// 1. 打开文件
	// 2. 逐行读取
	lines, err := readFile(filename)
	if err != nil {
		return err
	}
	// 3. 将每行转换为 rune 切片
	runes := make([][]rune, len(lines))
	for i, line := range lines {
		runes[i] = []rune(line)
	}
	// 4. 更新 labyrinth.Map, labyrinth.Rows, labyrinth.Cols
	labyrinth.Map = runes
	labyrinth.Rows = len(runes)
	labyrinth.Cols = len(runes[0])
	return nil
}

// FindPlayer 在地图中查找指定玩家的位置
func FindPlayer(labyrinth *Labyrinth, playerID rune) (*Position, error) {
	// 提示：遍历地图，找到与 playerID 匹配的位置
	// 如果找不到，返回 Position{-1, -1}
	for i := 0; i < labyrinth.Rows; i++ {
		for j := 0; j < labyrinth.Cols; j++ {
			if labyrinth.Map[i][j] == playerID {
				return &Position{i, j}, nil
			}
		}
	}

	position, err := FindFirstEmptySpace(labyrinth)
	if err != nil {
		return nil, err
	}
	return position, nil
}

// FindFirstEmptySpace 找到地图中第一个空位置
func FindFirstEmptySpace(labyrinth *Labyrinth) (*Position, error) {
	// 提示：遍历地图，找到第一个 '.' 字符
	for i := 0; i < labyrinth.Rows; i++ {
		for j := 0; j < labyrinth.Cols; j++ {
			if labyrinth.Map[i][j] == '.' {
				return &Position{i, j}, nil
			}
		}
	}
	return nil, errors.New("player not found")

}

// IsEmptySpace 检查指定位置是否为空
func IsEmptySpace(labyrinth *Labyrinth, row, col int) bool {
	// 提示：
	// 1. 检查边界
	// 2. 检查该位置是否为 '.'
	if row >= 0 && row < labyrinth.Rows && col >= 0 && col < labyrinth.Cols && labyrinth.Map[row][col] == '.' {
		return true
	}
	return false
}

// MovePlayer 移动玩家到指定方向
func MovePlayer(labyrinth *Labyrinth, playerID rune, direction string) error {
	// 提示：
	// 1. 找到玩家当前位置
	// 2. 根据方向计算新位置
	// 3. 检查新位置是否有效（在边界内且为空）
	// 4. 移动玩家（更新地图）
	// 5. 检查移动后的地图连通性
	// 方向：up, down, left, right
	p, err := FindPlayer(labyrinth, playerID)
	if err != nil {
		return err
	}
	row, col := p.Row, p.Col
	var newPosition Position
	switch direction {
	case "up":
		newPosition = Position{row - 1, col}
	case "down":
		newPosition = Position{row + 1, col}
	case "left":
		newPosition = Position{row, col - 1}
	case "right":
		newPosition = Position{row, col + 1}
	}
	if !IsEmptySpace(labyrinth, newPosition.Row, newPosition.Col) {
		return errors.New("invalid move")
	}
	labyrinth.Map[row][col] = '.'
	labyrinth.Map[newPosition.Row][newPosition.Col] = playerID
	return nil
}

// SaveMap 保存地图到文件
func SaveMap(labyrinth *Labyrinth, filename string) error {
	// 提示：
	// 1. 创建或覆盖文件
	// 2. 逐行写入地图内容
	lines := make([]string, labyrinth.Rows)
	for i := 0; i < labyrinth.Rows; i++ {
		lines[i] = string(labyrinth.Map[i])
	}
	err := writeFile(filename, lines)
	if err != nil {
		return err
	}
	return nil
}

// DFS 深度优先搜索，用于检查连通性
func DFS(labyrinth *Labyrinth, row, col int, visited [][]bool) {
	// 提示：
	// 1. 检查边界和访问状态
	// 2. 标记当前位置为已访问
	// 3. 递归访问四个方向的邻居
	if row < 0 || row >= labyrinth.Rows || col < 0 || col >= labyrinth.Cols || visited[row][col] || labyrinth.Map[row][col] == '#' {
		return
	}
	visited[row][col] = true
	DFS(labyrinth, row-1, col, visited)
	DFS(labyrinth, row+1, col, visited)
	DFS(labyrinth, row, col-1, visited)
	DFS(labyrinth, row, col+1, visited)
	return
}

// IsConnected 检查所有空位置是否连通
func IsConnected(labyrinth *Labyrinth) error {
	// 提示：
	// 1. 找到第一个空位置作为起点
	// 2. 使用 DFS 标记所有可达的空位置
	// 3. 检查是否所有空位置都被访问过
	position, err := FindFirstEmptySpace(labyrinth)
	if err != nil {
		return err
	}
	visited := make([][]bool, labyrinth.Rows)
	for i := 0; i < labyrinth.Rows; i++ {
		visited[i] = make([]bool, labyrinth.Cols)
	}
	DFS(labyrinth, position.Row, position.Col, visited)
	for i := 0; i < labyrinth.Rows; i++ {
		for j := 0; j < labyrinth.Cols; j++ {
			if labyrinth.Map[i][j] == '.' && !visited[i][j] {
				return errors.New("map is not connected")
			}
		}
	}
	return nil
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
