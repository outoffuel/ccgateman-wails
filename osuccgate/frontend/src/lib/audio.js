// Web Audio API によるシンセサイズ効果音ジェネレータ
// 外部音声ファイル不要で動作

let audioCtx = null;

function getAudioContext() {
  if (!audioCtx) {
    const AudioContext = window.AudioContext || window.webkitAudioContext;
    if (AudioContext) {
      audioCtx = new AudioContext();
    }
  }
  if (audioCtx && audioCtx.state === 'suspended') {
    audioCtx.resume();
  }
  return audioCtx;
}

/**
 * 周波数とタイミングを指定してトーンを生成
 */
function playTone(freq, type, duration, startTime = 0, gainLevel = 0.15) {
  const ctx = getAudioContext();
  if (!ctx) return;

  const osc = ctx.createOscillator();
  const gain = ctx.createGain();

  osc.type = type;
  osc.frequency.setValueAtTime(freq, ctx.currentTime + startTime);

  gain.gain.setValueAtTime(gainLevel, ctx.currentTime + startTime);
  gain.gain.exponentialRampToValueAtTime(0.0001, ctx.currentTime + startTime + duration);

  osc.connect(gain);
  gain.connect(ctx.destination);

  osc.start(ctx.currentTime + startTime);
  osc.stop(ctx.currentTime + startTime + duration);
}

export const sounds = {
  // 学生入室: ポップで爽快な上昇メロディ (ド・ミ・ソ・高ド)
  studentEntry() {
    const ctx = getAudioContext();
    if (!ctx) return;
    playTone(523.25, 'sine', 0.25, 0.00, 0.2); // C5
    playTone(659.25, 'sine', 0.25, 0.10, 0.2); // E5
    playTone(783.99, 'sine', 0.25, 0.20, 0.2); // G5
    playTone(1046.50, 'sine', 0.45, 0.30, 0.25); // C6
  },

  // 学生退室: 落ち着いた終了メロディ (高ド・ソ・ミ・ド)
  studentExit() {
    const ctx = getAudioContext();
    if (!ctx) return;
    playTone(880.00, 'sine', 0.25, 0.00, 0.2); // A5
    playTone(783.99, 'sine', 0.25, 0.12, 0.2); // G5
    playTone(659.25, 'sine', 0.25, 0.24, 0.2); // E5
    playTone(523.25, 'sine', 0.40, 0.36, 0.2); // C5
  },

  // 教職員入室: 格式高い上品なコード
  staffEntry() {
    const ctx = getAudioContext();
    if (!ctx) return;
    playTone(587.33, 'triangle', 0.35, 0.00, 0.25); // D5
    playTone(739.99, 'triangle', 0.35, 0.12, 0.25); // F#5
    playTone(880.00, 'triangle', 0.35, 0.24, 0.25); // A5
    playTone(1174.66, 'sine', 0.55, 0.36, 0.30);    // D6
  },

  // 教職員退室: 穏やかな低音コード
  staffExit() {
    const ctx = getAudioContext();
    if (!ctx) return;
    playTone(783.99, 'triangle', 0.3, 0.00, 0.2); // G5
    playTone(659.25, 'triangle', 0.3, 0.15, 0.2); // E5
    playTone(440.00, 'triangle', 0.5, 0.30, 0.2); // A4
  },

  // エラー/警告 (ブザー音: ブー・ブー)
  booboo() {
    const ctx = getAudioContext();
    if (!ctx) return;
    playTone(180, 'sawtooth', 0.20, 0.00, 0.3);
    playTone(150, 'sawtooth', 0.30, 0.25, 0.3);
  },

  // サウンドタイプ名に応じた再生ディスパッチャ
  play(soundType) {
    switch (soundType) {
      case 'studentEntry':
        this.studentEntry();
        break;
      case 'studentExit':
        this.studentExit();
        break;
      case 'staffEntry':
        this.staffEntry();
        break;
      case 'staffExit':
        this.staffExit();
        break;
      case 'booboo':
      default:
        this.booboo();
        break;
    }
  }
};
