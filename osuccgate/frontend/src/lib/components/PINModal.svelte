<script>
  import { onMount, tick } from 'svelte';
  import { VerifyPIN } from '../adminApi.js';

  export let onSuccess = () => {};
  export let onCancel = () => {};

  let pin = '';
  let pinInput;
  let errorMessage = '';
  let isChecking = false;

  onMount(() => {
    tick().then(() => {
      pinInput?.focus();
    });
    const timer = setTimeout(() => {
      pinInput?.focus();
    }, 50);
    return () => clearTimeout(timer);
  });

  async function handleSubmit() {
    if (!pin) {
      errorMessage = 'PINコードを入力してください';
      return;
    }

    isChecking = true;
    errorMessage = '';

    try {
      const ok = await VerifyPIN(pin);
      if (ok) {
        onSuccess(pin);
      } else {
        errorMessage = 'PINコードが正しくありません';
        pin = '';
      }
    } catch (e) {
      errorMessage = '認証エラーが発生しました';
    } finally {
      isChecking = false;
    }
  }

  function handleKeydown(e) {
    if (e.key === 'Enter') {
      handleSubmit();
    } else if (e.key === 'Escape') {
      onCancel();
    }
  }
</script>

<!-- M3 Dialog Scrim & Container -->
<div 
  class="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50 animate-fade"
  on:click|self={onCancel}
>
  <div 
    class="bg-m3-surface-container-lowest border border-m3-outline-variant rounded-[28px] p-8 max-w-md w-full shadow-m3-3 text-center"
    on:click|stopPropagation={() => pinInput?.focus()}
  >
    <!-- M3 Hero Icon in Secondary/Primary Container -->
    <div class="w-16 h-16 bg-m3-primary-container text-m3-on-primary-container rounded-2xl flex items-center justify-center mx-auto mb-5">
      <svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"/>
      </svg>
    </div>
    
    <h2 class="text-2xl font-bold text-m3-on-surface mb-1.5">管理者認証</h2>
    <p class="text-m3-on-surface-variant text-sm mb-6">管理者PINコードを入力してください</p>

    {#if errorMessage}
      <div class="bg-m3-error-container text-m3-on-error-container border border-m3-error/20 text-xs font-semibold py-2.5 px-4 rounded-2xl mb-5">
        {errorMessage}
      </div>
    {/if}

    <!-- M3 Outlined Text Field -->
    <div class="mb-7 text-left">
      <label for="pin-modal-input" class="block text-xs font-bold text-m3-on-surface-variant mb-1.5 ml-1">
        PINコード
      </label>
      <input 
        id="pin-modal-input"
        type="password" 
        bind:this={pinInput}
        bind:value={pin} 
        on:keydown={handleKeydown}
        maxlength="8" 
        placeholder="••••" 
        autofocus
        class="w-full text-center text-3xl tracking-[0.4em] bg-m3-surface-container-low border-2 border-m3-outline text-m3-on-surface rounded-2xl py-3.5 px-4 focus:outline-none focus:border-m3-primary focus:bg-m3-surface-container-lowest transition-colors duration-150 font-mono"
      />
    </div>

    <!-- M3 Dialog Actions (Pill Buttons) -->
    <div class="flex items-center justify-end gap-3">
      <button 
        type="button" 
        on:click={onCancel}
        class="flex-1 bg-m3-surface-container-high hover:bg-m3-surface-container-highest text-m3-on-surface-variant font-bold py-3 px-5 rounded-full text-sm transition-colors duration-150"
      >
        キャンセル
      </button>
      <button 
        type="button" 
        on:click={handleSubmit}
        disabled={isChecking}
        class="flex-1 bg-m3-primary hover:bg-blue-700 text-m3-on-primary font-bold py-3 px-5 rounded-full text-sm transition-colors duration-150 shadow-m3-1 disabled:opacity-50"
      >
        {isChecking ? '確認中...' : '認証する'}
      </button>
    </div>
  </div>
</div>

<style>
  .animate-fade {
    animation: fadeIn 0.2s cubic-bezier(0.2, 0, 0, 1);
  }
  @keyframes fadeIn {
    from { opacity: 0; transform: scale(0.96); }
    to { opacity: 1; transform: scale(1); }
  }
</style>
