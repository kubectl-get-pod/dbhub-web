package connector

import (
	"database/sql"
	"fmt"
	"sync"
)

// PluginRegistry 管理所有数据库插件
type PluginRegistry struct {
	mu      sync.RWMutex
	plugins map[string]DBPlugin
}

var GlobalRegistry = &PluginRegistry{
	plugins: make(map[string]DBPlugin),
}

// Register 注册一个数据库插件
func (r *PluginRegistry) Register(name string, plugin DBPlugin) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.plugins[name] = plugin
}

// Get 获取插件
func (r *PluginRegistry) Get(name string) (DBPlugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.plugins[name]
	if !ok {
		return nil, fmt.Errorf("connector: 未知的数据库类型: %s", name)
	}
	return p, nil
}

// List 列出所有已注册的插件名称
func (r *PluginRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var names []string
	for name := range r.plugins {
		names = append(names, name)
	}
	return names
}

// Pool 管理活动数据库连接
type Pool struct {
	mu     sync.RWMutex
	conns  map[string]*sql.DB // key = connection ID
	config map[string]*ConnectionConfig
}

var GlobalPool = &Pool{
	conns:  make(map[string]*sql.DB),
	config: make(map[string]*ConnectionConfig),
}

// Open 建立数据库连接
func (p *Pool) Open(id string, db *sql.DB, cfg *ConnectionConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.conns[id] = db
	p.config[id] = cfg
}

// Get 获取数据库连接
func (p *Pool) Get(id string) (*sql.DB, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	db, ok := p.conns[id]
	if !ok {
		return nil, fmt.Errorf("连接 %s 未打开", id)
	}
	return db, nil
}

// GetConfig 获取连接配置
func (p *Pool) GetConfig(id string) (*ConnectionConfig, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	cfg, ok := p.config[id]
	if !ok {
		return nil, fmt.Errorf("连接 %s 未找到", id)
	}
	return cfg, nil
}

// Close 关闭并移除连接
func (p *Pool) Close(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	db, ok := p.conns[id]
	if ok {
		delete(p.conns, id)
		delete(p.config, id)
		return db.Close()
	}
	return nil
}

// CloseAll 关闭所有连接
func (p *Pool) CloseAll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for id, db := range p.conns {
		db.Close()
		delete(p.conns, id)
		delete(p.config, id)
	}
}

// IsOpen 检查连接是否已打开
func (p *Pool) IsOpen(id string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	_, ok := p.conns[id]
	return ok
}
