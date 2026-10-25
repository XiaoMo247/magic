# 回声密令

Vue + Go 实现的多人语音接力游戏，作为“灵矽应用：语音类游戏”比赛原型。

## 目录结构

```text
frontend/  Vue 3 + Vite 客户端
backend/   Go HTTP API（房间、回合、计分、聊天）
docs/      设计和接入说明
```

## 启动

在项目根目录打开两个终端：

终端 1：

```bash
go run ./backend
```

终端 2：

```bash
cd frontend
npm install
npm run dev
```

打开 <http://localhost:5173>。前端开发服务器会把 `/api` 请求代理到 `http://localhost:8080`。

也可以使用根目录命令：

```bash
make backend   # 启动 Go API
make frontend  # 启动 Vue 开发服务器
make test      # 运行 Go 测试
make build     # 测试后端并构建前端
```

每次推送和 Pull Request 都会通过 GitHub Actions 自动执行后端测试和前端构建。

生产构建：

```bash
cd frontend && npm run build
```

当前语音识别使用浏览器 Web Speech API，不支持时自动使用演示回声；灵矽 SDK 尚未接入，后续只需替换前端语音适配层。

详细的 5 人对局流程、计分规则、接口状态和比赛演示脚本见 [docs/GAMEPLAY.md](docs/GAMEPLAY.md)。第一次试玩建议先看 [docs/QUICKSTART-3P.md](docs/QUICKSTART-3P.md)。
