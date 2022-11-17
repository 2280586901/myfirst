package geecache

import (
	"fmt"
	"log"
	"sync"
)

type Group struct {
	name      string              //学生信息，班级信息，大name
	getter    Getter              //回调函数
	mainCache cache               //实际的数据
	peers     PeerPicker          //真实节点
	loader    *singleflight.Group //防止缓存击穿
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	//构建时需要加锁，避免并发问题
	mu.Lock()
	defer mu.Lock()
	//赋值
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
	}
	//更新map
	groups[name] = g
	return g
}

// GetGroup
func GetGroup(name string) *Group {
	//只读锁，不涉及任何变量的写操作
	mu.RLock()
	defer mu.RUnlock()
	g := groups[name]
	return g
}

// getter是一个接口类型，含有ge函数，当我们缓存未命中，需要从数据库拿到数据
type Getter interface {
	Get(key string) ([]byte, error)
}

// 定义一个函数类型，和上面Get一样
type GetterFunc func(key string) ([]byte, error)

// 回调函数，函数类型实现某一个接口，方便传入不同类型的数据作为参数
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

// get方法具体实现
// 请求一个缓存，两种情况，缓存存在获取，不存在去数据库拿
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}
	//获取缓存
	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache] hit")
		return v, nil
	}
	//构建缓存
	return g.load(key)
}

// 使用 PickPeer() 方法选择节点，非本机节点，调用 getFromPeer() 从远程获取。是本机节点或失败，回退到 getLocally()
func (g *Group) load(key string) (value ByteView, err error) {
	//确保了并发场景下针对相同的 key，load 过程只会调用一次。
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				if value, err = g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[GeeCache] Failed to get from peer", err)
			}
		}
		return g.getLocally(key)
	})
	if err == nil {
		return viewi.(ByteView), nil
	}
	return
}

func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	//数据拷贝下来，变为只读模式
	value := ByteView{b: cloneBytes(bytes)}

	g.populateCache(key, value)
	return value, err
}
func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}

// 将实现PeerPicker 接口的 HTTPPool 注入到 Group 中
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

// 使用实现了 PeerGetter 接口的 httpGetter 从访问远程节点，获取缓存值。
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	bytes, err := peer.Get(g.name, key)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: bytes}, nil
}
