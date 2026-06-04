package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"dbhub-web/handler"
)

//go:embed all:frontend-dist
var frontendAssets embed.FS

func main() {
	// 确保数据目录存在
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("无法获取用户目录: %v", err)
	}
	dataDir := filepath.Join(homeDir, ".dbhub")
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		log.Fatalf("无法创建数据目录: %v", err)
	}

	mux := http.NewServeMux()

	// API 路由
	handler.RegisterRoutes(mux, dataDir)

	// 前端静态文件 + SPA fallback
	serveFrontend(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("dbhub-web 启动成功，访问 http://localhost:%s", port)

	// 优雅退出
	srv := &http.Server{Addr: ":" + port, Handler: mux}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务启动失败: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("正在关闭服务...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("关闭异常: %v", err)
	}
	log.Println("服务已关闭")
}

// serveFrontend 提供前端静态文件，对不存在的路径回退到 index.html（SPA 兼容）
func serveFrontend(mux *http.ServeMux) {
	distFS, err := fs.Sub(frontendAssets, "frontend-dist")
	if err != nil {
		log.Println("[DEV] 前端未嵌入，API 模式运行（前端请通过 npm run dev 启动）")
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" || r.URL.Path == "" {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"status":"dbhub-web API running","mode":"dev"}`))
				return
			}
			http.NotFound(w, r)
		})
		return
	}

	// 内嵌内容转为可读的 http.FileSystem
	fileServer := http.FileServer(http.FS(distFS))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// API 路由由 Go 1.22 ServeMux 精确匹配，不会走到这里
		// 只有非 /api/* 的请求才会到这里
		path := strings.TrimPrefix(r.URL.Path, "/")

		// 尝试打开文件
		f, err := distFS.Open(path)
		if err != nil {
			// 文件不存在或路径为目录，回退到 index.html
			r.URL.Path = "/"
		} else {
			f.Close()
		}

		fileServer.ServeHTTP(w, r)
	})
}
