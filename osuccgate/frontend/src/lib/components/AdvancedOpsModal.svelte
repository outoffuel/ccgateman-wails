<script>
  import { onMount } from 'svelte';
  import { 
    DeleteUser, 
    ClearAllLogs, 
    ForceExitAllInside, 
    ExportFiscalYearLogsCSV,
    ImportUsersCSV,
    ImportLogsCSV,
    GetAppVersion,
    CheckUpdate,
    ApplyUpdate
  } from '../adminApi.js';

  export let onClose = () => {};
  export let onRefresh = () => {};

  let isConfirmed = false; // 「継続」を押したか
  let exportYear = new Date().getFullYear();
  let deleteCardId = '';
  let statusMessage = '';
  let isError = false;

  // システムアップデート状態
  let currentVersion = 'dev';
  let isCheckingUpdate = false;
  let isApplyingUpdate = false;
  let updateInfo = null;
  let updateCheckMessage = '';

  onMount(async () => {
    try {
      const v = await GetAppVersion();
      if (v) currentVersion = v;
    } catch (e) {
      console.warn('Failed to get app version:', e);
    }
  });

  async function handleCheckUpdate() {
    isCheckingUpdate = true;
    updateCheckMessage = '';
    isError = false;
    updateInfo = null;
    try {
      const info = await CheckUpdate();
      updateInfo = info;
      if (info && info.hasUpdate) {
        updateCheckMessage = `新しいバージョンが見つかりました: ${info.version}`;
      } else {
        updateCheckMessage = `お使いのバージョン (${currentVersion}) は最新です。`;
      }
    } catch (err) {
      isError = true;
      updateCheckMessage = '更新の確認に失敗しました: ' + (err?.message || err);
    } finally {
      isCheckingUpdate = false;
    }
  }

  async function handleApplyUpdate() {
    if (!confirm(`バージョン ${updateInfo?.version || '最新版'} にアップデートして再起動しますか？\n（更新ファイルのダウンロード後、システムが自動的に再起動します）`)) {
      return;
    }
    isApplyingUpdate = true;
    updateCheckMessage = 'アップデートをダウンロード・適用中... しばらくお待ちください';
    try {
      await ApplyUpdate();
      updateCheckMessage = 'アップデートが完了しました。再起動しています...';
    } catch (err) {
      isError = true;
      updateCheckMessage = 'アップデートに失敗しました: ' + (err?.message || err);
      isApplyingUpdate = false;
    }
  }

  function handleConfirm() {
    isConfirmed = true;
  }

  async function handleImportUsers() {
    statusMessage = '利用者CSVを読み込み中...';
    isError = false;
    try {
      const msg = await ImportUsersCSV();
      if (msg) {
        statusMessage = msg;
        onRefresh();
      } else {
        statusMessage = '利用者CSVのインポートがキャンセルされました';
      }
    } catch (err) {
      isError = true;
      statusMessage = '利用者CSVのインポートに失敗しました: ' + err;
    }
  }

  async function handleImportLogs() {
    statusMessage = '入退室ログCSVを読み込み中...';
    isError = false;
    try {
      const msg = await ImportLogsCSV();
      if (msg) {
        statusMessage = msg;
        onRefresh();
      } else {
        statusMessage = '入退室ログCSVのインポートがキャンセルされました';
      }
    } catch (err) {
      isError = true;
      statusMessage = '入退室ログCSVのインポートに失敗しました: ' + err;
    }
  }

  async function handleExportCSV() {
    statusMessage = 'CSV出力中...';
    isError = false;
    try {
      const savedPath = await ExportFiscalYearLogsCSV(Number(exportYear));
      if (savedPath) {
        statusMessage = `CSVファイルを保存しました:\n${savedPath}`;
      } else {
        statusMessage = 'エクスポートがキャンセルされました';
      }
    } catch (err) {
      isError = true;
      statusMessage = 'CSV出力に失敗しました: ' + err;
    }
  }

  async function handleForceExitAll() {
    if (!confirm('現在在室中のすべての利用者を強制退室処理しますか？\n（ログには「強制退室」として記録されます）')) {
      return;
    }

    try {
      const count = await ForceExitAllInside();
      statusMessage = `${count} 名の利用者を強制退室処理しました`;
      onRefresh();
    } catch (err) {
      isError = true;
      statusMessage = '強制退室処理に失敗しました: ' + err;
    }
  }

  async function handleDeleteUser() {
    const id = deleteCardId.trim();
    if (!id) {
      isError = true;
      statusMessage = '削除するカードIDまたは学籍番号を入力してください';
      return;
    }

    if (!confirm(`本当にID: ${id} の利用者を削除しますか？\n（関連する打刻ログも削除されます）`)) {
      return;
    }

    try {
      await DeleteUser(id);
      deleteCardId = '';
      statusMessage = `ID: ${id} を削除しました`;
      onRefresh();
    } catch (err) {
      isError = true;
      statusMessage = '削除に失敗しました: ' + err;
    }
  }

  async function handleClearAllLogs() {
    if (!confirm('【重大な警告】\nすべての入退室ログ履歴を完全に削除します。\nこの操作は元に戻せません。本当に実行しますか？')) {
      return;
    }

    try {
      await ClearAllLogs();
      statusMessage = 'すべての入退室ログを削除しました';
      onRefresh();
    } catch (err) {
      isError = true;
      statusMessage = 'ログ削除に失敗しました: ' + err;
    }
  }
</script>

<!-- M3 Dialog Scrim (No backdrop-blur for smooth rendering) -->
<div 
  class="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50 animate-fade"
  on:click|self={onClose}
>
  <!-- M3 Dialog Container (rounded-[28px]) -->
  <div 
    class="bg-m3-surface-container-lowest border border-m3-outline-variant rounded-[28px] p-7 max-w-xl w-full shadow-m3-3"
    on:click|stopPropagation
  >
    <!-- ヘッダー -->
    <div class="flex items-center justify-between pb-4 mb-4 border-b border-m3-outline-variant">
      <div class="flex items-center gap-3.5">
        <div class="w-11 h-11 rounded-2xl bg-m3-error-container text-m3-on-error-container flex items-center justify-center text-xl">
          ⚠️
        </div>
        <div>
          <h3 class="text-lg font-bold text-m3-on-surface">高度な操作</h3>
          <p class="text-xs text-m3-on-surface-variant font-medium">システム設定・CSV移行・ログ管理・破壊的操作</p>
        </div>
      </div>
      <button on:click={onClose} class="text-m3-on-surface-variant hover:text-m3-on-surface hover:bg-m3-surface-container-high p-2 rounded-full transition-colors duration-150">
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
        </svg>
      </button>
    </div>

    {#if !isConfirmed}
      <!-- 警告確認画面 (M3 Error Container) -->
      <div class="bg-m3-error-container/70 border border-m3-error/25 rounded-3xl p-6 mb-6 text-left">
        <p class="text-m3-on-error-container text-sm font-bold leading-relaxed mb-2.5">
          【重要】システムの管理者及びリーダーに了承を得て操作を行ってください。
        </p>
        <p class="text-m3-on-error-container/80 text-xs leading-relaxed">
          このメニューでは登録データの削除、ログの完全消去、手動一斉強制退室、および年度別アーカイブの書き出しを実行できます。誤った操作によるデータ消失にご注意ください。
        </p>
      </div>

      <div class="flex items-center justify-end gap-3">
        <button 
          type="button" 
          on:click={onClose}
          class="m3-btn-outlined py-3 px-6 text-sm"
        >
          中止（戻る）
        </button>
        <button 
          type="button" 
          on:click={handleConfirm}
          class="inline-flex items-center justify-center rounded-full bg-m3-error hover:brightness-105 text-m3-on-error font-bold py-3 px-6 text-sm transition shadow-m3-1"
        >
          了承して継続
        </button>
      </div>

    {:else}
      <!-- 操作パネル -->
      <div class="space-y-4 mb-5 text-left max-h-[68vh] overflow-y-auto pr-1">
        {#if statusMessage}
          <div class={`p-3.5 rounded-2xl text-xs font-mono whitespace-pre-line font-semibold ${isError ? 'bg-m3-error-container text-m3-on-error-container' : 'bg-m3-tertiary-container text-m3-on-tertiary-container'}`}>
            {statusMessage}
          </div>
        {/if}

        <!-- 0. システムアップデート (M3 Primary Container Tonal Section) -->
        <div class="p-5 bg-m3-surface-container-low rounded-3xl border border-m3-outline-variant/60 space-y-3">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <div class="font-bold text-m3-primary text-sm flex items-center gap-2">
              <span>🔄</span> システムアップデート (自己更新)
            </div>
            <div class="text-xs text-m3-on-primary-container font-mono bg-m3-primary-container px-3 py-1 rounded-full font-bold">
              現在のバージョン: {currentVersion}
            </div>
          </div>
          <p class="text-xs text-m3-on-surface-variant">
            GitHub Releases から最新リリースを確認し、自動で更新・再起動を行います（毎日 23:05 にも自動更新チェックが実行されます）。
          </p>

          {#if updateCheckMessage}
            <div class={`p-3 rounded-2xl text-xs font-mono whitespace-pre-line font-semibold ${isError ? 'bg-m3-error-container text-m3-on-error-container' : (updateInfo?.hasUpdate ? 'bg-m3-primary-container text-m3-on-primary-container' : 'bg-m3-surface-container-highest text-m3-on-surface')}`}>
              {updateCheckMessage}
            </div>
          {/if}

          <div class="flex flex-wrap items-center gap-3 pt-1">
            <button 
              type="button"
              on:click={handleCheckUpdate}
              disabled={isCheckingUpdate || isApplyingUpdate}
              class="m3-btn-tonal py-2 px-4 text-xs"
            >
              <svg class={`w-3.5 h-3.5 ${isCheckingUpdate ? 'animate-spin' : ''}`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
              <span>{isCheckingUpdate ? '確認中...' : '更新を確認'}</span>
            </button>

            {#if updateInfo && updateInfo.hasUpdate}
              <button 
                type="button"
                on:click={handleApplyUpdate}
                disabled={isApplyingUpdate}
                class="inline-flex items-center gap-1.5 bg-m3-tertiary hover:brightness-105 disabled:opacity-50 text-m3-on-tertiary text-xs font-bold py-2 px-5 rounded-full transition shadow-m3-1"
              >
                <span>🚀</span>
                <span>{isApplyingUpdate ? '更新中 (再起動待機)...' : `今すぐ更新して再起動 (${updateInfo.version})`}</span>
              </button>
            {/if}
          </div>
        </div>

        <!-- 1. CSVインポート (旧システムからのデータ移行) -->
        <div class="p-5 bg-m3-surface-container-low rounded-3xl border border-m3-outline-variant/60 space-y-3">
          <div class="font-bold text-m3-on-surface text-sm flex items-center gap-2">
            <span>📥</span> CSVデータインポート (旧システム移行)
          </div>
          <p class="text-xs text-m3-on-surface-variant">
            旧システム等で管理していた利用者マスターや入退室ログのCSVファイルを一括取り込みします。
          </p>
          
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-1">
            <!-- 利用者CSVインポート -->
            <div class="bg-m3-surface-container-lowest p-4 rounded-2xl border border-m3-outline-variant/50 flex flex-col justify-between gap-3">
              <div>
                <div class="text-xs font-bold text-m3-on-surface flex items-center gap-1.5">
                  <span>👥</span> 利用者登録マスター CSV
                </div>
                <p class="text-[11px] text-m3-on-surface-variant mt-1 leading-relaxed">
                  管理番号、登録日、ID、氏名、フリガナ、性別、区分コード(1:学生 9:学生スタッフ 0:教職員)、利用目的、連絡先を取り込みます。
                </p>
              </div>
              <button 
                on:click={handleImportUsers}
                class="w-full m3-btn-filled py-2 px-3 text-xs"
              >
                利用者CSVをインポート
              </button>
            </div>

            <!-- ログCSVインポート -->
            <div class="bg-m3-surface-container-lowest p-4 rounded-2xl border border-m3-outline-variant/50 flex flex-col justify-between gap-3">
              <div>
                <div class="text-xs font-bold text-m3-on-surface flex items-center gap-1.5">
                  <span>📋</span> 入退室ログ CSV
                </div>
                <p class="text-[11px] text-m3-on-surface-variant mt-1 leading-relaxed">
                  過去の入退室打刻日時、種別（入室/退室/強制退室）等の履歴を取り込みます。
                </p>
              </div>
              <button 
                on:click={handleImportLogs}
                class="w-full m3-btn-tonal py-2 px-3 text-xs"
              >
                ログCSVをインポート
              </button>
            </div>
          </div>
        </div>

        <!-- 2. 年度別入退室ログ出力 -->
        <div class="p-5 bg-m3-surface-container-low rounded-3xl border border-m3-outline-variant/60">
          <div class="font-bold text-m3-on-surface text-sm mb-1 flex items-center gap-2">
            <span>📊</span> 年度別 入退室ログ出力 (CSV)
          </div>
          <p class="text-xs text-m3-on-surface-variant mb-3">
            指定した年度（4月1日〜翌年3月31日）のログをCSV形式でダウンロード・保存します。
          </p>
          <div class="flex gap-2.5">
            <input 
              type="number" 
              bind:value={exportYear} 
              min="2000" 
              max="2100" 
              class="m3-input-outlined w-32 font-mono py-2"
            />
            <button 
              on:click={handleExportCSV}
              class="inline-flex items-center justify-center bg-m3-tertiary hover:brightness-105 text-m3-on-tertiary text-xs font-bold px-5 py-2 rounded-full transition shadow-m3-1"
            >
              CSVエクスポート保存
            </button>
          </div>
        </div>

        <!-- 3. 手動一斉強制退室 -->
        <div class="p-5 bg-m3-warning-container/40 rounded-3xl border border-m3-warning/25">
          <div class="font-bold text-m3-on-warning-container text-sm mb-1 flex items-center gap-2">
            <span>🚪</span> 在室者の一括強制退室 (手動実行)
          </div>
          <p class="text-xs text-m3-on-warning-container/80 mb-3">
            現在在室中のすべての利用者を即座に強制退室させます（毎日23:00に自動実行される処理の手動トリガーです）。
          </p>
          <button 
            on:click={handleForceExitAll}
            class="w-full bg-m3-warning hover:brightness-105 text-m3-on-warning text-xs font-bold py-2.5 px-4 rounded-full transition shadow-m3-1"
          >
            在室中メンバーを一括強制退室
          </button>
        </div>

        <!-- 4. 登録者の削除 -->
        <div class="p-5 bg-m3-surface-container-low rounded-3xl border border-m3-outline-variant/60">
          <div class="font-bold text-m3-error text-sm mb-1 flex items-center gap-2">
            <span>🗑️</span> 登録利用者の削除
          </div>
          <p class="text-xs text-m3-on-surface-variant mb-3">
            削除対象の識別カードID（磁気またはNFC IDm）を指定して削除します。
          </p>
          <div class="flex gap-2.5">
            <input 
              type="text" 
              bind:value={deleteCardId} 
              placeholder="カードIDまたは学籍番号を入力" 
              class="m3-input-outlined flex-1 font-mono py-2"
            />
            <button 
              on:click={handleDeleteUser}
              class="bg-m3-error hover:brightness-105 text-m3-on-error text-xs font-bold px-5 py-2 rounded-full transition shadow-m3-1 whitespace-nowrap"
            >
              削除実行
            </button>
          </div>
        </div>

        <!-- 5. 入退室ログ全削除 -->
        <div class="p-5 bg-m3-error-container/50 rounded-3xl border border-m3-error/30">
          <div class="font-bold text-m3-on-error-container text-sm mb-1 flex items-center gap-2">
            <span>💣</span> 全入退室ログの完全初期化
          </div>
          <p class="text-xs text-m3-on-error-container/80 mb-3">
            蓄積されたすべての入退室ログを削除します（利用者マスターデータは残ります）。
          </p>
          <button 
            on:click={handleClearAllLogs}
            class="w-full bg-m3-error-container hover:brightness-95 border border-m3-error/40 text-m3-on-error-container text-xs font-bold py-2.5 px-4 rounded-full transition"
          >
            全ログ履歴を消去する
          </button>
        </div>
      </div>

      <div class="flex justify-end pt-3 border-t border-m3-outline-variant/40">
        <button 
          on:click={onClose}
          class="m3-btn-outlined py-2 px-6 text-xs"
        >
          閉じる
        </button>
      </div>
    {/if}

  </div>
</div>

<style>
  .animate-fade {
    animation: fadeIn 0.2s cubic-bezier(0.2, 0, 0, 1);
  }
  @keyframes fadeIn {
    from { opacity: 0; transform: scale(0.97); }
    to { opacity: 1; transform: scale(1); }
  }
</style>
