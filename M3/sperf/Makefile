.PHONY: build clean demo-memory demo-filesystem demo-all visualize

# 构建 sperf
build:
	go build -o sperf .

clean:
	rm -f sperf

# 示例1: memory intensive 纯计算型程序 (brk, munmap 耗时最多)
# 使用 sort 对大量数据排序
demo-memory: build
	@echo "=== Memory Intensive 示例: sort 大量数据 ==="
	seq 100000000 | shuf | ./sperf sort -n > /dev/null

# 示例2: 文件系统遍历 (lstat, getdents 耗时最多)
# 使用 find 遍历目录
demo-filesystem: build
	@echo "=== Filesystem 示例: find 遍历目录 ==="
	./sperf find /usr -maxdepth 3 -type f 2>/dev/null | head -100

# 运行所有示例
demo-all: demo-memory demo-filesystem

# 可视化示例
visualize-memory: build
	@echo "=== Memory Intensive 可视化 ==="
	seq 100000000 | shuf | ./sperf sort -n 2>&1 | python3 visualize.py

visualize-filesystem: build
	@echo "=== Filesystem 可视化 ==="
	./sperf find /usr -maxdepth 3 -type f 2>&1 | python3 visualize.py