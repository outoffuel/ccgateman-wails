<script>
  import { onMount } from 'svelte';
  import { sounds } from '../audio.js';
  import { ProcessSwipe } from '../../../wailsjs/go/main/App.js';
  import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime.js';

  export let onOpenAdmin = () => {};
  export let isPaused = false;

  let swipeInput;
  let inputValue = '';
  let status = 'idle'; // 'idle' | 'loading' | 'success' | 'error'
  let currentResult = null;
  let resetTimeout = null;

  // 入力欄に常時フォーカスを当てる（一時停止中はフォーカスを奪わない）
  export function focusInput() {
    if (isPaused) return;
    if (swipeInput) {
      swipeInput.focus();
    }
  }

  // isPausedが解除されたらフォーカスを復帰
  $: if (!isPaused) {
    focusInput();
  }

  // 未更新（今年度未登録）エラーかどうかの判定
  $: isRenewalWarning = status === 'error' && currentResult && (
    currentResult.eventType === 'unregistered_renewal' ||
    (currentResult.errorMessage && currentResult.errorMessage.includes('今年度'))
  );

  onMount(() => {
    focusInput();

    // 画面のどこをクリックしてもフォーカスを維持（一時停止中はスキップ）
    const handleClick = () => {
      if (!isPaused) {
        focusInput();
      }
    };
    window.addEventListener('click', handleClick);

    // NFC PaSoRiからのバックグラウンド打刻イベントを受信
    EventsOn('card_swiped', (resp) => {
      if (isPaused) return;
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

<div 
  class="h-screen w-full p-6 bg-m3-surface-container-low flex flex-col justify-center items-center relative cursor-pointer select-none overflow-y-auto"
  on:click={focusInput}
>
  <!-- 右上の管理者画面切替ボタン (M3 Tonal Pill Button) -->
  <button 
    class="absolute top-6 right-6 z-10 inline-flex items-center gap-2 px-5 py-2.5 rounded-full bg-m3-surface-container-highest hover:bg-m3-secondary-container text-m3-on-surface-variant hover:text-m3-on-secondary-container border border-m3-outline-variant text-sm font-bold shadow-m3-1 transition-colors duration-150" 
    on:click|stopPropagation={onOpenAdmin}
    title="管理者画面を開く"
  >
    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
    </svg>
    <span>管理者モード</span>
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

  <!-- メイン M3 Surface Container Card -->
  <div class="w-full max-w-3xl bg-m3-surface-container-lowest rounded-[32px] border border-m3-outline-variant shadow-m3-2 px-10 py-12 sm:px-14 sm:py-14 text-center">
    <!-- ヘッダータイトル & サブタイトル -->
    <div class="inline-flex items-center gap-2 px-4 py-1.5 rounded-full bg-m3-primary-container/60 text-m3-on-primary-container text-xs font-bold tracking-wider uppercase mb-3">
      Create Center Gate System
    </div>
    <h1 class="text-3xl sm:text-4xl font-extrabold text-m3-on-surface tracking-tight mb-8">
      入退室管理システム
    </h1>

    <div id="display" class="min-h-[420px] flex flex-col justify-center items-center">
      {#if status === 'idle'}
        <!-- 待機中画面 (M3 Clean Tonal State) -->
        <div id="content" class="animate-fade flex flex-col items-center w-full max-w-xl">
          <!-- ソフトなアイコンコンテナ -->
          <div class="w-28 h-28 rounded-[28px] bg-m3-primary-container text-m3-primary flex items-center justify-center mb-8 shadow-m3-1">
            <svg class="w-14 h-14" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.8" d="M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z"/>
            </svg>
          </div>

          <div class="text-3xl sm:text-5xl font-extrabold text-m3-on-surface tracking-tight leading-tight">
            学生証を通してください
          </div>
          <div class="text-lg sm:text-2xl text-m3-on-surface-variant font-medium mt-4">
            カードリーダーへタッチまたはスライドしてください
          </div>

          <!-- M3 Status Chip -->
          <div class="mt-10 inline-flex items-center gap-2.5 px-5 py-2.5 rounded-full bg-m3-surface-container-low border border-m3-outline-variant/60 text-m3-on-surface-variant text-sm font-semibold">
            <span class="inline-block w-2.5 h-2.5 rounded-full bg-m3-tertiary animate-pulse"></span>
            <span>磁気カード / NFC (PaSoRi) 読み取り待機中</span>
          </div>
        </div>

      {:else if status === 'loading'}
        <!-- 読み取り中 -->
        <div class="flex flex-col items-center gap-5">
          <div class="w-20 h-20 rounded-full bg-m3-secondary-container text-m3-secondary flex items-center justify-center animate-pulse">
            <svg class="w-10 h-10 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
            </svg>
          </div>
          <div class="text-3xl sm:text-4xl font-bold text-m3-on-surface tracking-wide animate-pulse">
            読み取り中...
          </div>
        </div>

      {:else if status === 'success' && currentResult}
        <!-- 正常打刻完了画面 (M3 Primary / Secondary Container) -->
        <div class="w-full flex flex-col items-center animate-scale-up">
          <!-- 区分チップ (M3 Assist/Suggestion Chip) -->
          <span class="inline-flex items-center px-6 py-2 rounded-full bg-m3-secondary-container text-m3-on-secondary-container text-xl font-bold mb-4">
            {currentResult.roleName || '利用者'}
          </span>

          <div class="text-4xl sm:text-5xl font-extrabold text-m3-on-surface my-2">
            {currentResult.userName} <span class="text-2xl sm:text-3xl font-bold text-m3-on-surface-variant">様</span>
          </div>
          
          {#if currentResult.eventType === 'entry'}
            <!-- 入室: M3 Primary / Tertiary Container -->
            <div class="w-full rounded-3xl bg-m3-primary-container text-m3-on-primary-container border border-m3-primary/20 py-7 px-6 my-6 shadow-m3-1">
              <div class="text-6xl sm:text-7xl font-black tracking-widest">入室</div>
              <div class="text-sm font-bold text-m3-primary mt-2 tracking-wider uppercase"> Checked In </div>
            </div>
          {:else}
            <!-- 退室: M3 Tertiary / Warning Tonal Container -->
            <div class="w-full rounded-3xl bg-m3-warning-container text-m3-on-warning-container border border-m3-warning/25 py-7 px-6 my-6 shadow-m3-1">
              <div class="text-6xl sm:text-7xl font-black tracking-widest">退室</div>
              <div class="text-sm font-bold text-m3-warning mt-2 tracking-wider uppercase"> Checked Out </div>
            </div>
          {/if}

          <!-- 打刻時刻 & 滞在時間チップ -->
          <div class="flex flex-wrap items-center justify-center gap-3 mt-2">
            <div class="inline-flex items-center gap-2 px-5 py-2.5 rounded-2xl bg-m3-surface-container text-m3-on-surface text-xl font-bold">
              <svg class="w-5 h-5 text-m3-on-surface-variant" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
              </svg>
              <span>打刻時刻: {currentResult.timestamp}</span>
            </div>

            {#if currentResult.eventType === 'exit' && currentResult.durationText}
              <div class="inline-flex items-center gap-2 px-5 py-2.5 rounded-2xl bg-m3-secondary-container/70 text-m3-on-secondary-container text-xl font-bold">
                <span>滞在時間: {currentResult.durationText}</span>
              </div>
            {/if}
          </div>
        </div>

      {:else if status === 'error' && currentResult}
        <!-- エラー / 未登録 / 今年度未更新画面 (M3 Warning or Error Container) -->
        <div class="w-full flex flex-col items-center animate-scale-up">
          {#if isRenewalWarning}
            <!-- 今年度未登録 (M3 Warning Container) -->
            <div class="w-full rounded-3xl bg-m3-warning-container text-m3-on-warning-container border border-m3-warning/30 py-7 px-6 mb-6 shadow-m3-1">
              <div class="text-4xl sm:text-5xl font-black tracking-wide">今年度未更新</div>
              <div class="text-sm font-bold text-m3-warning mt-2">RENEWAL REQUIRED</div>
            </div>
            <div class="text-2xl sm:text-3xl font-bold text-m3-on-warning-container leading-relaxed max-w-xl">
              {currentResult.errorMessage}
            </div>
          {:else}
            <!-- 未登録 / 認証エラー (M3 Error Container) -->
            <div class="w-full rounded-3xl bg-m3-error-container text-m3-on-error-container border border-m3-error/30 py-7 px-6 mb-6 shadow-m3-1">
              <div class="text-5xl sm:text-6xl font-black tracking-widest">エラー</div>
              <div class="text-sm font-bold text-m3-error mt-2">AUTHENTICATION FAILED</div>
            </div>
            <div class="text-2xl sm:text-3xl font-bold text-m3-error leading-relaxed max-w-xl">
              {currentResult.errorMessage || '認証に失敗しました'}
            </div>
          {/if}

          {#if currentResult.cardId}
            <div class="mt-6 inline-flex items-center gap-2 px-5 py-2 rounded-full bg-m3-surface-container-high text-m3-on-surface-variant font-mono text-base font-semibold border border-m3-outline-variant/60">
              <span>カードID: {currentResult.cardId}</span>
            </div>
          {/if}
        </div>
      {/if}
    </div>
  </div>
</div>

<style>
  #swipeInput {
    position: absolute;
    left: -9999px;
    opacity: 0;
  }

  .animate-scale-up {
    animation: scaleUp 0.28s cubic-bezier(0.2, 0, 0, 1);
  }

  @keyframes scaleUp {
    0% {
      opacity: 0;
      transform: scale(0.94);
    }
    100% {
      opacity: 1;
      transform: scale(1);
    }
  }

  .animate-fade {
    animation: fadeIn 0.35s cubic-bezier(0.2, 0, 0, 1);
  }

  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
  }
</style>
