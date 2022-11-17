package geecache

// b是一个只读的数据结构，存储真实的缓存值，选择byte是为了支持任意数据类型的存储，比如字符串，图片
type ByteView struct {
	b []byte
}

// 返回缓存所占的内存大小
func (v ByteView) Len() int {
	return len(v.b)
}

// 返回复制一份数据的拷贝，防止缓存被外部程序修改
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

func (v ByteView) String() string {
	return string(v.b)
}

func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
