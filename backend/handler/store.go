package handler

import (
	"log"
	"sync"

	"dbhub-web/store"
)

var (
	storeMu  sync.RWMutex
	storeMgr *store.StoreManager
)

// initStore 初始化存储（在 router.go 的 RegisterRoutes 中调用）
func initStore(path string) {
	storeMu.Lock()
	defer storeMu.Unlock()
	storeMgr = store.New(path)
}

// GetStore 返回存储管理器（线程安全，保证非 nil）
func GetStore() *store.StoreManager {
	storeMu.RLock()
	mgr := storeMgr
	storeMu.RUnlock()
	if mgr == nil {
		log.Println("[WARN] store 未初始化，使用临时内存存储")
		storeMu.Lock()
		defer storeMu.Unlock()
		if storeMgr == nil {
			storeMgr = store.New("") // 降级：不持久化
		}
		mgr = storeMgr
	}
	return mgr
}
