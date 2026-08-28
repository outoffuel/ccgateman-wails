<script>
  import { onMount, onDestroy } from 'svelte';
  import { sounds } from '../audio.js';
  import { ProcessSwipe } from '../../../wailsjs/go/main/App.js';
  import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime.js';

  export let onOpenAdmin = () => {};

  let swipeInput;
  let inputValue = '';
  let status = 'idle'; // 'idle' | 'loading' | 'success' | 'error'
  let currentResult = null;
  let resetTimeout = null;

  // 入力欄に常時フォーカスを当てる
  export function focusInput() {
    if (swipeInput) {
      swipeInput.focus();
    }
  }

  onMount(() => {
    focusInput();

    // 画面のどこをクリックしてもフォーカスを維持
    const handleClick = () => focusInput();
    window.addEventListener('click', handleClick);

    // NFC PaSoRiからのバックグラウンド打刻イベントを受信
    EventsOn('card_swiped', (resp) => {
      handleSwipeResult(resp);
    });

    return () => {
      window.removeEventListener('click', handleClick);
      EventsOff('card_swiped');
      if (resetTimeout) clearTimeout(resetTimeout);
    };
  });

  async function handleKeydown(e) {
    if (e.key === 'Enter') {
      const cardId = inputValue.trim();
      inputValue = '';
      if (!cardId) return;

      status = 'loading';
      try {
        const res = await ProcessSwipe(cardId);
        handleSwipeResult(res);
      } catch (err) {
        handleSwipeResult({
          success: false,
          errorMessage: 'システム通信エラー',
          soundType: 'booboo'
        });
      }
    }
  }

  function handleSwipeResult(res) {
    if (!res) return;
    if (res.isDebounced) return; // デバウンス時は画面遷移させない

    currentResult = res;

    if (resetTimeout) {
      clearTimeout(resetTimeout);
    }

    if (res.success) {
      status = 'success';
      // 効果音の再生
      if (res.soundType) {
        sounds.play(res.soundType);
      }

      // 4秒後に待機画面へリセット
      resetTimeout = setTimeout(() => {
        status = 'idle';
        currentResult = null;
        focusInput();
      }, 4000);
    } else {
      status = 'error';
      sounds.play(res.soundType || 'booboo');

      // 3.5秒後に待機画面へリセット
      resetTimeout = setTimeout(() => {
        status = 'idle';
        currentResult = null;
        focusInput();
      }, 3500);
    }
  }
</script>

<div class="kiosk-container" on:click={focusInput}>
  <!-- 右上の管理者画面切替ボタン (小さく配置) -->
  <button 
    class="admin-switch-btn" 
    on:click|stopPropagation={onOpenAdmin}
    title="管理者画面を開く"
  >
    ⚙️ 管理者モード
  </button>

  <!-- 非表示の磁気カードHID受信用input (常にフォーカス) -->
  <input 
    type="text" 
    id="swipeInput" 
    bind:this={swipeInput} 
    bind:value={inputValue} 
    on:keydown={handleKeydown} 
    autofocus 
    autocomplete="off" 
  />

  <div class="card">
    <div class="title-text">入退室管理システム</div>

    <div id="display">
      {#if status === 'idle'}
        <!-- 待機中画面 -->
        <div id="content" class="animate-fade">
          <div style="font-size: 2.8em; color: #34495e; font-weight: bold;">学生証を通してください</div>
          <div style="font-size: 1.6em; color: #bdc3c7; margin-top: 25px;">読み取り待機中...</div>
          <div class="mt-8 text-slate-400 text-sm flex items-center justify-center gap-2">
            <span class="inline-block w-2.5 h-2.5 rounded-full bg-emerald-500 animate-pulse"></span>
            磁気カード / NFC (PaSoRi) 受付中
          </div>
        </div>

      {:else if status === 'loading'}
        <!-- 読み取り中 -->
        <div class="loading-text">読み取り中...</div>

      {:else if status === 'success' && currentResult}
        <!-- 正常打刻完了画面 -->
        <div class="result-box animate-scale-up">
          <div class="role-badge">{currentResult.roleName || '利用者'}</div>
          <div class="user-name">{currentResult.userName} 様</div>
          
          {#if currentResult.eventType === 'entry'}
            <div class="status-box type-entry">入室</div>
          {:else}
            <div class="status-box type-exit">退室</div>
          {/if}

          <div class="info-text">打刻時刻: {currentResult.timestamp}</div>
          
          {#if currentResult.eventType === 'exit' && currentResult.durationText}
            <div class="info-text" style="color: #e67e22; margin-top: 15px;">
              滞在時間: {currentResult.durationText}
            </div>
          {/if}
        </div>

      {:else if status === 'error' && currentResult}
        <!-- エラー画面 -->
        <div class="result-box animate-scale-up">
          <div class="status-box type-error">エラー</div>
          <div class="user-name" style="font-size: 2.2em; color: #e74c3c;">
            {currentResult.errorMessage || '認証に失敗しました'}
          </div>
          {#if currentResult.cardId}
            <div class="info-text" style="font-size: 1.4em; color: #7f8c8d; font-family: monospace;">
              カードID: {currentResult.cardId}
            </div>
          {/if}
        </div>
      {/if}
    </div>
  </div>
</div>

<style>
  .kiosk-container {
    height: 100vh;
    width: 100vw;
    margin: 0;
    padding: 0;
    overflow: hidden;
    background-color: #f4f7f9;
    font-family: 'Helvetica Neue', Arial, sans-serif;
    display: flex;
    justify-content: center;
    align-items: center;
    cursor: pointer;
    position: relative;
    user-select: none;
  }

  .admin-switch-btn {
    position: absolute;
    top: 20px;
    right: 20px;
    background: rgba(44, 62, 80, 0.08);
    hover: background 0.2s;
    border: 1px solid rgba(44, 62, 80, 0.15);
    color: #2c3e50;
    padding: 8px 16px;
    border-radius: 20px;
    font-size: 0.9em;
    font-weight: bold;
    cursor: pointer;
    transition: all 0.2s;
    z-index: 10;
  }

  .admin-switch-btn:hover {
    background: rgba(44, 62, 80, 0.18);
    transform: translateY(-1px);
  }

  .card {
    background: white;
    width: 95%;
    max-width: 800px;
    padding: 60px;
    border-radius: 40px;
    box-shadow: 0 20px 50px rgba(0, 0, 0, 0.15);
    text-align: center;
  }

  #display {
    min-height: 500px;
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
  }

  .title-text {
    color: #2c3e50;
    margin: 0 0 30px 0;
    font-size: 2.5em;
    font-weight: bold;
  }

  .role-badge {
    display: inline-block;
    background: #2c3e50;
    color: white;
    padding: 10px 30px;
    border-radius: 50px;
    font-size: 1.6em;
    margin-bottom: 20px;
  }

  .user-name {
    font-size: 3.5em;
    font-weight: bold;
    color: #333;
    margin: 15px 0;
  }

  .status-box {
    font-size: 6em;
    font-weight: 900;
    padding: 25px 0;
    border-radius: 30px;
    margin: 25px 0;
    width: 100%;
    letter-spacing: 5px;
  }

  .info-text {
    font-size: 2em;
    color: #2c3e50;
    font-weight: bold;
    margin-top: 10px;
  }

  .type-entry {
    background-color: #3498db;
    color: white;
  }

  .type-exit {
    background-color: #e67e22;
    color: white;
  }

  .type-error {
    background-color: #e74c3c;
    color: white;
  }

  .loading-text {
    font-size: 3.5em;
    color: #34495e;
    font-weight: bold;
    animation: blink 0.8s infinite;
  }

  @keyframes blink {
    0%, 100% {
      opacity: 1;
    }
    50% {
      opacity: 0.2;
    }
  }

  #swipeInput {
    position: absolute;
    left: -9999px;
    opacity: 0;
  }

  .result-box {
    width: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
  }

  .animate-scale-up {
    animation: scaleUp 0.3s cubic-bezier(0.16, 1, 0.3, 1);
  }

  @keyframes scaleUp {
    0% {
      opacity: 0;
      transform: scale(0.92);
    }
    100% {
      opacity: 1;
      transform: scale(1);
    }
  }

  .animate-fade {
    animation: fadeIn 0.4s ease-out;
  }

  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
  }
</style>
