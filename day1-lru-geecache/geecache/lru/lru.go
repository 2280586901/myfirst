package lru

import (
	"container/list"
	"fmt"
)

// 声明缓存cache结构体
type Cache struct {
	maxBytes  int64                         //允许使用的最大内存
	nbyBytes  int64                         //当前已使用的最大内存
	ll        *list.List                    //lru双向链表
	cache     map[string]*list.Element      //字典 key为string类型  value为指向链表中的节点
	OnEvicted func(key string, value Value) //
}

// entry 是双向链表节点的数据类型，保存key值的好处在于淘汰尾节点时，需要用key从字典中删除对应的value
type entry struct {
	key   string
	value Value
}

// 返回值所占用的内存大小
type Value interface {
	Len() int
}

// 实例化cahce 的new函数，如初始缓存内存取多少
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// 查询功能，通过key找到对应的节点
func (c *Cache) Get(key string) (value Value, ok bool) {
	//根据map查找对应链表节点
	if ele, ok := c.cache[key]; ok {
		//将链表中的节点 ele 移动到队首
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		//返回当前节点的内存大小
		return kv.value, true
	}
	return
}

// 删除，缓存淘汰，移除最近最少访问的节点
func (c *Cache) RemoveOldest() {
	//取到队尾节点，从链表中删除。
	ele := c.ll.Back()
	if ele != nil {
		//移除尾节点
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		//删除map中的键值
		delete(c.cache, kv.key)
		//更新缓存内存大小
		c.nbyBytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// 新增加或修改
// 与查询区别，查找是查之前有过的key，同样需要判断这个只在缓存中是否存在
func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		//找到节点，直接移动到队首
		fmt.Println("已存在")
		c.ll.MoveToFront(ele)
		//通过类型断言，判断节点类型
		kv := ele.Value.(*entry)
		c.nbyBytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		fmt.Println("新增")
		//这个key为新增的key
		ele := c.ll.PushFront(&entry{key, value})
		//map中加入键值对
		c.cache[key] = ele
		//内存增加
		c.nbyBytes += int64(len(key)) + int64(value.Len())
	}
	//如果当前内存不够用，淘汰队尾节点
	for c.maxBytes != 0 && c.maxBytes < c.nbyBytes {
		c.RemoveOldest()
	}
}

// 获取多少数据,也就是查看链表节点数
func (c *Cache) Len() int {
	return c.ll.Len()
}
