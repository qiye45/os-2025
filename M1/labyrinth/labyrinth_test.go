package main

import (
	"os"
	"testing"
)

// TestIsValidPlayer 测试玩家ID验证
func TestIsValidPlayer(t *testing.T) {
	tests := []struct {
		playerID string
		expected bool
	}{
		{"0", true},
		{"1", true},
		{"5", true},
		{"a", false},
		{"#", false},
		{" ", false},
	}

	for _, tt := range tests {
		result := IsValidPlayer(tt.playerID)
		if result != tt.expected {
			t.Errorf("IsValidPlayer(%s) = %v, expected %v", tt.playerID, result, tt.expected)
		}
	}
}

// TestIsEmptySpace 测试空位置检查
func TestIsEmptySpace(t *testing.T) {
	lab := &Labyrinth{
		Rows: 3,
		Cols: 3,
		Map: [][]rune{
			{'.', '.', '#'},
			{'#', '.', '.'},
			{'.', '.', '.'},
		},
	}

	tests := []struct {
		row      int
		col      int
		expected bool
	}{
		{0, 0, true},   // 空位置
		{0, 2, false},  // 墙
		{-1, 0, false}, // 越界
		{3, 0, false},  // 越界
		{0, -1, false}, // 越界
		{0, 3, false},  // 越界
		{1, 0, false},  // 墙
		{1, 1, true},   // 空位置
	}

	for _, tt := range tests {
		result := IsEmptySpace(lab, tt.row, tt.col)
		if result != tt.expected {
			t.Errorf("IsEmptySpace(%d, %d) = %v, expected %v", tt.row, tt.col, result, tt.expected)
		}
	}
}

// TestFindPlayer 测试查找玩家
func TestFindPlayer(t *testing.T) {
	lab := &Labyrinth{
		Rows: 3,
		Cols: 3,
		Map: [][]rune{
			{'.', '.', '1'},
			{'.', '.', '.'},
			{'.', '.', '.'},
		},
	}

	// 测试找到玩家
	pos, err := FindPlayer(lab, '1')
	if err != nil {
		t.Errorf("FindPlayer('1') error: %v", err)
	}
	if pos.Row != 0 || pos.Col != 2 {
		t.Errorf("FindPlayer('1') = (%d, %d), expected (0, 2)", pos.Row, pos.Col)
	}

	// 测试找不到玩家（应该返回第一个空位置）
	pos, err = FindPlayer(lab, '2')
	if err != nil {
		t.Errorf("FindPlayer('2') error: %v", err)
	}
	if pos.Row != 0 || pos.Col != 0 {
		t.Errorf("FindPlayer('2') = (%d, %d), expected (0, 0)", pos.Row, pos.Col)
	}
}

// TestFindFirstEmptySpace 测试查找第一个空位置
func TestFindFirstEmptySpace(t *testing.T) {
	lab := &Labyrinth{
		Rows: 2,
		Cols: 2,
		Map: [][]rune{
			{'#', '.'},
			{'#', '#'},
		},
	}

	pos, err := FindFirstEmptySpace(lab)
	if err != nil {
		t.Errorf("FindFirstEmptySpace() error: %v", err)
	}
	if pos.Row != 0 || pos.Col != 1 {
		t.Errorf("FindFirstEmptySpace() = (%d, %d), expected (0, 1)", pos.Row, pos.Col)
	}
}

// TestIsConnected 测试地图连通性
func TestIsConnected(t *testing.T) {
	// 测试连通的地图
	connected := &Labyrinth{
		Rows: 3,
		Cols: 3,
		Map: [][]rune{
			{'.', '.', '.'},
			{'.', '#', '.'},
			{'.', '.', '.'},
		},
	}

	if err := IsConnected(connected); err != nil {
		t.Errorf("IsConnected() error for connected maze: %v", err)
	}

	// 测试不连通的地图
	disconnected := &Labyrinth{
		Rows: 3,
		Cols: 3,
		Map: [][]rune{
			{'.', '.', '#'},
			{'#', '#', '#'},
			{'#', '.', '.'},
		},
	}

	if err := IsConnected(disconnected); err == nil {
		t.Error("IsConnected() should return error for disconnected maze")
	}
}

// TestLoadAndSaveMap 测试地图加载和保存
func TestLoadAndSaveMap(t *testing.T) {
	// 创建测试地图文件
	testFile := "test_map.txt"
	testContent := "...\n.#.\n...\n"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	// 测试加载地图
	lab := &Labyrinth{}
	if err := LoadMap(lab, testFile); err != nil {
		t.Errorf("LoadMap() error: %v", err)
		return
	}

	if lab.Rows != 3 || lab.Cols != 3 {
		t.Errorf("LoadMap() rows=%d, cols=%d, expected rows=3, cols=3", lab.Rows, lab.Cols)
	}

	// 测试保存地图
	saveFile := "test_save.txt"
	if err := SaveMap(lab, saveFile); err != nil {
		t.Errorf("SaveMap() error: %v", err)
		return
	}
	defer os.Remove(saveFile)

	// 验证保存的文件
	content, err := os.ReadFile(saveFile)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("SaveMap() content mismatch")
	}
}

// TestMovePlayer 测试玩家移动
func TestMovePlayer(t *testing.T) {
	lab := &Labyrinth{
		Rows: 3,
		Cols: 3,
		Map: [][]rune{
			{'.', '0', '.'},
			{'.', '.', '.'},
			{'.', '.', '.'},
		},
	}

	// 测试向右移动
	if err := MovePlayer(lab, '0', "right"); err != nil {
		t.Errorf("MovePlayer(right) error: %v", err)
	}

	// 验证玩家位置
	pos, err := FindPlayer(lab, '0')
	if err != nil {
		t.Errorf("FindPlayer error: %v", err)
	}
	if pos.Row != 0 || pos.Col != 2 {
		t.Errorf("After moving right, player at (%d, %d), expected (0, 2)", pos.Row, pos.Col)
	}

	// 测试无效移动（移出边界）
	if err := MovePlayer(lab, '0', "right"); err == nil {
		t.Error("MovePlayer(right) should fail when moving out of bounds")
	}
}

// TestDFS 测试深度优先搜索
func TestDFS(t *testing.T) {
	lab := &Labyrinth{
		Rows: 3,
		Cols: 3,
		Map: [][]rune{
			{'.', '.', '#'},
			{'.', '#', '.'},
			{'.', '.', '.'},
		},
	}

	visited := make([][]bool, 3)
	for i := range visited {
		visited[i] = make([]bool, 3)
	}

	DFS(lab, 0, 0, visited)

	// 检查可达的位置
	if !visited[0][0] || !visited[0][1] || !visited[1][0] {
		t.Error("DFS should visit connected empty spaces")
	}

	// 检查不可达的位置
	if visited[0][2] || visited[1][1] {
		t.Error("DFS should not visit walls")
	}
}

// 基准测试
func BenchmarkIsConnected(b *testing.B) {
	lab := &Labyrinth{
		Rows: 10,
		Cols: 10,
		Map:  make([][]rune, 10),
	}

	for i := 0; i < 10; i++ {
		lab.Map[i] = make([]rune, 10)
		for j := 0; j < 10; j++ {
			lab.Map[i][j] = '.'
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsConnected(lab)
	}
}
