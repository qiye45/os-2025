.PHONY: build test clean run

# 运行测试
test:
	go test -v

# 运行基准测试
bench:
	go test -bench=BenchmarkInference -benchtime=5x -benchmem

# 性能分析
profile:
	go run gpt.go -cpuprofile=cpu.prof 31373 612 338 635 281
	go tool pprof -http=:8080 cpu.prof