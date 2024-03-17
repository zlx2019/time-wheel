package timewheel

// Lua 执行脚本.

// zAddTaskScript 添加定时任务 lua 脚本指令
// 将定时任务添加到ZSet结构，定时任务的执行时间戳(秒)作为score值.
const zAddTaskScript = `
	-- 获取第一个参数名称, 也就是任务要添加的那个ZSet结构的Key
	local zSetKey = KEYS[1]
	-- 获取第二个参数名称, 也就是该任务对应的删除后要放入的Set结构的Key
	local delZSetKey = KEYS[2]
	-- 获取第一个参数的值, 也就是该定时任务在ZSet中的score
	local score = ARGV[1]
	-- 获取第二个参数值, 也就是该定时任务的具体存储内容
	local task = ARGV[2]
	-- 获取第三个参数值，该值为定时任务的唯一标识，用于从删除列表中移除
	local taskKey = ARGV[3]
	
	-- 可能该任务之前就添加过，并且删除了，所以最好在添加之前，就在标识删除的Set集合中删除一次更为保险.
	redis.call('srem', delZSetKey, taskKey)
	-- 将定时任务添加到Key为 'zSetKey' 的ZSet结构中
	return redis.call('zadd', zSetKey, score, task)
`

// removeTaskScript 删除定时任务的 lua 脚本指令
// 将要删除的定时任务标识，添加到一个Set集合中，表示该任务已被删除，不必再次执行，并且返回该集合的元素数量.
const removeTaskScript = `
	-- 获取 set 的key
	local delSetKey = KEYS[1]
	-- 获取任务的唯一标识
	local taskKey = ARGV[1]
	-- 将 taskKey 添加到 set集合中
	redis.call('sadd', delSetKey, taskKey)
	-- 如果是集合中的第一个元素，就给该Set集合设置一个过期时间
	local sCount = redis.call('scard', delSetKey)
	if (tonumber(sCount) == 1) then
		redis.call('expire', delSetKey, 120)
	end
	return sCount
`

// getTaskScript 获取当前时间对应的 定时任务列表，以及标识已经删除的任务列表
const getTaskScript = `
	local tasksZSetKey = KEYS[1] -- 要读取的 ZSet 定时任务集合的Key
	local taskDelSetKey = KEYS[2] -- 要读取的 Set 标识删除集合的Key
	local scoreStart = ARGV[1]	-- 要检索的score起始大小
	local scoreEnd = ARGV[2]	-- 要检索的score截止大小
	-- 通过 zrange 搜索，获取到满足时间戳条件的定时任务
	local tasks = redis.call('zrange', tasksZSetKey, scoreStart, scoreEnd, 'byscore')
	-- 将本次获取到的定时任务，从 ZSet 任务集合中移除掉，保证分布式情况下不会重复调度
	redis.call('zremrangebyscore', tasksZSetKey, scoreStart, scoreEnd)
	-- 获取任务删除标识集合，筛选出已经被取消的定时任务
	local delTasks = redis.call('smembers', taskDelSetKey)
	-- 将返回结果封装为一个 table
	local reply = {}
	-- 删除标识集合，设置到响应结果的首位元素
	reply[1] = delTasks
	-- 遍历检索到的定时任务，分别追加到响应结果中
	for index, item in ipairs(tasks) do
		reply[#reply+1] = item
	end
	return reply
`
