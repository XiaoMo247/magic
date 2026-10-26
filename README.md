# Magic 传声局

`Magic 传声局（Magic Relay）` 是一个 Vue + Go 多人语音接龙游戏，也是本项目参加“灵矽应用：语音类游戏”比赛的作品。

## 游戏是什么

灵矽 AI 先生成并播报一句密令。第一位玩家听到密令后录音复述，下一位只能听上一棒的内容并继续复述，最后一位玩家根据听到的内容猜原句。每轮结束后公开结果，大家一起看这句话在传递中发生了什么变化。

```text
AI 出题并播报
  → 玩家 1 复述
  → 玩家 2 复述
  → 最后一位猜测
  → 展示传声链和得分
```

当前版本已经实现固定题库、轮流状态机、玩家私有视图、浏览器录音、上一棒音频播放、最终猜测和本地计分。灵矽真实 ASR/TTS/LLM 接口仍待接入。

## 目录结构

```text
frontend/  Vue 3 + Vite 客户端
backend/   Go HTTP API（房间、回合、录音、计分、聊天）
docs/      玩法、接入和项目阶段说明
```

## 启动

在项目根目录打开两个终端：

```bash
go run ./backend
```

```bash
cd frontend
npm install
npm run dev
```

打开 <http://localhost:5173>。前端开发服务器会把 `/api` 请求代理到 `http://localhost:8080`。浏览器录音建议使用 Chrome 或 Edge，并允许麦克风权限。

也可以使用根目录命令：

```bash
make backend   # 启动 Go API
make frontend  # 启动 Vue 开发服务器
make test      # 运行 Go 测试
make build     # 测试后端并构建前端
```

## 文档

- [三人快速上手](docs/QUICKSTART-3P.md)：第一次试玩先看这里。
- [五人完整流程](docs/GAMEPLAY.md)：完整对局、回合和比赛演示脚本。
- [灵矽接入说明](docs/LINXI-INTEGRATION.md)：灵矽能力映射和后续接入方案。
- [项目阶段说明](docs/PROJECT-STATUS.md)：已完成、未完成和延期原因。

## 当前边界

- 浏览器使用 Web Speech API 做识别回退，使用 `speechSynthesis` 播报文本；
- 录音暂存在 Go 进程内存，服务重启后房间和录音会消失；
- 房间同步使用 1 秒轮询；
- 灵矽 API Key 尚未放入项目，真实 ASR、TTS、智能体和大模型接口待后续接入。

每次推送和 Pull Request 都会通过 GitHub Actions 自动执行 Go 测试和 Vue 构建。
