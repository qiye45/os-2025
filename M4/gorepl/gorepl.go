package main

/*
#cgo LDFLAGS: -ldl
#include <stdlib.h>
#include <dlfcn.h>
#include <stdio.h>

// 定义一个函数指针类型，匹配 wrapper 函数的签名：int func();
typedef int (*expr_func)();

// 辅助函数：调用通过 dlsym 找到的函数指针
int call_wrapper(void *f) {
    expr_func func = (expr_func)f;
    return func();
}

// 辅助函数：加载库 (Go string 转 C string 比较麻烦，封装一下)
void* load_library(char* filename) {
    // RTLD_LAZY: 延迟解析
    // RTLD_GLOBAL: 符号对后续加载的库可见 (关键!)
    return dlopen(filename, RTLD_LAZY | RTLD_GLOBAL);
}

// 辅助函数：查找符号
void* find_symbol(void* handle, char* symbol) {
    return dlsym(handle, symbol);
}

// 获取 dlerror
char* get_dlerror() {
    return dlerror();
}
*/
import "C"

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"unsafe"
)

var loadedSoFiles []string    // 保存已加载的 .so 文件
var definedFunctions []string // 保存已加载的 .so 文件

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	counter := 0 // 用于生成唯一的文件名

	fmt.Print(">> ")
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			fmt.Print(">> ")
			continue
		}

		counter++
		handleInput(line, counter)
		fmt.Print(">> ")
	}
}

func handleInput(line string, id int) {
	// 定义临时文件路径
	// srcFile: /tmp/crepl_{id}.c
	// soFile: /tmp/libcrepl_{id}.so
	err := os.MkdirAll("tmp", 0755)
	if err != nil {
		log.Fatal(err)
		return
	}
	srcFile := fmt.Sprintf("tmp/gorepl_%d.c", id)
	soFile := fmt.Sprintf("tmp/libgorepl_%d.so", id)

	// 区分函数定义和表达式
	// 如果以 "int" 开头，则为函数定义
	// 否则为表达式，需要包装成 wrapper 函数
	var isExpr bool
	var content string
	var funcName string
	if strings.HasPrefix(line, "int") {
		if !strings.HasSuffix(line, ";") {
			line += ";"
		}
		content = line
	} else {
		funcName = fmt.Sprintf("wrapper_%d", id)
		// 添加函数声明，解决编译报错
		var declarations string
		for _, decl := range definedFunctions {
			declarations += decl + "\n"
		}
		content = fmt.Sprintf("%sint %s(){ return %s; };", declarations, funcName, line)
		isExpr = true
	}

	// 写入 .c 文件
	// 使用 os.WriteFile 写入源代码
	err = os.WriteFile(srcFile, []byte(content), 0644)
	if err != nil {
		log.Println(err)
		return
	}
	// 调用 GCC 编译
	// 命令: gcc -x c -fPIC -shared -w -o soFile srcFile
	// -fPIC: 生成位置无关代码
	// -shared: 生成共享库
	// -w: 关闭警告
	err = compileToSharedLib(srcFile, soFile)
	if err != nil {
		log.Println(err)
		return
	}

	// 动态加载 .so
	// 使用 C.load_library 加载共享库
	// 注意: 需要使用 C.CString 转换字符串，并用 C.free 释放
	handle, err := loadSharedLib(soFile)
	if err != nil {
		log.Println(err)
		return
	}

	// 如果是表达式，执行并输出结果
	// 使用 C.find_symbol 查找 wrapper 函数
	// 使用 C.call_wrapper 调用函数
	// 输出格式: = {result}
	if isExpr {
		result, err := callExprWrapper(handle, funcName)
		if err != nil {
			log.Println(err)
			return
		}
		fmt.Printf("result = %d\n", result)
	} else {
		// 如果是函数定义，输出 OK.
		fmt.Println("OK.")
		loadedSoFiles = append(loadedSoFiles, soFile)
		header := strings.Split(line, "{")
		definedFunctions = append(definedFunctions, header[0]+";")
	}
}

// compileToSharedLib 编译 C 代码为共享库
// srcFile: 源文件路径
// soFile: 输出的 .so 文件路径
// 返回: error
func compileToSharedLib(srcFile, soFile string) error {
	// 使用 exec.Command 执行 gcc
	// 参数: -x c -fPIC -shared -w -o soFile srcFile
	args := []string{"-x", "c", "-fPIC", "-shared", "-w", "-o", soFile, srcFile}

	// 添加库搜索路径，解决链接错误
	if len(loadedSoFiles) > 0 {
		args = append(args, "-L./tmp")
	}
	// 添加依赖库
	for _, file := range loadedSoFiles {
		libName := strings.TrimPrefix(file, "tmp/lib")
		libName = strings.TrimSuffix(libName, ".so")
		args = append(args, "-l"+libName)
	}

	cmd := exec.Command("gcc", args...)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// loadSharedLib 加载共享库
// soFile: .so 文件路径
// 返回: handle (unsafe.Pointer) 或 error
func loadSharedLib(soFile string) (unsafe.Pointer, error) {
	// 使用 C.load_library 加载
	// 注意内存管理: C.CString 需要 C.free
	cSoFile := C.CString(soFile)
	defer C.free(unsafe.Pointer(cSoFile))

	handle := C.load_library(cSoFile)
	if handle == nil {
		errMsg := C.GoString(C.get_dlerror())
		return nil, fmt.Errorf("dlopen failed: %s", errMsg)
	}
	return handle, nil
}

// callExprWrapper 调用表达式的 wrapper 函数
// handle: dlopen 返回的句柄
// funcName: wrapper 函数名
// 返回: 函数返回值 或 error
func callExprWrapper(handle unsafe.Pointer, funcName string) (int, error) {
	// 使用 C.find_symbol 查找函数
	// 使用 C.call_wrapper 调用函数
	cFuncName := C.CString(funcName)
	defer C.free(unsafe.Pointer(cFuncName))

	sym := C.find_symbol(handle, cFuncName)
	if sym == nil {
		errMsg := C.GoString(C.get_dlerror())
		return 0, fmt.Errorf("dlsym failed: %s", errMsg)
	}

	result := C.call_wrapper(sym)
	return int(result), nil
}
