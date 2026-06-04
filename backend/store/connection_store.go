package store

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
	"path/filepath"

	"dbhub-web/connector"
)

const connectionsFile = "connections.json"
const historyFile = "query_history.json"
const favoritesFile = "favorites.json"

// StoreManager 管理持久化数据
type StoreManager struct {
	dataDir string
	aesKey  []byte
}

// New 创建存储管理器
func New(dataDir string) *StoreManager {
	return &StoreManager{
		dataDir: dataDir,
		aesKey:  deriveKey(),
	}
}

// deriveKey 从机器特征派生加密密钥
func deriveKey() []byte {
	hostname, _ := os.Hostname()
	hash := sha256.Sum256([]byte("dbhub-web-v1-" + hostname))
	return hash[:] // 32 bytes = AES-256
}

// --- 连接持久化 ---

// SaveConnections 保存所有连接到文件
func (s *StoreManager) SaveConnections(conns []connector.ConnectionConfig) error {
	encrypted := make([]connector.ConnectionConfig, len(conns))
	for i, c := range conns {
		encrypted[i] = c
		var err error
		encrypted[i].Password, err = s.encrypt(c.Password)
		if err != nil {
			return fmt.Errorf("store: 加密密码失败: %w", err)
		}
		encrypted[i].SSHPass, err = s.encrypt(c.SSHPass)
		if err != nil {
			return fmt.Errorf("store: 加密SSH密码失败: %w", err)
		}
	}
	return s.writeJSON(connectionsFile, encrypted)
}

// LoadConnections 从文件加载所有连接
func (s *StoreManager) LoadConnections() ([]connector.ConnectionConfig, error) {
	var conns []connector.ConnectionConfig
	if err := s.readJSON(connectionsFile, &conns); err != nil {
		if os.IsNotExist(err) {
			return []connector.ConnectionConfig{}, nil
		}
		s.backupCorrupted(connectionsFile)
		return []connector.ConnectionConfig{}, nil
	}
	for i := range conns {
		var err error
		conns[i].Password, err = s.decrypt(conns[i].Password)
		if err != nil {
			return nil, fmt.Errorf("store: 解密密码失败 (连接 %s): %w", conns[i].Name, err)
		}
		conns[i].SSHPass, err = s.decrypt(conns[i].SSHPass)
		if err != nil {
			return nil, fmt.Errorf("store: 解密SSH密码失败 (连接 %s): %w", conns[i].Name, err)
		}
	}
	return conns, nil
}

// --- 查询历史持久化 ---

type QueryHistoryItem struct {
	ID        string `json:"id"`
	SQL       string `json:"sql"`
	ConnName  string `json:"connName"`
	CreatedAt string `json:"createdAt"`
	Duration  string `json:"duration,omitempty"`
}

func (s *StoreManager) SaveHistory(items []QueryHistoryItem) error {
	return s.writeJSON(historyFile, items)
}

func (s *StoreManager) LoadHistory() ([]QueryHistoryItem, error) {
	var items []QueryHistoryItem
	if err := s.readJSON(historyFile, &items); err != nil {
		if os.IsNotExist(err) {
			return []QueryHistoryItem{}, nil
		}
		// 文件损坏时备份并返回空数据
		s.backupCorrupted(historyFile)
		return []QueryHistoryItem{}, nil
	}
	return items, nil
}

// --- 收藏查询持久化 ---

type FavoriteItem struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	SQL       string `json:"sql"`
	ConnType  string `json:"connType"`
	CreatedAt string `json:"createdAt"`
}

func (s *StoreManager) SaveFavorites(items []FavoriteItem) error {
	return s.writeJSON(favoritesFile, items)
}

func (s *StoreManager) LoadFavorites() ([]FavoriteItem, error) {
	var items []FavoriteItem
	if err := s.readJSON(favoritesFile, &items); err != nil {
		if os.IsNotExist(err) {
			return []FavoriteItem{}, nil
		}
		s.backupCorrupted(favoritesFile)
		return []FavoriteItem{}, nil
	}
	return items, nil
}

// --- JSON 文件读写 ---

func (s *StoreManager) writeJSON(filename string, data interface{}) error {
	path := filepath.Join(s.dataDir, filename)
	tmpPath := path + ".tmp"

	file, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("store: 创建临时文件失败: %w", err)
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		file.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("store: JSON编码失败: %w", err)
	}
	file.Close()

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("store: 文件重命名失败: %w", err)
	}
	return nil
}

func (s *StoreManager) readJSON(filename string, dest interface{}) error {
	path := filepath.Join(s.dataDir, filename)
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewDecoder(file).Decode(dest)
}

func (s *StoreManager) backupCorrupted(filename string) {
	src := filepath.Join(s.dataDir, filename)
	dst := filepath.Join(s.dataDir, filename+fmt.Sprintf(".corrupted-%d", time.Now().Unix()))
	if err := os.Rename(src, dst); err != nil {
		data, err := os.ReadFile(src)
		if err != nil {
			return
		}
		os.WriteFile(dst, data, 0644)
	}
}

// --- AES-256-GCM 加密/解密 ---

func (s *StoreManager) encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	block, err := aes.NewCipher(s.aesKey)
	if err != nil {
		return "", fmt.Errorf("创建加密器失败: %w", err)
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建GCM失败: %w", err)
	}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("生成随机数失败: %w", err)
	}
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (s *StoreManager) decrypt(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("base64解码失败: %w", err)
	}
	block, err := aes.NewCipher(s.aesKey)
	if err != nil {
		return "", fmt.Errorf("创建解密器失败: %w", err)
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建GCM失败: %w", err)
	}
	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("密文太短")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("解密失败（密钥可能不匹配）: %w", err)
	}
	return string(plaintext), nil
}
