# 部署说明

本项目为单体二进制应用：前端产物通过 `go:embed` 嵌入后端，构建完成后仅需运行一个可执行文件。

---

## 一、本地构建

### 1. 构建前端

```bash
cd web
bun install
bun run build
```

### 2. 构建后端

```bash
go build -o new-api
```

> 前端产物 `web/dist` 会被嵌入二进制，修改前端后必须先重新执行 `bun run build`，再执行 `go build`。

### 3. 运行

```bash
./new-api
```

默认监听 `3000` 端口，首次启动请在浏览器打开 `http://localhost:3000` 完成初始化向导。

---

## 二、Docker 部署

### 方式一：常规构建

```bash
cd web && bun install && bun run build && cd ..
docker build -f Dockerfile.offline -t new-api:latest .
docker run -d -p 3000:3000 \
  -v $(pwd)/data:/data \
  new-api:latest
```

### 方式二：离线环境（预置基础镜像）

离线环境可先构建运行时基础镜像 `Dockerfile.runtime-base`，再以该镜像作为运行阶段基础，避免构建时联网安装系统包。

```bash
docker build -f Dockerfile.runtime-base -t new-api-runtime-base:bookworm .
```

---

## 三、运行配置

```yaml
port: 3000

env:
  SQL_DSN: "postgresql://user:password@pg-host:5432/new-api"
  SESSION_SECRET: "至少32位随机字符串"
  REDIS_CONN_STRING: "redis://:password@redis-host:6379"
  TZ: "Asia/Shanghai"
  BATCH_UPDATE_ENABLED: "true"
  NODE_NAME: "new-api-node-1"

healthCheck:
  path: /api/status
  initialDelay: 30s
  period: 10s
```

---

## 四、开源合规说明

本项目基于 [QuantumNous/new-api](https://github.com/QuantumNous/new-api) 二次开发，遵循 GNU AGPL-3.0 许可。

- 通过公网提供服务时，须向使用者提供与线上版本一致的完整源代码（AGPLv3 §13）
- 分发构建产物（Docker 镜像、二进制、前端 bundle）时，须随附 `LICENSE`、`NOTICE`、`THIRD-PARTY-LICENSES.md`
- 不得移除 `NOTICE` 中的作者归属声明与上游项目链接
