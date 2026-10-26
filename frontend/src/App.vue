<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
const api = "/api",
  name = ref(localStorage.getItem("echo-name") || ""),
  roomCode = ref(""),
  room = ref(null),
  mode = ref("home"),
  error = ref(""),
  chat = ref(""),
  transcript = ref(""),
  listening = ref(false),
  busy = ref(false);
let poller, recognition, recorder, recordingStream, recordingStop;
let audioChunks = [];
const me = computed(() =>
    room.value?.players?.find(
      (p) => p.id === localStorage.getItem("echo-player"),
    ),
  ),
  isMyTurn = computed(
    () =>
      room.value?.phase === "playing" &&
      room.value?.current_player_id === me.value?.id,
  ),
  stageLabel = computed(() =>
    room.value?.stage === "guess" ? "最终猜测" : "语音接力",
  ),
  displayPhrase = computed(
    () =>
      room.value?.relay_text ||
      room.value?.prompt ||
      "等待当前玩家完成这一棒",
  ),
  recognitionAvailable = !!(
    window.SpeechRecognition || window.webkitSpeechRecognition
  );
async function request(path, options = {}) {
  let r;
  try {
    r = await fetch(`${api}${path}`, {
      headers: { "Content-Type": "application/json" },
      ...options,
    });
  } catch {
    throw Error("无法连接 Go 服务，请先在项目根目录运行：go run ./backend");
  }
  const d = await r.json().catch(() => ({}));
  if (!r.ok) throw Error(d.error || "请求失败");
  return d;
}
async function createRoom() {
  if (busy.value) return;
  busy.value = true;
  error.value = "";
  try {
    const d = await request("/rooms", {
      method: "POST",
      body: JSON.stringify({ name: name.value || "玩家" }),
    });
    localStorage.setItem("echo-name", name.value || "玩家");
    localStorage.setItem("echo-player", d.player_id);
    roomCode.value = d.code;
    mode.value = "room";
    await refresh();
  } catch (e) {
    error.value = e.message;
  } finally {
    busy.value = false;
  }
}
async function joinRoom() {
  if (busy.value) return;
  busy.value = true;
  error.value = "";
  try {
    const code = roomCode.value.trim().toUpperCase();
    if (code.length !== 4) throw Error("请输入 4 位房间码");
    const d = await request(`/rooms/${code}/join`, {
      method: "POST",
      body: JSON.stringify({ name: name.value || "玩家" }),
    });
    localStorage.setItem("echo-name", name.value || "玩家");
    localStorage.setItem("echo-player", d.player_id);
    roomCode.value = code;
    mode.value = "room";
    await refresh();
  } catch (e) {
    error.value = e.message;
  } finally {
    busy.value = false;
  }
}
async function refresh() {
  if (!roomCode.value) return;
  try {
    const playerID = localStorage.getItem("echo-player");
    room.value = await request(
      `/rooms/${roomCode.value}${playerID ? `?player_id=${encodeURIComponent(playerID)}` : ""}`,
    );
  } catch (e) {
    error.value = e.message;
  }
}
async function ready() {
  if (!me.value) {
    error.value = "当前玩家身份已失效，请重新加入房间";
    return;
  }
  try {
    await request(`/rooms/${roomCode.value}/ready`, {
      method: "POST",
      body: JSON.stringify({ player_id: me.value.id }),
    });
    await refresh();
  } catch (e) {
    error.value = e.message;
  }
}
async function startGame() {
  if (!me.value) {
    error.value = "当前玩家身份已失效，请重新加入房间";
    return;
  }
  if (busy.value) return;
  busy.value = true;
  error.value = "";
  try {
    await request(`/rooms/${roomCode.value}/start`, {
      method: "POST",
      body: JSON.stringify({ player_id: me.value.id }),
    });
    await refresh();
  } catch (e) {
    error.value = e.message;
  } finally {
    busy.value = false;
  }
}
async function begin() {
  if (!isMyTurn.value || listening.value) return;
  listening.value = true;
  transcript.value = "";
  audioChunks = [];
  if (recognitionAvailable) {
    const R = window.SpeechRecognition || window.webkitSpeechRecognition;
    recognition = new R();
    recognition.lang = "zh-CN";
    recognition.interimResults = true;
    recognition.onresult = (e) =>
      (transcript.value = Array.from(e.results)
        .map((r) => r[0].transcript)
        .join(""));
    recognition.start();
  }
  try {
    recordingStream = await navigator.mediaDevices?.getUserMedia({ audio: true });
    if (recordingStream) {
      recorder = new MediaRecorder(recordingStream);
      recordingStop = new Promise((resolve) => {
        recorder.ondataavailable = (event) => event.data.size && audioChunks.push(event.data);
        recorder.onstop = async () => resolve(await audioToDataURL(new Blob(audioChunks, { type: recorder.mimeType })));
      });
      recorder.start();
    }
  } catch {
    recordingStream = null;
    error.value = "麦克风不可用，将提交语音识别文本";
  }
}
function audioToDataURL(blob) {
  return new Promise((resolve) => {
    if (!blob.size) return resolve("");
    const reader = new FileReader();
    reader.onloadend = () => resolve(reader.result);
    reader.readAsDataURL(blob);
  });
}
async function end() {
  if (!listening.value && !transcript.value && recorder?.state !== "recording") return;
  recognition?.stop();
  if (recorder?.state === "recording") recorder.stop();
  recordingStream?.getTracks().forEach((track) => track.stop());
  const audioData = recordingStop ? await recordingStop : "";
  recorder = null;
  recordingStop = null;
  listening.value = false;
  try {
    const path = room.value?.stage === "guess" ? "/guess" : "/submit";
    const payload =
      room.value?.stage === "guess"
        ? { player_id: me.value.id, guess: transcript.value }
        : { player_id: me.value.id, transcript: transcript.value, audio_data: audioData };
    await request(`/rooms/${roomCode.value}${path}`, {
      method: "POST",
      body: JSON.stringify(payload),
    });
    transcript.value = "";
    await refresh();
  } catch (e) {
    error.value = e.message;
  }
}
function speak() {
  const u = new SpeechSynthesisUtterance(
    room.value?.relay_text || room.value?.prompt || "",
  );
  u.lang = "zh-CN";
  u.rate = 0.86;
  window.speechSynthesis?.cancel();
  window.speechSynthesis?.speak(u);
}
async function sendChat() {
  if (!chat.value.trim()) return;
  try {
    await request(`/rooms/${roomCode.value}/chat`, {
      method: "POST",
      body: JSON.stringify({ player_id: me.value.id, text: chat.value.trim() }),
    });
    chat.value = "";
    await refresh();
  } catch (e) {
    error.value = e.message;
  }
}
function leave() {
  room.value = null;
  mode.value = "home";
  roomCode.value = "";
  clearInterval(poller);
}
function playerName(playerID) {
  return room.value?.players?.find((player) => player.id === playerID)?.name || "玩家";
}
function handleKeyDown(event) {
  if (event.code === "Space" && isMyTurn.value && !event.repeat) {
    event.preventDefault();
    begin();
  }
}
function handleKeyUp(event) {
  if (event.code === "Space" && isMyTurn.value) {
    event.preventDefault();
    end();
  }
}
onMounted(() => {
  poller = setInterval(refresh, 1000);
  window.addEventListener("keydown", handleKeyDown);
  window.addEventListener("keyup", handleKeyUp);
});
onBeforeUnmount(() => {
  clearInterval(poller);
  window.removeEventListener("keydown", handleKeyDown);
  window.removeEventListener("keyup", handleKeyUp);
});
</script>
<template>
  <div class="app-shell">
    <header class="topbar">
      <a class="brand" href="#" @click.prevent="leave"
        ><span class="brand-mark">◒</span>Magic 传声局</a
      >
      <div v-if="room" class="room-meta">
        <span class="live-dot"></span>房间 {{ room.code }} ·
        {{ room.players.length }} 人
      </div>
      <span class="wordmark">灵矽应用 · 语音游戏创意赛</span>
    </header>
    <main v-if="mode === 'home'" class="landing">
      <section>
        <div class="eyebrow">语音接力 · 轻策略派对</div>
        <h1>把一句话，<em>传到最后。</em></h1>
        <p class="subtitle">
          听清、记住、说准。每一轮都会加长记忆链，看看谁能守住回声。
        </p>
        <div class="entry-card">
          <label
            >你的昵称<input
              v-model="name"
              maxlength="12"
              placeholder="输入昵称"
              @keyup.enter="createRoom"
          /></label>
          <div class="entry-actions">
            <button class="primary" :disabled="busy" @click="createRoom">
              {{ busy ? "创建中…" : "创建房间 →" }}
            </button>
            <div class="join">
              <input
                v-model="roomCode"
                maxlength="4"
                placeholder="房间码"
                @keyup.enter="joinRoom"
              /><button class="secondary" :disabled="busy" @click="joinRoom">
                {{ busy ? "加入中…" : "加入" }}
              </button>
            </div>
          </div>
        </div>
        <p v-if="error" class="error">{{ error }}</p>
      </section>
      <aside class="landing-art">
        <div class="orb"></div>
        <div class="art-label">ECHO<br />RELAY</div>
        <div class="art-caption">3–8 人 · 约 5 分钟一局</div>
      </aside>
    </main>
    <main v-else class="game-layout">
      <section class="main-column">
        <div class="eyebrow">
          {{ stageLabel }} · 第 {{ room.round }} / {{ room.max_rounds }} 轮
        </div>
        <div class="title-row">
          <div>
            <h1>把一句话，<em>传到最后。</em></h1>
            <p class="subtitle">轮流听取密令，再用你的声音把回声交给下一位。</p>
          </div>
          <div class="round-badge">
            <span>剩余时间</span
            ><strong
              >00:{{ String(room.remaining_seconds).padStart(2, "0") }}</strong
            >
          </div>
        </div>
        <div v-if="room.phase === 'lobby'" class="game-card lobby-card">
          <span class="status-pill">等待开始</span>
          <h2>房间已准备好</h2>
          <p>把房间码发给队友，所有人准备后即可开局。</p>
          <div class="room-code">{{ room.code }}</div>
          <button v-if="!me?.ready" class="primary" @click="ready">
            我准备好了 →</button
          ><button v-else class="secondary" disabled>已准备 ✓</button
          ><button
            v-if="me?.host && room.players.length >= 2"
            class="start-link"
            :disabled="busy"
            @click="startGame"
          >
            {{ busy ? "启动中…" : "开始游戏" }}
          </button>
          <p
            v-if="me?.host && room.players.some((player) => !player.ready)"
            class="lobby-hint"
          >
            还有
            {{
              room.players.filter((player) => !player.ready).length
            }}
            位玩家未准备
          </p>
        </div>
        <div v-else-if="room.phase === 'finished'" class="game-card reveal-card">
          <span class="status-pill">本局完成</span>
          <h2>这句话，传成了什么？</h2>
          <p class="reveal-source">原句：{{ room.prompt }}</p>
          <div class="relay-chain">
            <div v-for="(turn, index) in room.turns" :key="`${turn.player_id}-${index}`" class="relay-item">
              <div class="relay-meta"><span>{{ index + 1 }} · {{ playerName(turn.player_id) }}</span><strong>{{ turn.score }} 分</strong></div>
              <p>{{ turn.transcript || "未识别到文字" }}</p>
              <audio v-if="turn.audio_data" :src="turn.audio_data" controls></audio>
            </div>
            <div class="relay-item guess-item">
              <div class="relay-meta"><span>最终猜测 · {{ room.final_guess_player_name || "玩家" }}</span><strong>{{ room.last_result?.score || 0 }} 分</strong></div>
              <p>{{ room.final_guess || room.last_result?.transcript || "未提交猜测" }}</p>
            </div>
          </div>
          <p class="reveal-note">本轮结果已公开。重新创建房间即可开始下一局。</p>
        </div>
        <div v-else class="game-card">
          <div class="card-topline">
            <span class="status-pill">{{
              isMyTurn ? (room.stage === "guess" ? "你的猜测" : "你的回合") : "等待队友"
            }}</span
            ><span class="muted"
              >{{ room.current_player_name }} · {{ room.round }} /
              {{ room.max_rounds }}</span
            >
          </div>
          <div class="prompt-panel">
            <span class="prompt-kicker">{{ room.stage === "guess" ? "上一棒传来的内容" : "当前传递内容" }}</span>
            <div class="prompt-text">{{ displayPhrase }}</div>
            <audio
              v-if="room.relay_audio"
              class="relay-audio"
              :src="room.relay_audio"
              controls
            ></audio>
            <div class="waveform"><i v-for="n in 20" :key="n"></i></div>
            <p class="hint">
              {{ isMyTurn ? (listening ? "正在聆听你的声音…" : room.stage === "guess" ? "听完上一棒后，说出你认为的原句" : "听清当前内容，再按住按钮复述") : "等待当前玩家完成这一棒" }}
            </p>
          </div>
          <div class="record-row">
            <button
              class="record-button"
              :class="{ recording: listening }"
              @pointerdown="begin"
              @pointerup="end"
              @pointerleave="end"
            >
              <span class="mic-symbol">●</span
              >{{ listening ? "松开提交" : room.stage === "guess" ? "按住开始猜" : "按住开始说"
              }}<kbd>空格</kbd></button
            ><button class="secondary" @click="speak">↻ {{ room.stage === "guess" ? "重播上一棒" : "重播内容" }}</button>
          </div>
          <div v-if="room.last_result" class="recognition">
            <div class="recognition-label">
              <span>上一条回声</span
              ><span>匹配度 {{ room.last_result.score }}%</span>
            </div>
            <p>“{{ room.last_result.transcript || "演示回声" }}”</p>
            <div class="match-bar">
              <span :style="{ width: `${room.last_result.score}%` }"></span>
            </div>
          </div>
        </div>
        <p v-if="error" class="error">{{ error }}</p>
      </section>
      <aside class="side-column">
        <div class="side-head">
          <div>
            <span class="eyebrow">现场玩家</span>
            <h2>回声小队</h2>
          </div>
          <span class="player-count">{{ room.players.length }} 人</span>
        </div>
        <div class="players">
          <div
            v-for="p in room.players"
            :key="p.id"
            class="player"
            :class="{ current: p.id === room.current_player_id }"
          >
            <span class="avatar">{{ p.name.slice(0, 1) }}</span>
            <div class="player-info">
              <strong>{{ p.name }} <small v-if="p.host">房主</small></strong
              ><small>{{ p.ready ? "已准备" : "未准备" }}</small>
            </div>
            <span class="player-score">{{ p.score }}</span>
          </div>
        </div>
        <div class="score-card">
          <div class="score-head">
            <span>团队进度</span><strong>{{ room.team_progress }}%</strong>
          </div>
          <div class="progress">
            <span :style="{ width: `${room.team_progress}%` }"></span>
          </div>
          <p>完成 {{ room.round - 1 }} 轮，继续守住回声</p>
        </div>
        <div class="chat-card">
          <div class="chat-head"><span>现场短讯</span></div>
          <div class="chat-list">
            <p v-for="(m, i) in room.chat" :key="i">
              <b>{{ m.name }}</b
              ><span>{{ m.text }}</span>
            </p>
          </div>
          <div class="chat-input">
            <input
              v-model="chat"
              placeholder="发一句鼓励…"
              maxlength="28"
              @keyup.enter="sendChat"
            /><button @click="sendChat">↑</button>
          </div>
        </div>
      </aside>
    </main>
    <footer class="footer">
      <span>无需注册 · 适合 3–8 人</span><span>语音能力可替换为灵矽 SDK</span>
    </footer>
  </div>
</template>
