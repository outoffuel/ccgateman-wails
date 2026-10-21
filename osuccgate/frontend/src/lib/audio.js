// Half-Life 効果音ジェネレータ / プレイヤー
// エラー音: otkaz-v-dostupe (アクセス拒否音)
// 通過音: knopka-iz-igry-2 (ボタン・通過音)

import errorSoundUrl from '../assets/sounds/error.mp3';
import passSoundUrl from '../assets/sounds/pass.mp3';

let audioCtx = null;
const audioBuffers = {
  error: null,
  pass: null,
};

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

// 音声ファイルを AudioBuffer としてデコードしてキャッシュ
async function loadAudioBuffer(url) {
  const ctx = getAudioContext();
  if (!ctx) return null;
  try {
    const response = await fetch(url);
    const arrayBuffer = await response.arrayBuffer();
    return await ctx.decodeAudioData(arrayBuffer);
  } catch (e) {
    console.warn('AudioContext decode failed:', e);
    return null;
  }
}

// 初回ロード時にプリロードを試行
if (typeof window !== 'undefined') {
  // バックグラウンドでデコードキャッシュを準備
  setTimeout(() => {
    loadAudioBuffer(passSoundUrl).then((buf) => {
      if (buf) audioBuffers.pass = buf;
    });
    loadAudioBuffer(errorSoundUrl).then((buf) => {
      if (buf) audioBuffers.error = buf;
    });
  }, 100);
}

function playAudioBufferOrFallback(type) {
  const ctx = getAudioContext();
  const url = type === 'error' ? errorSoundUrl : passSoundUrl;

  if (ctx) {
    if (audioBuffers[type]) {
      try {
        const source = ctx.createBufferSource();
        source.buffer = audioBuffers[type];
        source.connect(ctx.destination);
        source.start(0);
        return;
      } catch (e) {
        console.warn('Buffer playback failed, fallback to Audio element:', e);
      }
    } else {
      // キャッシュが無ければ非同期ロードしつつ、初回はAudioで鳴らす
      loadAudioBuffer(url).then((buf) => {
        if (buf) audioBuffers[type] = buf;
      });
      try {
        const audio = new Audio(url);
        audio.play().catch((e) => console.warn('Audio element play error:', e));
        return;
      } catch (err) {
        console.warn('Audio element failed:', err);
      }
    }
  }

  // フォールバック
  try {
    const audio = new Audio(url);
    audio.play().catch((e) => console.warn('Fallback audio failed:', e));
  } catch (e) {
    console.warn('Audio instantiation failed:', e);
  }
}

export const sounds = {
  // 通過音 (Half-Life)
  pass() {
    playAudioBufferOrFallback('pass');
  },

  // エラー音 (Half-Life)
  error() {
    playAudioBufferOrFallback('error');
  },

  // 学生入室: 通過音
  studentEntry() {
    this.pass();
  },

  // 学生退室: 通過音
  studentExit() {
    this.pass();
  },

  // 教職員入室: 通過音
  staffEntry() {
    this.pass();
  },

  // 教職員退室: 通過音
  staffExit() {
    this.pass();
  },

  // エラー/警告: エラー音
  booboo() {
    this.error();
  },

  // サウンドタイプ名に応じた再生ディスパッチャ
  play(soundType) {
    switch (soundType) {
      case 'studentEntry':
      case 'studentExit':
      case 'staffEntry':
      case 'staffExit':
      case 'pass':
      case 'success':
        this.pass();
        break;
      case 'booboo':
      case 'error':
      default:
        this.error();
        break;
    }
  }
};
