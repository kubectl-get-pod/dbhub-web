.PHONY: dev dev-backend dev-frontend build build-win build-linux clean

# 开发模式：前后端独立运�?dev-backend:
	cd backend && go run .

dev-frontend:
	cd frontend && npm run dev

dev:
	@echo "请分别在两个终端运行:"
	@echo "  make dev-backend"
	@echo "  make dev-frontend"

# 生产构建
build: build-frontend build-backend

build-frontend:
	cd frontend && npm ci && npm run build

build-backend: build-frontend
	rm -rf backend/frontend-dist
	cp -r frontend/dist backend/frontend-dist
	cd backend && CGO_ENABLED=0 go build -ldflags="-s -w" -o ../dbhub-web .

# Windows 交叉编译
build-win: build-frontend
	rm -rf backend/frontend-dist
	cp -r frontend/dist backend/frontend-dist
	cd backend && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ../dbhub-web.exe .

# Linux 交叉编译
build-linux: build-frontend
	rm -rf backend/frontend-dist
	cp -r frontend/dist backend/frontend-dist
	cd backend && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ../dbhub-web .

clean:
	rm -rf frontend/dist frontend/node_modules backend/frontend-dist dbhub-web dbhub-web.exe
