# DHT-2022 Report

### Chord

#### 文件结构

- function.go : 定义一些工具函数
- network.go : 包装rpc实现方便的远程调用
- node.go : chord算法的主体部分
- wrapNode.go : 对chord结点进行封装，使函数符合go语言远程rpc调用的规范

#### 算法架构

主要的类：

```go
type ChordNode struct{}
//包含结点的IP,ID,data，以及运行时需要维护的一些内容，此外包含一个network类型的指针，方便进行网络间通信

type network struct{}
//包含一个指向包装结点的指针，和服务器，监听器，方便远程调用

type WrapNode struct{}
//起到包装作用，使函数调用符合规范
```

### Kademlia

#### 文件结构

- toolfunction.go ： 定义一些工具函数
- network.go :  包装rpc相关，方便rpc的远程调用
- node.go : kademlia主体逻辑部分
- wrapNode.go ：对结点进行包装，作用同chord

### 算法架构

主要的类：

```go
type KadNode struct {}
//包含一些结点的信息，现对于chord对节点的信息进行了二次包装

type network struct {}
//同chord，对网络进行包装

type wrapNode struct {}
//同样起到包装作用
```

### Application

Bittorrent主要功能：

- 将本地文件上传到网络，并在目的路径生成一个.torrent种子文件
- 通过本地的.torrent获取网络中的资源，并且将文件下载到目的路径

### Reference

##### go语言学习

[Go 语言教程 | 菜鸟教程 (runoob.com)](https://www.runoob.com/go/go-tutorial.html)

[A Tour of Go](https://go.dev/tour/welcome/1)

[Introduction · Go RPC编程指南 (studygolang.com)](https://books.studygolang.com/go-rpc-programming-guide/)

[golang中的rpc包用法 - andyidea - 博客园 (cnblogs.com)](https://www.cnblogs.com/andyidea/p/6525714.html)

##### 算法学习

[分布式哈希表 (DHT) 和 P2P 技术 - Luyu Huang's Tech Blog](https://luyuhuang.tech/2020/03/06/dht-and-p2p.html)

[Kademlia、DHT、KRPC、BitTorrent 协议、DHT Sniffer - 郑瀚Andrew.Hann - 博客园 (cnblogs.com)](https://www.cnblogs.com/LittleHann/p/6180296.html)

chord与kademlia论文

##### Bittorrent

[用GO从零建立BitTorrent客户端 – HaoranDeng's blog – 会写点代码，会写点小说 (mynameisdhr.com)](https://blog.mynameisdhr.com/YongGOCongLingJianLiBitTorrentKeHuDuan/)

<https://github.com/jackpal/bencode-go>的bencode编码解码包

> 此外，我还参考学习了赵一龙学长、夏天学长和林超凡学长的代码仓库，十分感谢三位学长！