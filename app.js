const players = [
  { name: '小满', role: '正在聆听', score: 860, avatar: '小', cls: 'av-a' },
  { name: '阿岛', role: '已准备', score: 720, avatar: '岛', cls: 'av-b' },
  { name: '北北', role: '已准备', score: 680, avatar: '北', cls: 'av-c' },
  { name: '你', role: '轮到你', score: 540, avatar: '你', cls: 'av-d', current: true },
  { name: '禾木', role: '已准备', score: 410, avatar: '禾', cls: 'av-e' },
];
const prompts = ['雨后的电车穿过橘色隧道', '月亮把影子藏进玻璃杯', '三只蓝色纸船顺流而下', '晚风带来一颗会唱歌的种子'];
const playersEl = document.querySelector('#players');
playersEl.innerHTML = players.map((p) => `<div class="player ${p.current ? 'current' : ''}"><span class="avatar ${p.cls}">${p.avatar}</span><div class="player-info"><strong>${p.name}</strong><small>${p.role}</small></div><span class="player-score">${p.score}</span></div>`).join('');

let promptIndex = 0;
let seconds = 24;
let recording = false;
let timerHandle;
let speechRecognition = null;
let capturedTranscript = '';
const timer = document.querySelector('#timer');
const recordButton = document.querySelector('#recordButton');
const recordLabel = document.querySelector('#recordLabel');
const recognitionBox = document.querySelector('#recognitionBox');
const recognitionText = document.querySelector('#recognitionText');
const accuracy = document.querySelector('#accuracy');
const matchBar = document.querySelector('#matchBar');
const promptText = document.querySelector('#promptText');
const hintText = document.querySelector('#hintText');

const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition;
if (SpeechRecognition) {
  speechRecognition = new SpeechRecognition();
  speechRecognition.lang = 'zh-CN';
  speechRecognition.continuous = false;
  speechRecognition.interimResults = true;
  speechRecognition.onresult = (event) => {
    capturedTranscript = Array.from(event.results).map((result) => result[0].transcript).join('');
    recognitionBox.hidden = false;
    recognitionText.textContent = `“${capturedTranscript || '正在识别…'}”`;
    hintText.textContent = '已捕捉到声音，松开按钮提交回声';
  };
  speechRecognition.onerror = () => {
    capturedTranscript = '';
    hintText.textContent = '没有听清，再试一次，尽量靠近麦克风';
  };
}

function resetTimer() { clearInterval(timerHandle); seconds = 24; timer.textContent = `00:${String(seconds).padStart(2, '0')}`; timerHandle = setInterval(() => { seconds -= 1; timer.textContent = `00:${String(Math.max(seconds, 0)).padStart(2, '0')}`; if (seconds <= 0) clearInterval(timerHandle); }, 1000); }
function replayPrompt() { const utterance = new SpeechSynthesisUtterance(promptText.textContent); utterance.lang = 'zh-CN'; utterance.rate = .86; window.speechSynthesis?.cancel(); window.speechSynthesis?.speak(utterance); hintText.textContent = '密令已重播，准备好后按住按钮'; }
function calculateMatch(transcript, target) {
  const clean = (value) => value.replace(/[^\u4e00-\u9fa5a-z0-9]/gi, '').toLowerCase();
  const heard = clean(transcript);
  const expected = clean(target);
  if (!heard) return 0;
  const matched = [...new Set(heard)].filter((char) => expected.includes(char)).length;
  return Math.max(45, Math.min(99, Math.round((matched / new Set(expected).size) * 100)));
}
function finishRecording() {
  if (!recording) return;
  recording = false;
  recordButton.classList.remove('recording');
  recordLabel.textContent = '按住开始说';
  if (speechRecognition) speechRecognition.stop();
  const fallback = 78 + Math.floor(Math.random() * 18);
  const score = capturedTranscript ? calculateMatch(capturedTranscript, promptText.textContent) : fallback;
  recognitionBox.hidden = false;
  accuracy.textContent = `匹配度 ${score}%`;
  matchBar.style.width = `${score}%`;
  recognitionText.textContent = capturedTranscript ? `“${capturedTranscript}”` : `“${promptText.textContent}” · 演示回声已捕捉`;
  hintText.textContent = score > 88 ? '漂亮！回声非常清晰，+180 分' : '不错的接力，再清晰一点会更高分';
  document.querySelector('#teamScore').textContent = `${Math.min(99, 68 + Math.floor(score / 30))}%`;
  capturedTranscript = '';
}
function startRecording() {
  if (recording) return;
  recording = true;
  recordButton.classList.add('recording');
  recordLabel.textContent = '松开提交回声';
  recognitionBox.hidden = true;
  hintText.textContent = speechRecognition ? '正在聆听你的声音…' : '浏览器未开放识别，使用演示回声…';
  if (speechRecognition) {
    capturedTranscript = '';
    try { speechRecognition.start(); } catch { /* recognition may still be closing */ }
  }
}
recordButton.addEventListener('pointerdown', startRecording); window.addEventListener('pointerup', finishRecording); recordButton.addEventListener('keydown', (e) => { if (e.code === 'Space') { e.preventDefault(); startRecording(); } }); recordButton.addEventListener('keyup', (e) => { if (e.code === 'Space') finishRecording(); });
document.querySelector('#replayButton').addEventListener('click', replayPrompt);
document.querySelector('#soundToggle').addEventListener('click', (e) => { e.currentTarget.textContent = e.currentTarget.textContent === '◖' ? '◗' : '◖'; });
document.querySelector('#clearChat').addEventListener('click', () => { document.querySelector('#chatList').innerHTML = ''; });
function sendChat() { const input = document.querySelector('#chatInput'); const value = input.value.trim(); if (!value) return; const p = document.createElement('p'); p.innerHTML = `<b>你</b><span>${value.replace(/[<>]/g, '')}</span>`; document.querySelector('#chatList').append(p); input.value = ''; }
document.querySelector('#sendChat').addEventListener('click', sendChat); document.querySelector('#chatInput').addEventListener('keydown', (e) => { if (e.key === 'Enter') sendChat(); });
setTimeout(replayPrompt, 450); resetTimer();
