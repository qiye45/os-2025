# M8: 文件系统格式化恢复 (fsrecov)

⏰ **Soft Deadline: 同 Hard Deadline (期末后；无奖励加分)**

## 1. 背景

我们都知道移动存储是不太可靠的，也早就习惯把重要的数据保存在云端；例如，做一个 `~/iCloud` 的符号链接，随时可以在命令行里访问云盘。但这并不意味着 FAT 系列的文件系统就 "没有" 了。首先，在小设备上，FAT 的存储是相当紧凑的，加上实现简单，EFI 分区规定使用 FAT 文件系统就是个更通用和兼容的设计。此外，对于大部分文件都不小，而且是不太频繁写入的场景，FAT 的性能其实还不错——例如，数码相机。今天的相机，哪怕有数百 GB 的 SD 卡，依然保持了 exFAT (类似 "FAT64") 文件系统。

对于 FAT 系列的文件系统，格式化只会重置文件系统里的 File Allocation Table，数据块仍然存在，因此原则上我们可以把数据抢救回来！注意到文件系统是磁盘上的数据结构，如果你希望 "消除" 一个数据结构，你就只需要：

```c
root->left = root->right = NULL;
```

数据结构的其他部分也就永久地丢失了，我们完成了一次完美的 "内存泄漏"。当然，因为整个数据结构都被摧毁，你也可以重置内存分配器的状态，这样所有磁盘上的空间就变得可以被分配，磁盘也就 "焕然一新" (被格式化) 了。这解释了为什么 1TB 磁盘的快速格式化瞬间就可以完成。操作系统也提供了多种格式化的选项，包括更安全、也更慢 (更伤存储设备) 覆盖数据的格式化。

## 2. 实验描述

🗒️ **实验要求：从快速格式化的 FAT32 文件系统中恢复图片数据**

实现命令行工具 `fsrecov`，给定一个经过快速格式化 (mkfs.vfat) 的 FAT32 文件系统镜像，镜像格式化前绝大部分文件都是以 BMP 格式存储的，你需要尽可能地从文件系统中恢复出完整的图片文件。

### 2.1 总览

```bash
fsrecov FILE
```

FILE 是一个 FAT-32 文件系统的镜像。每恢复一张图片文件 (完整的文件，包含 BMP 头和所有数据)，调用系统中的 `sha1sum` 命令获得它的校验和，在标准输出中逐行输出图片文件的校验和以及你恢复出的文件名。只有校验和与文件名**都恢复正确且一致**，才被认为正确恢复了一个文件。

### 2.2 文件系统

作为一个 "小实验"，我们对恢复文件的任务作出了一些简化。首先，我们保证我们的文件系统镜像使用如下方法创建的 (主要使用 `mkfs.fat` 工具):

首先，创建一个空文件 (例如在下面的例子中，文件系统镜像的大小是 64 MiB)，例如：

```bash
$ cat /dev/zero | head -c $(( 1024 * 1024 * 64 )) > fs.img
```

得到 fs.img，然后在这个文件上创建 FAT-32 文件系统：

```bash
$ mkfs.fat -v -F 32 -S 512 -s 8 fs.img
mkfs.fat 4.2 (2021-01-31)
WARNING: Number of clusters for 32 bit FAT is less then suggested minimum.
fs.img has 8 heads and 32 sectors per track,
hidden sectors 0x0000;
logical sector size is 512,
using 0xf8 media descriptor, with 131072 sectors;
drive number 0x80;
filesystem has 2 32-bit FATs and 8 sectors per cluster.
FAT size is 128 sectors, and provides 16348 clusters.
There are 32 reserved sectors.
Volume ID is 80121567, no volume label.
```

注意我们使用的选项：`-S 512` 表示 sector 大小是 512, `-s 8` 表示每个 cluster 有 8 个 sectors。这个信息对大家正确编程非常重要——如果你想偷懒，可以假设我们总是用这种方式创建文件系统镜像 (即硬编码这个信息)，但我们更推荐你阅读手册，写出兼容 FAT 标准的 fsrecov。如果你用 `file` 命令，可以查看到镜像已经被正确格式化：

```bash
$ file fs.img
fs.img: DOS/MBR boot sector, code offset 0x58+2, OEM-ID "mkfs.fat", sectors/cluster 8, Media descriptor 0xf8, sectors/track 32, heads 8, sectors 131072 (volumes > 32 MB), FAT (32 bit), sectors/FAT 128, serial number 0x80121567, unlabeled
```

我们会挂载这个镜像 (一个空的文件系统)，并在根目录下创建 `DCIM` 目录。现在仍然有很多相机延续了这个命名习惯。然后我们会在 DCIM 目录中进行很多次如下的文件操作。尽管图片文件分辨率、大小可能不同，但都**保证是真实世界中有意义的图片** (而不是随机数生成器生成的随机数据)：

- 向 DCIM 中复制图片文件 (文件名为大/小写字母和数字、减号、下划线，以 ".bmp" 或 ".BMP" 结尾)
- 删除 DCIM 中的图片文件
- ……
- (反复操作之后，文件系统中可能存在一些碎片化的情况)

操作完成后，我们会 unmount 文件系统镜像，然后再进行一次文件系统的快速格式化，通过使用同样的选项再次调用 `mkfs.fat`：

```bash
$ mkfs.fat -v -F 32 -S 512 -s 8 fs.img
```

此时的 `fs.img` 就是你要恢复的文件系统镜像。此外，你可以假设所有的 BMP 文件，都是使用 Python PIL 库创建的 24-bit 位图：

```bash
$ file 0M15CwG1yP32UPCp.bmp 
0M15CwG1yP32UPCp.bmp: PC bitmap, Windows 3.x format, 364 x 448 x 24
```

### 2.3 输出格式

你的任务是尝试恢复出 DCIM 目录下尽可能多的图片文件。对于每个恢复出的文件，输出一行，第一个字符串是该文件的 SHA1 fingerprint (通过调用系统的 `sha1sum` 命令得到)，然后可以输出一个或多个空格，接下来输出图片的文件名，例如：

```
d60e7d3d2b47d19418af5b0ba52406b86ec6ef83  0M15CwG1yP32UPCp.bmp
1ab8c4f2e61903ae2a00d0820ea0111fac04d9d3  1yh0sw8n6.bmp
1681e23d7b8bb0b36c399c065514bc04badfde79  2Kbg82NaSqPga.bmp
...
```

## 3. 正确性标准

⚠️ **严格按照要求输出**

只有一行同时包含 40 字节的 sha1sum 之后是文件名，这一行才会被 Online Judge 解析。你的输出中可能带有一些调试信息，我们会忽略它 (不要输出太多调试信息，否则会导致 output limit exceeded)。

### 3.1 评测说明

我们会使用不超过 128 MiB 的镜像文件来测试你的文件，时间限制为 10s。

- 超过 10% 的文件名被恢复正确，可以通过所有 easy test cases；
- 超过 50% 的文件名被恢复正确，可以通过一个 hard test case；
- 超过 75% 的文件名和 50% 的图片被恢复正确，可以通过所有 hard test cases。

不必把图片恢复任务想象得太困难——大文件在文件系统中是倾向于连续存储的，就像在下面参考镜像的 FAT 表中看到的那样。此外，Online Judge 会把你的输出作为一个 utf-8 字符串进行读取。因此，如果你输出了非法的字符 (例如不经检查地输出恢复的文件名，但其实并不是合法的文件名)，将有可能导致解码失败。因此，你输出时请只保留文件名中的可打印 ASCII 字符。

☕️ **Time Limit Exceeded?**

你的程序可能无法在时限内恢复出所有的图片；首先，你可以在每恢复出一个图片后打印，并 flush stdout (超时的程序会被终止，但只要恢复的文件名/图片正确即判定为正确)。此外，你还可以使用 fork 创建多个进程并行恢复。你可以利用 Online Judge 服务器上的所有处理器核心：评测是串行的。

### 3.2 参考镜像

我们为大家提供了一个参考文件系统镜像。实际测试的图像来自同一个数据集 (WikiArt)，但我们可能会挑选不同的图片、赋予文件其他的随机名称或改变图像的大小，但所有随机的参数都与我们给出的镜像相同 (例如随机的文件名长度的分布等)。

镜像下载完毕后可以直接在文件系统中挂载 (你可能需要 root 权限)，这个镜像文件就成为了文件系统的一部分：

```bash
$ mount /tmp/fsrecov.img /mnt/
$ tree /mnt/
/mnt
└── DCIM
    ├── 0M15CwG1yP32UPCp.bmp
    ├── 1yh0sw8n6.bmp
    ├── 2Kbg82NaSqPga.bmp
    ...
```

你可以查看其中的图片文件。如果你用二进制工具 (例如我们使用的是 xxd) 查看镜像文件，你能发现正确的 FAT 表，以链表的形式保存了每个图像文件的下一个数据块 (在 FAT 系统中，是 cluster 的编号)：

```
00004000: f8ff ff0f ffff ff0f f8ff ff0f 1720 0000  ............. ..
00004010: 0500 0000 0600 0000 0700 0000 0800 0000  ................
00004020: 0900 0000 0a00 0000 0b00 0000 0c00 0000  ................
00004030: 0d00 0000 0e00 0000 0f00 0000 1000 0000  ................
00004040: 1100 0000 1200 0000 1300 0000 1400 0000  ................
```

接下来，你可以模拟 Online Judge 在测试你的代码前所做的操作：使用 `mkfs.fat` 快速格式化这个磁盘镜像：

```bash
$ mkfs.fat -v -F 32 -S 512 -s 8 fsrecov.img
mkfs.fat 4.1 (2017-01-24)
WARNING: Not enough clusters for a 32 bit FAT!
/tmp/fsrecov.img has 64 heads and 32 sectors per track,
hidden sectors 0x0000;
logical sector size is 512,
using 0xf8 media descriptor, with 131072 sectors;
drive number 0x80;
filesystem has 2 32-bit FATs and 8 sectors per cluster.
FAT size is 128 sectors, and provides 16348 clusters.
There are 32 reserved sectors.
Volume ID is a332d0ad, no volume label.
```

如果你接下来再次挂载这个镜像，将会看到完全空白的目录，仿佛磁盘镜像上的所有文件都被删除了：

```bash
$ tree /mnt/
/mnt/

0 directories, 0 files
```

如果再次查看 `fsrecov.img` 二进制文件，你会发现分区表已经被 "抹除" 了：

```
00004000: f8ff ff0f ffff ff0f f8ff ff0f 0000 0000  ................
00004010: 0000 0000 0000 0000 0000 0000 0000 0000  ................
00004020: 0000 0000 0000 0000 0000 0000 0000 0000  ................
00004030: 0000 0000 0000 0000 0000 0000 0000 0000  ................
00004040: 0000 0000 0000 0000 0000 0000 0000 0000  ................
```

虽然操作系统已经看不到磁盘上的文件了，但如果你仔细地搜索 (使用 "查找" 工具) 一下，还是可以发现一些蛛丝马迹：

```
00025ae0: 4250 0043 0070 002e 0062 000f 0089 6d00  BP.C.p...b....m.
00025af0: 7000 0000 ffff ffff ffff 0000 ffff ffff  p...............
00025b00: 0130 004d 0031 0035 0043 000f 0089 7700  .0.M.1.5.C....w.
00025b10: 4700 3100 7900 5000 3300 0000 3200 5500  G.1.y.P.3...2.U.
00025b20: 304d 3135 4357 7e31 424d 5020 0064 2b5a  0M15CW~1BMP .d+Z
00025b30: ac50 ac50 0000 2b5a ac50 6915 3677 0700  .P.P..+Z.Pi.6w..
```

这好像以某种格式 (FAT32 的 directory entry) 存储了 "`0M15CwG1yP32UPCp.bmp`" 相关的信息。此外，bitmap 图片文件的文件头也被完整地在数据区里保留下来：

```
000fb000: 424d 2ecf 0f00 0000 0000 3600 0000 2800  BM........6...(.
000fb010: 0000 0202 0000 9f02 0000 0100 1800 0000  ................
000fb020: 0000 f8ce 0f00 c40e 0000 c40e 0000 0000  ................
000fb030: 0000 0000 0000 7d74 9986 7ba3 6c61 8888  ......}t..{.la..
000fb040: 7ea4 8076 9d84 7ca2 766d 9469 6187 6a64  ~..v..|.vm.ia.jd
```

你的 `fsrecov` 会被调用，运行在这个格式化后的镜像上，然后预期会得到一定的输出：

```
d60e7d3d2b47d19418af5b0ba52406b86ec6ef83  0M15CwG1yP32UPCp.bmp
...
```

如果你挂载没有被格式化过的 `fsrecov.img`，你可以查看所有图片的 sha1sum，从而检查你正确恢复了哪些图片。

```bash
$ cd /mnt/DCIM && sha1sum *.bmp
d60e7d3d2b47d19418af5b0ba52406b86ec6ef83  0M15CwG1yP32UPCp.bmp
1ab8c4f2e61903ae2a00d0820ea0111fac04d9d3  1yh0sw8n6.bmp
1681e23d7b8bb0b36c399c065514bc04badfde79  2Kbg82NaSqPga.bmp
aabd1ef8a2371dd64fb64fc7f10a0a31047d1023  2pxHTrpI.bmp
...
```