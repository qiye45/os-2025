# M7: HTTP Daemon (httpd)

## 实验背景

HTTP (Hypertext Transfer Protocol) 是互联网上最广泛使用的应用层协议之一。本实验要求实现一个支持多线程的 HTTP 服务器，能够处理并发的 HTTP 请求并执行 CGI 程序。

## 实验要求

### 基本功能

实现一个 HTTP 服务器 `httpd [port]`，监听指定端口（默认 8080），支持以下功能：

1. **CGI 程序执行**
   - 处理以 `/cgi-bin/` 开头的 URL，调用 CGI 程序返回动态内容
   - CGI 程序位于 `cgi-bin/` 目录下
   - 通过环境变量传递 `REQUEST_METHOD` 和 `QUERY_STRING`
   - 将 CGI 程序的标准输出作为 HTTP 响应返回给客户端

2. **错误处理**
   - 路径非法或 CGI 脚本不存在：返回 404 错误
   - CGI 脚本执行失败：返回 500 错误

3. **并发处理**
   - 并发到达的请求应当尽快接收并尽可能并行处理
   - 并行执行的请求数不得超过 4 个
   - CGI 脚本执行时间可能很长，需要正确处理

4. **日志输出**
   - 按照请求到来的顺序输出日志到标准输出
   - 日志格式：`[timestamp] [method] [path] [status_code]`
   - 每次日志输出后执行 `fflush` 确保立即输出

### HTTP 协议说明

**HTTP Request 格式：**
```
GET /cgi-bin/echo?name=world HTTP/1.1
Host: localhost:8080
User-Agent: curl/7.64.1
```

**HTTP Response 格式：**
```
HTTP/1.1 200 OK
Content-Type: text/html
Content-Length: 123
Connection: close

<response body>
```

### CGI 协议说明

CGI (Common Gateway Interface) 是 Web 服务器与外部程序通信的标准协议：

1. 服务器创建子进程执行 CGI 程序
2. 通过环境变量传递请求信息：
   - `REQUEST_METHOD`: 请求方法（如 GET、POST）
   - `QUERY_STRING`: URL 中的查询字符串（如 `name=world`）
3. CGI 程序输出完整的 HTTP Response（包括状态行、响应头和响应体）
4. 服务器将 CGI 程序的输出发送给客户端

**示例：**
请求 `GET /cgi-bin/echo?name=world` 时：
- 执行 `cgi-bin/echo` 程序
- 设置环境变量：`REQUEST_METHOD=GET`, `QUERY_STRING=name=world`
- CGI 程序输出 HTTP 响应，服务器转发给客户端

## 使用示例

```bash
# 启动服务器（默认端口 8080）
./httpd

# 启动服务器（指定端口）
./httpd 9000

# 测试 CGI 程序
curl http://localhost:8080/cgi-bin/echo?name=world
```

## 测试说明

### Easy Test Cases
- 正确解析 URL
- 正确传递环境变量 `REQUEST_METHOD` 和 `QUERY_STRING`
- 正确返回 CGI 脚本的输出
- 正确处理 404 和 500 错误

### Hard Test Cases
- 并发请求处理
- 按请求到达顺序输出日志
- 日志包含正确的 HTTP 状态码（如 200, 404, 500）
- 最多 4 个请求并行执行