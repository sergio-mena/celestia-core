package mempool

import (
	"container/list"

	tmsync "github.com/tendermint/tendermint/libs/sync"
	"github.com/tendermint/tendermint/types"
)

// TxCache defines an interface for raw transaction caching in a mempool.
// Currently, a TxCache does not allow direct reading or getting of transaction
// values. A TxCache is used primarily to push transactions and removing
// transactions. Pushing via Push returns a boolean telling the caller if the
// transaction already exists in the cache or not.
type TxCache interface {
	// Reset resets the cache to an empty state.
	Reset()

	// Push adds the txKey to the cache and returns true if it was
	// newly added. Otherwise, it returns false.
	Push(txKey types.TxKey) bool

	// RemoveTxByKey removes a given transaction hash from the cache.
	RemoveTxByKey(key types.TxKey)

	// Has reports whether tx is present in the cache. Checking for presence is
	// not treated as an access of the value.
	Has(tx types.Tx) bool
}

var _ TxCache = (*LRUTxCache)(nil)

// LRUTxCache maintains a thread-safe LRU cache of raw transactions. The cache
// only stores the hash of the raw transaction.
type LRUTxCache struct {
	mtx      tmsync.Mutex
	size     int
	cacheMap map[types.TxKey]*list.Element
	list     *list.List
}

func NewLRUTxCache(cacheSize int) *LRUTxCache {
	return &LRUTxCache{
		size:     cacheSize,
		cacheMap: make(map[types.TxKey]*list.Element, cacheSize),
		list:     list.New(),
	}
}

// GetList returns the underlying linked-list that backs the LRU cache. Note,
// this should be used for testing purposes only!
func (c *LRUTxCache) GetList() *list.List {
	return c.list
}

func (c *LRUTxCache) Reset() {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	c.cacheMap = make(map[types.TxKey]*list.Element, c.size)
	c.list.Init()
}

func (c *LRUTxCache) Push(txKey types.TxKey) bool {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	moved, ok := c.cacheMap[txKey]
	if ok {
		c.list.MoveToBack(moved)
		return false
	}

	if c.list.Len() >= c.size {
		front := c.list.Front()
		if front != nil {
			frontKey := front.Value.(types.TxKey)
			delete(c.cacheMap, frontKey)
			c.list.Remove(front)
		}
	}

	e := c.list.PushBack(txKey)
	c.cacheMap[txKey] = e

	return true
}

func (c *LRUTxCache) Remove(tx types.Tx) {
	key := tx.Key()
	c.RemoveTxByKey(key)
}

func (c *LRUTxCache) RemoveTxByKey(key types.TxKey) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	e := c.cacheMap[key]
	delete(c.cacheMap, key)

	if e != nil {
		c.list.Remove(e)
	}
}

func (c *LRUTxCache) Has(tx types.Tx) bool {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	_, ok := c.cacheMap[tx.Key()]
	return ok
}

// NopTxCache defines a no-op raw transaction cache.
type NopTxCache struct{}

var _ TxCache = (*NopTxCache)(nil)

func (NopTxCache) Reset()                        {}
func (NopTxCache) Push(types.TxKey) bool         { return true }
func (NopTxCache) Remove(types.Tx)               {}
func (NopTxCache) RemoveTxByKey(key types.TxKey) {}
func (NopTxCache) Has(types.Tx) bool             { return false }