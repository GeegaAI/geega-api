# New API

统一的 LLM API 网关与分发平台：将多家模型供应商聚合为 OpenAI 兼容接口，提供渠道管理、令牌分发、配额计费与运营控制台。

本项目基于 [QuantumNous/new-api](https://github.com/QuantumNous/new-api) 二次开发，遵循 [GNU AGPL-3.0](LICENSE) 许可。

## 功能特性

- **多供应商中继**：OpenAI / Claude / Gemini / 通义千问 / DeepSeek / Vertex / AWS Bedrock 等 40+ 渠道适配，支持 OpenAI、Claude、Gemini 原生协议与流式响应
- **渠道调度**：按模型与分组路由、权重与优先级、渠道亲和、失败重试与自动禁用
- **令牌与配额**：`sk-` 令牌分发、模型范围与过期控制、配额预扣与结算、阶梯计费与倍率配置
- **用户与权限**：注册登录、OAuth / OIDC 单点登录、双因素认证、Passkey、RBAC 角色策略
- **计费与订阅**：充值码、订阅套餐、多通道支付（易支付 / Stripe / Creem）
- **运营能力**：模型定价与排行榜、消费日志与对账、渠道健康检查、系统任务调度
- **管理台**：React 19 + TanStack Router/Query + Tailwind CSS 4，支持深浅主题与多语言

## 快速开始

### 从源码运行

```bash
# 前端
cd web
bun install
bun run build
cd ..

# 后端（前端产物会被 go:embed 进二进制）
go build -o new-api
./new-api
```

启动后访问 `http://localhost:3000` 完成初始化向导。

### Docker

```bash
cd web && bun install && bun run build && cd ..
docker build -f Dockerfile.offline -t new-api:latest .
docker run -d -p 3000:3000 -v $(pwd)/data:/data new-api:latest
```

详细部署与运行环境变量见 [DEPLOY.md](DEPLOY.md)。

### 本地开发

```bash
# 后端（默认 SQLite，读取 .env）
go run .

# 前端（Rsbuild dev server，代理到后端）
cd web && bun run dev
```

## 验证

```bash
go build ./... && go vet ./... && go test ./...

cd web
bun run typecheck
bun run lint
bun run test
```

## 开源合规

本项目以 AGPL-3.0 发布：

- **源码可获取**：任何通过网络与本服务交互的用户，均有权获取与线上运行版本一致的完整源代码（AGPLv3 §13）。本项目源码发布于本仓库，版本与发布标签一一对应。
- **保留声明**：分发本项目或其修改版时，须随附 [LICENSE](LICENSE)、[NOTICE](NOTICE)、[THIRD-PARTY-LICENSES.md](THIRD-PARTY-LICENSES.md)，并保留上游项目归属与链接。
- **标注修改**：修改版须标注变更内容与时间（AGPLv3 §7(c)），变更记录见 [CHANGELOG.md](CHANGELOG.md)。
- **商标**：AGPL-3.0 仅授予著作权许可，不包含商标授权；请勿使用上游名称或标识暗示官方关联。

## 第三方依赖

直接依赖及其许可证清单见 [THIRD-PARTY-LICENSES.md](THIRD-PARTY-LICENSES.md)。发布构建前建议对传递依赖做一次完整许可证审计。

## 贡献

提交前请确保 `go test ./...` 与前端 `bun run typecheck`、`bun run lint`、`bun run test` 通过；提交信息遵循 Conventional Commits。

## 致谢

- [One API](https://github.com/songquanpeng/one-api) © JustSong
- [New API](https://github.com/QuantumNous/new-api) © QuantumNous
