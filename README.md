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

生产构建：

```bash
cd frontend && npm run build
```

当前语音识别使用浏览器 Web Speech API，不支持时自动使用演示回声；灵矽 SDK 尚未接入，后续只需替换前端语音适配层。
