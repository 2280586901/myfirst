package geecache

import (
	"example/geecache/consistendhash"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

//创建结构体HTTPPool，作为服务端和客户端通信的节点
//basepath

const defaultBasePath = "/_geecache/"

type HTTPPool struct {
	self        string     //端口号和地址
	basepath    string     //
	mu          sync.Mutex //锁
	peers       *consistendhash.Map
	httpGetters map[string]*httpGetter
}

// new
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basepath: defaultBasePath,
	}
}

// 日志输出服务端名字
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// 实现serverHTTP方法
func (p *HTTPPool) ServerHTTP(w http.ResponseWriter, r *http.Request) {
	//判断url地址前缀是否满足base
	if !strings.HasPrefix(r.URL.Path, p.basepath) {
		panic(("HTTPPool serving unexpectd path " + r.URL.Path))
	}
	//打印日志
	p.Log(r.Method, r.URL.Path)
	//分割路径
	parts := strings.SplitN(r.URL.Path[len(p.basepath):], "/", 2)
	//判断path中格式是否正确
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	//获取对应group之后获取对应name
	groupName := parts[0]
	key := parts[1]
	//通过GetGroup获取group
	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}
	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice())

}

// 创建HTTPGetter结构体
type httpGetter struct {
	baseURL string
}

// 实现Get方法
func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	u := fmt.Sprintf("%v%v/%v",
		h.baseURL,
		url.QueryEscape(group),
		url.QueryEscape(key))
	//发送请求，接收消息
	res, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	//关闭response
	defer res.Body.Close()
	//先对有异常情况处理，通过对比http响应的状态码
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sercer returned:%v", res.Status)
	}
	//从res中拿数据
	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %v", err)
	}
	return bytes, nil
}

var _peerGetter = (*HTTPPool)(nil)

// set方法传入节点，实例化hash算法， 为每一个节点创建一个http客户端httpGetter
func (p *HTTPPool) set(peers ...string) {
	//传入节点时上锁
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistendhash.New(defaultReplicas, nil)
	//将每个节点穿入add中
	p.peers.Add(peers...)
	//给每一个节点分配一个http客户端
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		//每一个客户端分配一个ip地址
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basepath}
	}
}

// 通过传入的key选择节点，返回节点对应的http客户端
func (p *HTTPPool) pickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	//获取节点 通过get方法 获取失败get方法返回的是" " 而且要判定此时返回的不是当前节点
	//需要向远处节点获取数据
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}

var _PeerPicker = (*HTTPPool)(nil)
