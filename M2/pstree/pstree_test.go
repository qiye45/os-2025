package main

import (
	"os/exec"
	"strings"
	"testing"
)

// runPstree 运行 pstree 命令并返回输出和退出码
func runPstree(args ...string) (string, int, error) {
	cmd := exec.Command("./pstree", args...)
	output, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return "", -1, err
		}
	}
	return string(output), exitCode, nil
}

// TestBasicNoArgs 测试基本功能（无参数）
func TestBasicNoArgs(t *testing.T) {
	output, exitCode, err := runPstree()
	if err != nil {
		t.Fatalf("Failed to run pstree: %v", err)
	}

	if exitCode != 0 {
		t.Errorf("Basic pstree command should exit with status 0, got %d", exitCode)
	}

	if len(output) == 0 {
		t.Error("Output should not be empty")
	}
}

// TestShowPidsShort 测试 -p 选项
func TestShowPidsShort(t *testing.T) {
	output, exitCode, err := runPstree("-p")
	if err != nil {
		t.Fatalf("Failed to run pstree -p: %v", err)
	}

	if exitCode != 0 {
		t.Errorf("pstree -p should exit with status 0, got %d", exitCode)
	}

	if len(output) == 0 {
		t.Error("Output should not be empty")
	}

	if !strings.Contains(output, "(") {
		t.Error("Output should contain PIDs in parentheses")
	}
}

// TestShowPidsLong 测试 --show-pids 选项
func TestShowPidsLong(t *testing.T) {
	output, exitCode, err := runPstree("--show-pids")
	if err != nil {
		t.Fatalf("Failed to run pstree --show-pids: %v", err)
	}

	if exitCode != 0 {
		t.Errorf("pstree --show-pids should exit with status 0, got %d", exitCode)
	}

	if len(output) == 0 {
		t.Error("Output should not be empty")
	}

	if !strings.Contains(output, "(") {
		t.Error("Output should contain PIDs in parentheses")
	}
}

// TestNumericSortShort 测试 -n 选项
func TestNumericSortShort(t *testing.T) {
	output, exitCode, err := runPstree("-n")
	if err != nil {
		t.Fatalf("Failed to run pstree -n: %v", err)
	}

	if exitCode != 0 {
		t.Errorf("pstree -n should exit with status 0, got %d", exitCode)
	}

	if len(output) == 0 {
		t.Error("Output should not be empty")
	}
}

// TestNumericSortLong 测试 --numeric-sort 选项
func TestNumericSortLong(t *testing.T) {
	output, exitCode, err := runPstree("--numeric-sort")
	if err != nil {
		t.Fatalf("Failed to run pstree --numeric-sort: %v", err)
	}

	if exitCode != 0 {
		t.Errorf("pstree --numeric-sort should exit with status 0, got %d", exitCode)
	}

	if len(output) == 0 {
		t.Error("Output should not be empty")
	}
}

// TestVersionShort 测试 -V 选项
func TestVersionShort(t *testing.T) {
	output, exitCode, err := runPstree("-V")
	if err != nil {
		t.Fatalf("Failed to run pstree -V: %v", err)
	}

	if exitCode != 0 {
		t.Errorf("pstree -V should exit with status 0, got %d", exitCode)
	}

	if len(output) == 0 {
		t.Error("Version information should not be empty")
	}

	if !strings.Contains(output, "pstree") {
		t.Error("Output should contain version information")
	}
}

// TestVersionLong 测试 --version 选项
func TestVersionLong(t *testing.T) {
	output, exitCode, err := runPstree("--version")
	if err != nil {
		t.Fatalf("Failed to run pstree --version: %v", err)
	}

	if exitCode != 0 {
		t.Errorf("pstree --version should exit with status 0, got %d", exitCode)
	}

	if len(output) == 0 {
		t.Error("Version information should not be empty")
	}

	if !strings.Contains(output, "pstree") {
		t.Error("Output should contain version information")
	}
}

// TestShowPidsAndNumericSort 测试组合选项 -p -n
func TestShowPidsAndNumericSort(t *testing.T) {
	output, exitCode, err := runPstree("-p", "-n")
	if err != nil {
		t.Fatalf("Failed to run pstree -p -n: %v", err)
	}

	if exitCode != 0 {
		t.Errorf("pstree -p -n should exit with status 0, got %d", exitCode)
	}

	if len(output) == 0 {
		t.Error("Output should not be empty")
	}

	if !strings.Contains(output, "(") {
		t.Error("Output should contain PIDs in parentheses")
	}
}

// TestAllOptionsLong 测试组合选项 --show-pids --numeric-sort
func TestAllOptionsLong(t *testing.T) {
	output, exitCode, err := runPstree("--show-pids", "--numeric-sort")
	if err != nil {
		t.Fatalf("Failed to run pstree --show-pids --numeric-sort: %v", err)
	}

	if exitCode != 0 {
		t.Errorf("pstree --show-pids --numeric-sort should exit with status 0, got %d", exitCode)
	}

	if len(output) == 0 {
		t.Error("Output should not be empty")
	}

	if !strings.Contains(output, "(") {
		t.Error("Output should contain PIDs in parentheses")
	}
}

// TestInvalidOption 测试无效选项
func TestInvalidOption(t *testing.T) {
	output, exitCode, err := runPstree("--invalid-option")
	if err != nil {
		t.Fatalf("Failed to run pstree --invalid-option: %v", err)
	}

	if exitCode == 0 {
		t.Error("pstree with invalid option should exit with non-zero status")
	}

	outputLower := strings.ToLower(output)
	if !strings.Contains(outputLower, "usage") && !strings.Contains(outputLower, "invalid") {
		t.Error("Output should mention invalid option or show usage")
	}
}

// 单元测试

// TestBuildTree 测试进程树构建
func TestBuildTree(t *testing.T) {
	processes := map[int]*Process{
		1: {PID: 1, PPID: 0, Name: "init"},
		2: {PID: 2, PPID: 1, Name: "child1"},
		3: {PID: 3, PPID: 1, Name: "child2"},
		4: {PID: 4, PPID: 2, Name: "grandchild"},
	}

	tree := BuildTree(processes)

	if len(tree) != 1 {
		t.Errorf("Expected 1 root process, got %d", len(tree))
	}

	if tree[0].PID != 1 {
		t.Errorf("Expected root PID 1, got %d", tree[0].PID)
	}

	if len(tree[0].Children) != 2 {
		t.Errorf("Expected 2 children for root, got %d", len(tree[0].Children))
	}
}

// TestSortTree 测试进程树排序
func TestSortTree(t *testing.T) {
	processes := map[int]*Process{
		1: {PID: 1, PPID: 0, Name: "init"},
		3: {PID: 3, PPID: 1, Name: "child3"},
		2: {PID: 2, PPID: 1, Name: "child2"},
	}

	tree := BuildTree(processes)
	SortTree(tree)

	if len(tree[0].Children) != 2 {
		t.Errorf("Expected 2 children, got %d", len(tree[0].Children))
	}

	if tree[0].Children[0].PID != 2 {
		t.Errorf("Expected first child PID 2, got %d", tree[0].Children[0].PID)
	}

	if tree[0].Children[1].PID != 3 {
		t.Errorf("Expected second child PID 3, got %d", tree[0].Children[1].PID)
	}
}
