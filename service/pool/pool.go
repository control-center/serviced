package pool

// A resource pool instance
type Instance struct {
	Key         string
	Name        string
	MemoryLimit uint64
	CoreLimit   int
	Priority    int
	Hosts       []host.Instance
	SubPools    []Instance
}
