package consistendhash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// 定义函数类型hash 用于计算hash值
type Hash func(data []byte) uint32

// map的四个成员变量
type Map struct {
	hash     Hash           //hash函数
	replicas int            //虚拟节点的倍数
	keys     []int          //哈希环
	hashMap  map[int]string //虚拟节点与真实节点的映射
}

// 构建函数 允许自定义虚拟节点倍数和hash函数
func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// 允许一次添加多个节点
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		//加入对应虚拟节点，虚拟节点名称是i+key
		for i := 0; i < m.replicas; i++ {
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			//hash值加入hash环中
			m.keys = append(m.keys, hash)
			//建立映射关系
			m.hashMap[hash] = key
		}
		//排序一下，便于取值
	}
}

// 获取get
// 步骤 1根据key计算hash值
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}
	hash := int(m.hash([]byte(key)))
	//顺时针找到第一个下标
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})
	//如果大于当前hash值的数不存在，代表对应的索引是环开始的第一个节点m.key[0]
	//需要注意取一下余数
	return m.hashMap[m.keys[idx%len(m.keys)]]
}
