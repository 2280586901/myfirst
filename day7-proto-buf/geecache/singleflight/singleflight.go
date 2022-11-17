package singleflight

import "sync"

// call代表一次http客户端请求
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

// singleflight的主要结构group 用来管理不同key 的请求
type Group struct {
	mu sync.Mutex
	m  map[string]*call
}

// 核心Do方法
// 这里的group是为了管理不同key请求，不是之前的group
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	//获取在m中的key对应客户端请求，如果获取成功，代表之前已经有一个当前key请求
	if c, ok := g.m[key]; ok {
		//首先解锁
		g.mu.Unlock()
		//进行同步操作，等第一个请求完成
		c.wg.Wait()
		//待第一个请求完成，获取其返回值
		return c.val, c.err
	}
	//接下来这部分处理如果是第一个请求
	c := new(call)
	//add添加一个线程开启提示
	c.wg.Add(1)
	//key放入map中
	g.m[key] = c
	//对map key读写操作需要互斥锁实现，对请求限制是需要同步wait实现
	g.mu.Unlock()
	//调用请求函数获取返回值，使用call对象接收
	c.val, c.err = fn()
	//做完这个线程任务需要通知其他请求
	c.wg.Done()
	//接下来进行对map删除key操作，因为此时请求已经完成，必须删除key
	//防止下一个同一个key来，一位已经有一次key请求，无限等待
	//互斥锁
	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()
	//最后返回请求函数的结果
	return c.val, c.err
}
