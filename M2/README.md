# M2: 打印进程树 (pstree)

## 实验描述

🗒️**实验要求：实现 pstree 打印进程之间的树状的父子关系**

Linux 系统中可以同时运行多个程序。运行的程序称为**进程**。除了所有进程的根之外，每个进程都有它唯一的父进程，你的任务就是把这棵树在命令行中输出。你可以自由选择展示树的方式 (例如使用缩进表示父子关系)。

Linux 系统中有 `pstree` 命令，进程树会以非常漂亮的格式排版 (每个进程的第一个孩子都与它处在同一行，之后的孩子保持相同的缩进)：

```text
systemd─┬─accounts-daemon─┬─{gdbus}
        │                 └─{gmain}
        ├─acpid
        ├─agetty
        ├─atd
        ├─cron
        ├─dbus-daemon
        ├─dhclient
        ├─2*[iscsid]
        ├─lvmetad
        ├─lxcfs───10*[{lxcfs}]
        ├─mdadm
        ├─polkitd─┬─{gdbus}
        │         └─{gmain}
        ├─rsyslogd─┬─{in:imklog}
        │          ├─{in:imuxsock}
        │          └─{rs:main Q:Reg}
        ...
```

## 功能要求

- 读取系统进程信息
- 构建进程树结构
- 以树状格式输出（支持缩进或图形化显示）
- 显示进程名称和PID
- 支持命令行选项


## 使用示例

```bash
cd pstree
go build
./pstree
./pstree -p
./pstree -n
./pstree -p -n
```

