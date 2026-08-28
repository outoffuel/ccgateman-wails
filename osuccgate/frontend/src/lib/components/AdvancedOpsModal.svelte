<script>
  import { 
    DeleteUser, 
    ClearAllLogs, 
    ForceExitAllInside, 
    ExportFiscalYearLogsCSV,
    ImportUsersCSV,
    ImportLogsCSV
  } from '../../../wailsjs/go/main/App.js';

  export let onClose = () => {};
  export let onRefresh = () => {};

  let isConfirmed = false; // 「継続」を押したか
  let exportYear = new Date().getFullYear();
  let deleteCardId = '';
  let statusMessage = '';
  let isError = false;

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

<div class="fixed inset-0 bg-slate-950/85 backdrop-blur-md flex items-center justify-center p-4 z-50 animate-fade">
  <div class="bg-slate-900 border border-rose-700/60 rounded-3xl p-6 max-w-xl w-full shadow-2xl">
    
    <!-- ヘッダー -->
    <div class="flex items-center gap-3 text-rose-400 mb-4">
      <span class="text-3xl">⚠️</span>
      <div>
        <h3 class="text-xl font-bold text-white">高度な操作</h3>
        <p class="text-xs text-rose-300">システム設定・ログ管理・破壊的操作</p>
      </div>
    </div>

    {#if !isConfirmed}
      <!-- 警告確認画面 -->
      <div class="bg-rose-950/50 border border-rose-800/80 rounded-2xl p-5 mb-6 text-left">
        <p class="text-rose-200 text-base font-bold leading-relaxed mb-3">
          【重要】システムの管理者及びリーダーに了承を得て操作を行ってください。
        </p>
        <p class="text-rose-300/80 text-xs leading-relaxed">
          このメニューでは登録データの削除、ログの完全消去、手動一斉強制退室、および年度別アーカイブの書き出しを実行できます。誤った操作によるデータ消失にご注意ください。
        </p>
      </div>

      <div class="grid grid-cols-2 gap-4">
        <button 
          type="button" 
          on:click={onClose}
          class="bg-slate-800 hover:bg-slate-700 text-slate-300 font-bold py-3 px-4 rounded-xl text-sm transition"
        >
          中止（戻る）
        </button>
        <button 
          type="button" 
          on:click={handleConfirm}
          class="bg-rose-600 hover:bg-rose-500 text-white font-bold py-3 px-4 rounded-xl text-sm transition shadow-lg shadow-rose-600/30"
        >
          了承して継続
        </button>
      </div>

    {:else}
      <!-- 操作パネル -->
      <div class="space-y-4 mb-6 text-left max-h-[70vh] overflow-y-auto pr-1">
        {#if statusMessage}
          <div class={`p-3 rounded-xl text-xs font-mono whitespace-pre-line ${isError ? 'bg-rose-950/80 border border-rose-800 text-rose-300' : 'bg-emerald-950/80 border border-emerald-800 text-emerald-300'}`}>
            {statusMessage}
          </div>
        {/if}

        <!-- 1. CSVインポート (旧システムからのデータ移行) -->
        <div class="p-4 bg-slate-800/60 rounded-2xl border border-slate-700/60 space-y-3">
          <div class="font-bold text-blue-400 text-sm flex items-center gap-2">
            <span>📥</span> CSVデータインポート (旧システム移行)
          </div>
          <p class="text-xs text-slate-400">
            旧システム等で管理していた利用者マスターや入退室ログのCSVファイルを一括取り込みします。
          </p>
          
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-1">
            <!-- 利用者CSVインポート -->
            <div class="bg-slate-900/80 p-3 rounded-xl border border-slate-800 flex flex-col justify-between gap-2">
              <div>
                <div class="text-xs font-bold text-white flex items-center gap-1.5">
                  <span>👥</span> 利用者登録マスター CSV
                </div>
                <p class="text-[11px] text-slate-400 mt-1">
                  学籍番号、氏名、フリガナ、性別、区分、カードIDを取り込みます。
                </p>
              </div>
              <button 
                on:click={handleImportUsers}
                class="w-full bg-blue-600 hover:bg-blue-500 text-white text-xs font-bold py-2 px-3 rounded-lg transition shadow-sm"
              >
                利用者CSVをインポート
              </button>
            </div>

            <!-- ログCSVインポート -->
            <div class="bg-slate-900/80 p-3 rounded-xl border border-slate-800 flex flex-col justify-between gap-2">
              <div>
                <div class="text-xs font-bold text-white flex items-center gap-1.5">
                  <span>📋</span> 入退室ログ CSV
                </div>
                <p class="text-[11px] text-slate-400 mt-1">
                  過去の入退室打刻日時、種別（入室/退室/強制退室）等の履歴を取り込みます。
                </p>
              </div>
              <button 
                on:click={handleImportLogs}
                class="w-full bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-bold py-2 px-3 rounded-lg transition shadow-sm"
              >
                ログCSVをインポート
              </button>
            </div>
          </div>
        </div>

        <!-- 2. 年度別入退室ログ出力 -->
        <div class="p-4 bg-slate-800/60 rounded-2xl border border-slate-700/60">
          <div class="font-bold text-white text-sm mb-1 flex items-center gap-2">
            <span>📊</span> yyyy年度 入退室ログ出力 (CSV)
          </div>
          <p class="text-xs text-slate-400 mb-3">
            指定した年度（4月1日〜翌年3月31日）のログをCSV形式でダウンロード・保存します。
          </p>
          <div class="flex gap-2">
            <input 
              type="number" 
              bind:value={exportYear} 
              min="2000" 
              max="2100" 
              class="bg-slate-900 border border-slate-700 text-white rounded-xl px-3 py-2 text-sm w-32 font-mono"
            />
            <button 
              on:click={handleExportCSV}
              class="bg-emerald-600 hover:bg-emerald-500 text-white text-sm font-semibold px-4 py-2 rounded-xl transition shadow-sm"
            >
              CSVエクスポート保存
            </button>
          </div>
        </div>

        <!-- 3. 手動一斉強制退室 -->
        <div class="p-4 bg-slate-800/60 rounded-2xl border border-slate-700/60">
          <div class="font-bold text-amber-300 text-sm mb-1 flex items-center gap-2">
            <span>🚪</span> 在室者の一括強制退室 (手動実行)
          </div>
          <p class="text-xs text-slate-400 mb-3">
            現在在室中のすべての利用者を即座に強制退室させます（毎日23:00に自動実行される処理の手動トリガーです）。
          </p>
          <button 
            on:click={handleForceExitAll}
            class="w-full bg-amber-700 hover:bg-amber-600 text-white text-sm font-bold py-2.5 px-4 rounded-xl transition shadow-sm"
          >
            在室中メンバーを一括強制退室
          </button>
        </div>

        <!-- 3. 登録者の削除 -->
        <div class="p-4 bg-slate-800/60 rounded-2xl border border-slate-700/60">
          <div class="font-bold text-rose-300 text-sm mb-1 flex items-center gap-2">
            <span>🗑️</span> 登録利用者の削除
          </div>
          <p class="text-xs text-slate-400 mb-2">
            削除対象の識別カードID（磁気またはNFC IDm）を指定して削除します。
          </p>
          <div class="flex gap-2">
            <input 
              type="text" 
              bind:value={deleteCardId} 
              placeholder="カードIDを入力" 
              class="bg-slate-900 border border-slate-700 text-white rounded-xl px-3 py-2 text-sm flex-1 font-mono"
            />
            <button 
              on:click={handleDeleteUser}
              class="bg-rose-700 hover:bg-rose-600 text-white text-sm font-semibold px-4 py-2 rounded-xl transition"
            >
              削除実行
            </button>
          </div>
        </div>

        <!-- 4. 入退室ログ全削除 -->
        <div class="p-4 bg-slate-800/60 rounded-2xl border border-slate-700/60">
          <div class="font-bold text-rose-400 text-sm mb-1 flex items-center gap-2">
            <span>💣</span> 全入退室ログの完全初期化
          </div>
          <p class="text-xs text-slate-400 mb-3">
            蓄積されたすべての入退室ログを削除します（利用者マスターデータは残ります）。
          </p>
          <button 
            on:click={handleClearAllLogs}
            class="w-full bg-rose-950/80 hover:bg-rose-900 border border-rose-700 text-rose-200 text-sm font-bold py-2.5 px-4 rounded-xl transition"
          >
            全ログ履歴を消去する
          </button>
        </div>
      </div>

      <div class="flex justify-end pt-2 border-t border-slate-800">
        <button 
          on:click={onClose}
          class="bg-slate-800 hover:bg-slate-700 text-slate-300 font-semibold px-5 py-2 rounded-xl text-sm transition"
        >
          閉じる
        </button>
      </div>
    {/if}

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
