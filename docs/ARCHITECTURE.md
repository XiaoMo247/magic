# Magic 传声局：架构设计

## 1. 总体架构

当前版本采用前后端分离的轻量架构：

```text
浏览器（Vue 3 + Vite）
  ├─ 房间大厅、回合状态、录音和播放
  ├─ 每秒轮询 GET /api/rooms/{code}
  └─ 通过 player_id 获取私有视图
          │
          ▼
Go 游戏服务（标准库 net/http）
  ├─ 房间和玩家管理
  ├─ 接龙状态机
  ├─ 权限过滤
  ├─ 文本计分
  └─ 内存保存回合和 data URL 录音
          │
          ▼
后续灵矽适配层（尚未接入）
  ├─ ASR：语音转文字
  ├─ TTS：AI 播报和上一棒复述
  ├─ LLM：出题、语义评分和点评
  └─ 智能体 / SSE / WebSocket
```

## 2. 核心状态机

房间状态：

```text
lobby
  └─ start → playing / relay
playing / relay
  ├─ submit（普通传声）→ 下一个传声玩家
  └─ 最后一棒传声完成 → playing / guess
playing / guess
  ├─ guess（最终猜测）→ 下一轮 relay
  └─ 最后一轮猜测完成 → finished
finished
  └─ 展示最后一轮传声链、最终猜测和排行榜
```

每轮的起始玩家向后轮换一位。5 人模式每轮有 4 次传声和 1 次猜测，3 人模式每轮有 2 次传声和 1 次猜测。

## 3. 信息隔离

房间查询必须携带 `player_id`：

```text
GET /api/rooms/MGMP?player_id=p_123456
```

后端根据当前阶段返回内容：

- 第一棒：看到原始密令；
- 中间玩家：只能看到上一棒文本和录音；
- 最后一棒：只能看到最后一棒文本和录音；
- 非当前玩家：看到等待状态，不拿到当前传递内容；
- `finished`：所有玩家可以看到完整揭晓结果。

## 4. 数据边界

当前原型的 `Room` 保存：

- 玩家、准备状态和积分；
- 当前轮次、当前阶段和当前玩家；
- 原始密令、当前传递文本；
- 每一棒的文本、分数和录音 data URL；
- 最终猜测和聊天消息。

生产版应将这些数据拆分为房间表、玩家表、回合表和音频对象存储，并使用 Redis 或 WebSocket 做实时同步。

## 5. 关键接口

| 接口 | 用途 |
| --- | --- |
| `POST /api/rooms` | 创建房间 |
| `POST /api/rooms/{code}/join` | 加入房间 |
| `POST /api/rooms/{code}/ready` | 玩家准备 |
| `POST /api/rooms/{code}/start` | 房主开始 |
| `GET /api/rooms/{code}?player_id=...` | 获取私有房间视图 |
| `POST /api/rooms/{code}/submit` | 提交传声文本和录音 |
| `POST /api/rooms/{code}/guess` | 提交最终猜测 |
| `POST /api/rooms/{code}/chat` | 发送房间消息 |

