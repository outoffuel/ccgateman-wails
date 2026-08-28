<script>
  import { VerifyPIN } from '../../../wailsjs/go/main/App.js';

  export let onSuccess = () => {};
  export let onCancel = () => {};

  let pin = '';
  let errorMessage = '';
  let isChecking = false;

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

<div class="fixed inset-0 bg-slate-950/80 backdrop-blur-md flex items-center justify-center p-4 z-50 animate-fade">
  <div class="bg-slate-900 border border-slate-700 rounded-3xl p-8 max-w-md w-full shadow-2xl text-center">
    <div class="w-16 h-16 bg-blue-600/20 text-blue-400 rounded-2xl flex items-center justify-center mx-auto mb-4 text-3xl shadow-inner">
      🔒
    </div>
    
    <h2 class="text-2xl font-bold text-white mb-2">管理者認証</h2>
    <p class="text-slate-400 text-sm mb-6">4桁の管理者PINコードを入力してください (初期値: 1234)</p>

    {#if errorMessage}
      <div class="bg-rose-950/60 border border-rose-800 text-rose-300 text-xs py-2 px-3 rounded-xl mb-4">
        {errorMessage}
      </div>
    {/if}

    <div class="mb-6">
      <input 
        type="password" 
        bind:value={pin} 
        on:keydown={handleKeydown}
        maxlength="8" 
        placeholder="••••" 
        class="w-full text-center text-3xl tracking-widest bg-slate-800 border border-slate-700 text-white rounded-2xl py-3 px-4 focus:outline-none focus:border-blue-500 transition shadow-inner font-mono"
        autofocus
      />
    </div>

    <div class="grid grid-cols-2 gap-3">
      <button 
        type="button" 
        on:click={onCancel}
        class="bg-slate-800 hover:bg-slate-700 text-slate-300 font-semibold py-3 px-4 rounded-xl text-sm transition"
      >
        キャンセル
      </button>
      <button 
        type="button" 
        on:click={handleSubmit}
        disabled={isChecking}
        class="bg-blue-600 hover:bg-blue-500 text-white font-bold py-3 px-4 rounded-xl text-sm transition shadow-lg shadow-blue-600/30 disabled:opacity-50"
      >
        {isChecking ? '確認中...' : '認証'}
      </button>
    </div>
  </div>
</div>

<style>
  .animate-fade {
    animation: fadeIn 0.2s ease-out;
  }
  @keyframes fadeIn {
    from { opacity: 0; transform: scale(0.98); }
    to { opacity: 1; transform: scale(1); }
  }
</style>
