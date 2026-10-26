# 灵矽接入方案

灵矽官方文档显示，平台可以提供 ASR、TTS、智能体、LLM、SSE 业务服务和 WebSocket 音频链路。Magic 传声局建议把灵矽放在 Go 后端之后，浏览器不保存 API Key。

## 能力映射

| 游戏能力 | 灵矽能力 | 当前状态 |
| --- | --- | --- |
| AI 生成密令 | 智能体 / LLM API | 固定题库 |
| AI 说出密令 | TTS | 浏览器 speechSynthesis |
| 玩家语音识别 | ASR / WebSocket | 浏览器 SpeechRecognition |
| 最终猜测判定 | LLM 语义比较 | 本地字符匹配 |
| 主持人点评 | 智能体 / LLM | 未接入 |
| 音频实时传输 | WebSocket | 浏览器录音上传 |

## 推荐链路

```text
Vue 录音
  → Go 游戏服务
  → 灵矽 ASR
  → 保存 transcript
  → 下一位播放或 TTS
  → 最后一位猜测
  → 灵矽 LLM 语义评分
```

灵矽第三方业务服务文档描述了 `POST /api/chat/stream` 和 SSE 响应模式；如果使用灵矽智能体做主持人，Go 可以作为业务服务接收识别后的文本，再以 `message` 事件返回主持词。设备型实时音频则按官方 WebSocket 协议接入，支持 PCM/Opus 音频帧。

## 接入顺序

1. 在 Go 中定义 `VoiceProvider`，保留当前本地 Mock；
2. 用灵矽 ASR 替换浏览器识别结果；
3. 用灵矽 TTS 替换密令和上一棒的浏览器朗读；
4. 用 LLM 生成主题、难度和密令；
5. 用 LLM 对原句与最终猜测做语义比较；
6. 增加 AI 主持人点评和回合总结。

API Key 只放后端环境变量或密钥服务，不能打包进 `frontend`。

