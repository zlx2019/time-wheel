# time-wheel 时间轮

使用 Go 从0到1实现一个时间轮数据结构。

## 前言

> **时间轮(TimingWheel)** 是一种 **实现延迟功能（定时器）** 的 **巧妙算法**。如果一个系统存在大量的任务调度，时间轮可以高效的利用线程资源来进行批量化调度。把大批量的调度任务全部都绑定时间轮上，通过时间轮进行所有任务的管理，触发以及运行。能够高效地管理各种延时任务，周期任务，通知任务等。

需要注意的是，时间轮调度器的**精度**不是特别高，对于精度要求特别高的调度任务可能不太适合。因为时间轮算法的精度取决于时间段“指针”单元的最小粒度大小。比如时间轮的格子是一秒跳一次，那么调度精度小于一秒的任务就无法被时间轮所调度。

时间轮（TimingWheel）算法应用范围非常广泛，各种操作系统的定时任务调度都有用到，我们熟悉的 Linux Crontab，以及 Java 开发过程中常用的 Dubbo、Netty、Akka、Quartz、ZooKeeper 、Kafka 等，几乎所有和 **时间任务调度** 都采用了时间轮的思想。

![](https://movies-bucket.oss-cn-beijing.aliyuncs.com/%E6%97%B6%E9%97%B4%E8%BD%AE%E7%A4%BA%E6%84%8F%E5%9B%BE.png)



## 单机版

单机版时间轮采用Go标准库提供的链表`list.List` + 定时器`time.Ticker`实现



## 分布式版

分布式时间轮基于Redis存储引擎.

### 设计思想

1. 使用**Redis**的`ZSet`有序集合作为存储结构，由于`ZSet`底层是基于跳表实现的有序结构，无论是查询、新增还是删除，时间复杂度都是O(log n);

2. 将每一笔定时任务的执行时间转换为时间戳作为`ZSet`的`score`排序值，完成定时任务的有序排列;

3. 定时器每秒调度一次，请求Redis，根据当前时间的时间戳获取所属当前时间范围的定时任务列表，然后进行调度执行;

### 技术细节

- **分钟级别时间分片**

  如果将所有的定时任务放在一个`ZSet`结构中，那么会导致该`ZSet`集合数据体积非常大，定时器每秒只需要当前这一秒要调度的任务数据，所以这样存储显然不合理，这种情况也被称之为**大Key**问题。

  为了避免这个问题，我们将所有的定时任务分割开来存储，以执行时间的**分钟**为维度，进行时间片的纵向划分。也就是说根据定时任务的执行时间进行分组存储，属于同一分钟被执行的任务归于与同一个`ZSet`结构。这样也能确保定时器每秒所要检索的数据规模仅为分钟级别的.

  

- **惰性删除**

  当取消一个定时任务时，这里采用惰性删除: 我们不直接将定时任务从对应的`ZSet`集合中删除，而是建立一个与该`ZSet`集合与之对应的`Set`普通集合，将取消的定时任务的**唯一标识**添加到该集合，定时器调度任务时，同时获取这两个集合，通过`Set`集合中的已删除任务标识，来筛选出`ZSet`集合中哪些任务是已经取消的，从而实现定时任务的删除。注意: 需要给`Set`标识删除集合设置过期时间，这里选择设置两分钟.

  

- **原子性(集群环境问题)**

  在使用`Lua`脚本对Redis进行原子性操作时，有一个前提条件: 那就是该脚本涉及到的所有数据都在同一个集群节点上，这样才可以保证原子性。

  当然如果是单机部署的Redis自然不存在这个问题，但是在生产环境中，我们通常都是集群模式，这使得不同的`Key`可能会被分配到不同的节点中，从而导致不再是原子性操作。

  所以在这里我们通过Redis的`hash_tag`标识进行特殊的散列处理，将每组定时任务对应的存储集合`ZSet`和标识删除的`Set`集合，分配在同一个节点上，达到定时器检索任务时的原子性。
