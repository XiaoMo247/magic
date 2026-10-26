# Magic 传声局：实现说明

## 前端实现

前端入口在 `frontend/src/App.vue`，当前使用 Vue `<script setup>` 管理状态：

- `room`：当前玩家看到的私有房间视图；
- `mode`：首页或房间页；
- `listening`：当前是否正在录音；
- `transcript`：浏览器语音识别的临时文本；
- `MediaRecorder`：采集当前棒的音频；
- `SpeechRecognition`：Chrome/Edge 中文识别，失败时仍允许演示流程。

房间每秒刷新一次，并在请求 URL 中带上本地保存的 `echo-player`。这样同一个房间的不同浏览器会获得不同的私有内容。

## 后端实现

后端入口在 `backend/main.go`，只依赖 Go 标准库。`Room` 是当前游戏的聚合状态，`Stage` 区分：

- `relay`：玩家提交复述；
- `guess`：最后一位提交猜测；
- `finished`：最终揭晓。

`submit` 会完成四件事：

1. 校验当前玩家和阶段；
2. 计算本次匹配度；
3. 保存文本、录音和分数；
4. 将控制权移动到下一位或切换到猜测阶段。

`guess` 会比较最终猜测和原始密令，并在最后一轮完成后保留完整的 `Turns` 给揭晓页面展示。

## 当前计分

当前是无外部依赖的本地回退算法：去除空白后计算目标字符串和输入字符串的重复字符比例，限制在 45–99 分。它只用于保证离线试玩闭环，不代表最终语义评分方案。

## 录音实现

浏览器使用 `navigator.mediaDevices.getUserMedia({ audio: true })` 获取麦克风，再用 `MediaRecorder` 生成音频 data URL。Go 接收 `audio_data` 并暂存在房间内，大小上限约 2MB 原始录音对应的请求体。

这套实现适合本地演示，不适合生产。正式版本应上传对象存储，只在回合中保存音频 URL，并设置过期清理策略。

## 灵矽接入位置

当前前端没有直接携带灵矽密钥。后续应在 Go 增加语音提供者接口：

```go
type VoiceProvider interface {
    Synthesize(text string) ([]byte, error)
    Transcribe(audio []byte) (string, error)
}
```

本地 Mock 与灵矽实现共用游戏状态机，避免灵矽暂时不可用时无法演示。详细能力和接入边界见 [LINXI-INTEGRATION.md](LINXI-INTEGRATION.md)。

