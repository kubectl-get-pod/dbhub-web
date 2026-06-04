package connector

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"sync"

	"golang.org/x/crypto/ssh"
)

// SSHTunnelResult 包含 SSH 隧道建立后的资源句柄
type SSHTunnelResult struct {
	LocalAddr string          // 本地转发地址 (127.0.0.1:port)
	client    *ssh.Client
	listener  net.Listener
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// Close 关闭 SSH 隧道，释放所有资源
func (t *SSHTunnelResult) Close() error {
	t.cancel()          // 通知 accept 循环退出
	t.listener.Close()  // 解除 Accept 阻塞
	t.wg.Wait()         // 等待 goroutine 退出
	return t.client.Close()
}

// SSHTunnel 通过 SSH 隧道建立 TCP 端口转发
func SSHTunnel(ctx context.Context, cfg *ConnectionConfig) (*SSHTunnelResult, error) {
	if !cfg.UseSSH {
		return nil, fmt.Errorf("SSH 未启用")
	}

	// SSH 认证配置
	authMethods := []ssh.AuthMethod{}
	if cfg.SSHPass != "" {
		authMethods = append(authMethods, ssh.Password(cfg.SSHPass))
	}
	if cfg.SSHKey != "" {
		key, err := os.ReadFile(cfg.SSHKey)
		if err != nil {
			return nil, fmt.Errorf("读取 SSH 私钥失败: %w", err)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("解析 SSH 私钥失败: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}
	if len(authMethods) == 0 {
		return nil, fmt.Errorf("SSH 认证方式为空（需要密码或私钥）")
	}

	sshConfig := &ssh.ClientConfig{
		User:            cfg.SSHUser,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	sshPort := cfg.SSHPort
	if sshPort == 0 {
		sshPort = 22
	}

	sshAddr := fmt.Sprintf("%s:%d", cfg.SSHHost, sshPort)
	client, err := ssh.Dial("tcp", sshAddr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("SSH 连接失败: %w", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("本地端口监听失败: %w", err)
	}

	localAddr := listener.Addr().String()
	dbPort := cfg.Port
	if dbPort == 0 {
		dbPort = defaultPort(cfg.Type)
	}
	remoteAddr := fmt.Sprintf("%s:%d", cfg.Host, dbPort)

	tunnelCtx, cancel := context.WithCancel(ctx)
	result := &SSHTunnelResult{
		LocalAddr: localAddr,
		client:    client,
		listener:  listener,
		ctx:       tunnelCtx,
		cancel:    cancel,
	}

	result.wg.Add(1)
	go func() {
		defer result.wg.Done()
		defer client.Close()
		defer listener.Close()

		for {
			select {
			case <-tunnelCtx.Done():
				return
			default:
			}

			localConn, err := listener.Accept()
			if err != nil {
				select {
				case <-tunnelCtx.Done():
					return
				default:
				}
				return
			}

			remoteConn, err := client.Dial("tcp", remoteAddr)
			if err != nil {
				localConn.Close()
				continue
			}

			go tunnelCopy(localConn, remoteConn)
		}
	}()

	return result, nil
}

func tunnelCopy(local, remote net.Conn) {
	defer local.Close()
	defer remote.Close()

	done := make(chan struct{}, 2)
	go func() {
		io.Copy(local, remote)
		done <- struct{}{}
	}()
	go func() {
		io.Copy(remote, local)
		done <- struct{}{}
	}()
	<-done
}

func defaultPort(dbType string) int {
	switch dbType {
	case "mysql", "mariadb":
		return 3306
	case "postgres", "postgresql":
		return 5432
	case "oracle":
		return 1521
	case "mssql", "sqlserver":
		return 1433
	default:
		return 0
	}
}
