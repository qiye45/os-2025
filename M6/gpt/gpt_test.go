package main

import (
	"os"
	"strings"
	"testing"
)

func TestInference(t *testing.T) {
	// 测试输入tokens
	testTokens := []string{"31373", "612", "338", "635", "281"}

	// 设置命令行参数
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	args := []string{"gpt"}
	args = append(args, testTokens...)
	os.Args = args

	// 捕获输出
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	done := make(chan bool)
	var outputStr string

	// 在单独的goroutine中读取输出
	go func() {
		buf := make([]byte, 4096)
		n, _ := r.Read(buf)
		outputStr = string(buf[:n])
		done <- true
	}()

	// 运行main函数
	main()
	w.Close()

	// 等待读取完成
	<-done
	os.Stdout = oldStdout

	// 检查是否有输出
	if outputStr == "" {
		t.Error("Program produced no output")
	}

	// 检查是否有足够的输出行
	lines := strings.Split(strings.TrimSpace(outputStr), "\n")
	if len(lines) < 1 {
		t.Errorf("Expected at least one line of output, got: %s", outputStr)
	}
}

// 辅助函数：测试模型加载
func TestModelLoading(t *testing.T) {
	var model GPT2

	// 测试加载不存在的文件
	err := gpt2BuildFromCheckpoint(&model, "nonexistent.bin")
	if err == nil {
		t.Error("Expected error when loading nonexistent file")
	}

	// 注意：实际的模型文件测试需要gpt2_124M.bin存在
	// 这个测试在CI环境中可能跳过
	if _, err := os.Stat("gpt2_124M.bin"); os.IsNotExist(err) {
		t.Skip("Model file gpt2_124M.bin not found, skipping model loading test")
	}

	err = gpt2BuildFromCheckpoint(&model, "gpt2_124M.bin")
	if err != nil {
		t.Errorf("Failed to load model: %v", err)
	}

	// 检查配置是否正确加载
	if model.config.vocabSize == 0 {
		t.Error("Model vocab size not loaded correctly")
	}
	if model.config.channels == 0 {
		t.Error("Model channels not loaded correctly")
	}
	if model.config.numLayers == 0 {
		t.Error("Model num layers not loaded correctly")
	}
}

// 辅助函数：测试采样函数
func TestSampleMult(t *testing.T) {
	// 测试简单的概率分布
	probs := []float32{0.1, 0.2, 0.3, 0.4}
	result := sampleMult(probs, len(probs))

	if result < 0 || result >= len(probs) {
		t.Errorf("SampleMult returned invalid result: %d", result)
	}

	// 测试边界情况
	probsZero := []float32{0.0, 0.0, 1.0}
	resultZero := sampleMult(probsZero, len(probsZero))

	if resultZero != 2 {
		t.Errorf("Expected sampleMult to return 2 for deterministic case, got %d", resultZero)
	}
}

// 性能基准测试
func BenchmarkInference(b *testing.B) {
	if _, err := os.Stat("gpt2_124M.bin"); os.IsNotExist(err) {
		b.Skip("Model file gpt2_124M.bin not found, skipping benchmark")
	}

	// 设置命令行参数
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	testTokens := []string{"31373", "612"}
	args := []string{"gpt"}
	args = append(args, testTokens...)
	os.Args = args

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 重定向输出到/dev/null以避免I/O影响基准测试
		oldStdout := os.Stdout
		devNull, _ := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		os.Stdout = devNull

		// 运行main函数（同步执行，不使用goroutine）
		main()

		// 恢复输出并关闭devNull
		devNull.Close()
		os.Stdout = oldStdout
	}
}
