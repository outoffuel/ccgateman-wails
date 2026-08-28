<script>
  import { onMount, onDestroy } from 'svelte';
  import { 
    GetDashboardStats, 
    GetAllUsers, 
    SaveUser, 
    DeleteUser, 
    GetRecentLogs, 
    GetCurrentInsideUsers 
  } from '../../../wailsjs/go/main/App.js';
  import AdvancedOpsModal from './AdvancedOpsModal.svelte';

  export let onBackToKiosk = () => {};

  let activeTab = 'logs'; // 'logs' | 'inside' | 'users'
  let stats = { currentInsideCount: 0, totalUserCount: 0, todayLogCount: 0 };
  let logs = [];
  let users = [];
  let insideUsers = [];
  let isAdvancedOpen = false;

  // ユーザー編集/登録用モーダルステート
  let isUserModalOpen = false;
  let editingUser = null;
  let formCardId = '';
  let formName = '';
  let formRoleName = '学生';
  let formRoleCode = 1;
  let formStudentNo = '';
  let formError = '';

  let refreshTimer = null;

  async function loadData() {
    try {
      const [s, l, u, i] = await Promise.all([
        GetDashboardStats(),
        GetRecentLogs(50),
        GetAllUsers(),
        GetCurrentInsideUsers()
      ]);
      stats = s || { currentInsideCount: 0, totalUserCount: 0, todayLogCount: 0 };
      logs = l || [];
      users = u || [];
      insideUsers = i || [];
    } catch (e) {
      console.error('Failed to load dashboard data:', e);
    }
  }

  onMount(() => {
    loadData();
    refreshTimer = setInterval(loadData, 10000);
    return () => {
      if (refreshTimer) clearInterval(refreshTimer);
    };
  });

  function openNewUserModal() {
    editingUser = null;
    formCardId = '';
    formName = '';
    formRoleName = '学生';
    formRoleCode = 1;
    formStudentNo = '';
    formError = '';
    isUserModalOpen = true;
  }

  function openEditUserModal(u) {
    editingUser = u;
    formCardId = u.cardId;
    formName = u.name;
    formRoleName = u.roleName;
    formRoleCode = u.roleCode;
    formStudentNo = u.studentNo || '';
    formError = '';
    isUserModalOpen = true;
  }

  function handleRoleChange() {
    if (formRoleName === '教職員') formRoleCode = 9;
    else if (formRoleName === 'スタッフ') formRoleCode = 5;
    else if (formRoleName === '学生') formRoleCode = 1;
    else formRoleCode = 1;
  }

  async function handleSaveUser() {
    if (!formCardId.trim() || !formName.trim()) {
      formError = 'カードIDと氏名は必須です';
      return;
    }

    try {
      await SaveUser({
        cardId: formCardId.trim(),
        name: formName.trim(),
        roleName: formRoleName,
        roleCode: Number(formRoleCode),
        studentNo: formStudentNo.trim()
      });
      isUserModalOpen = false;
      await loadData();
    } catch (err) {
      formError = '保存に失敗しました: ' + err;
    }
  }

  function formatDate(isoStr) {
    if (!isoStr) return '-';
    const d = new Date(isoStr);
    return d.toLocaleString('ja-JP', { 
      month: '2-digit', 
      day: '2-digit', 
      hour: '2-digit', 
      minute: '2-digit', 
      second: '2-digit' 
    });
  }

  function formatTime(isoStr) {
    if (!isoStr) return '-';
    const d = new Date(isoStr);
    return d.toLocaleTimeString('ja-JP');
  }
</script>

<div class="min-h-screen bg-slate-950 text-slate-100 font-sans pb-12">
  
  <!-- 上部ナビゲーションバー -->
  <header class="bg-slate-900 border-b border-slate-800 px-8 py-4 sticky top-0 z-30 flex flex-wrap items-center justify-between gap-4 shadow-xl">
    <div class="flex items-center gap-3">
      <div class="w-10 h-10 rounded-xl bg-blue-600 flex items-center justify-center font-black text-white text-lg shadow-lg shadow-blue-500/20">
        OG
      </div>
      <div>
        <h1 class="text-lg font-bold text-white leading-tight flex items-center gap-2">
          入退室管理システム
          <span class="text-xs bg-slate-800 text-slate-400 font-mono px-2 py-0.5 rounded-full border border-slate-700">Admin</span>
        </h1>
        <p class="text-xs text-slate-400">LAN公開ポート: 8080 (他PCブラウザからも閲覧可能)</p>
      </div>
    </div>

    <div class="flex items-center gap-3">
      <button 
        on:click={() => isAdvancedOpen = true}
        class="bg-rose-950/60 hover:bg-rose-900 text-rose-300 border border-rose-800/80 px-4 py-2 rounded-xl font-bold text-xs transition flex items-center gap-1.5"
      >
        ⚠️ 高度な操作
      </button>

      <button 
        on:click={loadData}
        class="bg-slate-800 hover:bg-slate-700 text-slate-300 px-3.5 py-2 rounded-xl text-xs font-semibold transition"
      >
        🔄 更新
      </button>

      <button 
        on:click={onBackToKiosk}
        class="bg-blue-600 hover:bg-blue-500 text-white px-4 py-2 rounded-xl text-xs font-bold transition shadow-md shadow-blue-600/30"
      >
        📱 打刻画面に戻る
      </button>
    </div>
  </header>

  <!-- メインコンテンツ -->
  <main class="max-w-7xl mx-auto px-6 pt-6 space-y-6">
    
    <!-- 統計サマリーカード -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
      <!-- 在室者数 -->
      <div class="bg-slate-900/90 border border-slate-800 rounded-3xl p-6 shadow-xl relative overflow-hidden group">
        <div class="absolute -right-4 -bottom-4 w-24 h-24 bg-blue-500/10 rounded-full blur-2xl pointer-events-none"></div>
        <div class="text-slate-400 text-xs font-bold uppercase tracking-wider mb-2">現在の在室者数</div>
        <div class="text-5xl font-black text-blue-400 flex items-baseline gap-2">
          {stats.currentInsideCount}
          <span class="text-lg font-normal text-slate-500">名</span>
        </div>
      </div>

      <!-- 登録ユーザー数 -->
      <div class="bg-slate-900/90 border border-slate-800 rounded-3xl p-6 shadow-xl relative overflow-hidden group">
        <div class="absolute -right-4 -bottom-4 w-24 h-24 bg-emerald-500/10 rounded-full blur-2xl pointer-events-none"></div>
        <div class="text-slate-400 text-xs font-bold uppercase tracking-wider mb-2">登録ユーザー総数</div>
        <div class="text-5xl font-black text-emerald-400 flex items-baseline gap-2">
          {stats.totalUserCount}
          <span class="text-lg font-normal text-slate-500">名</span>
        </div>
      </div>

      <!-- 本日の打刻数 -->
      <div class="bg-slate-900/90 border border-slate-800 rounded-3xl p-6 shadow-xl relative overflow-hidden group">
        <div class="absolute -right-4 -bottom-4 w-24 h-24 bg-amber-500/10 rounded-full blur-2xl pointer-events-none"></div>
        <div class="text-slate-400 text-xs font-bold uppercase tracking-wider mb-2">本日の総打刻回数</div>
        <div class="text-5xl font-black text-amber-400 flex items-baseline gap-2">
          {stats.todayLogCount}
          <span class="text-lg font-normal text-slate-500">回</span>
        </div>
      </div>
    </div>

    <!-- タブヘッダー -->
    <div class="flex items-center justify-between border-b border-slate-800 pb-3 pt-2">
      <div class="flex gap-2">
        <button 
          on:click={() => activeTab = 'logs'}
          class={`px-5 py-2.5 rounded-2xl font-bold text-xs transition ${activeTab === 'logs' ? 'bg-blue-600 text-white shadow-lg shadow-blue-600/30' : 'bg-slate-900 text-slate-400 hover:text-white border border-slate-800'}`}
        >
          📋 入退室ログ ({logs.length})
        </button>
        <button 
          on:click={() => activeTab = 'inside'}
          class={`px-5 py-2.5 rounded-2xl font-bold text-xs transition ${activeTab === 'inside' ? 'bg-blue-600 text-white shadow-lg shadow-blue-600/30' : 'bg-slate-900 text-slate-400 hover:text-white border border-slate-800'}`}
        >
          🟢 在室中メンバー ({insideUsers.length})
        </button>
        <button 
          on:click={() => activeTab = 'users'}
          class={`px-5 py-2.5 rounded-2xl font-bold text-xs transition ${activeTab === 'users' ? 'bg-blue-600 text-white shadow-lg shadow-blue-600/30' : 'bg-slate-900 text-slate-400 hover:text-white border border-slate-800'}`}
        >
          👥 利用者マスター ({users.length})
        </button>
      </div>

      {#if activeTab === 'users'}
        <button 
          on:click={openNewUserModal}
          class="bg-emerald-600 hover:bg-emerald-500 text-white font-bold px-4 py-2 rounded-xl text-xs transition shadow-lg shadow-emerald-600/30 flex items-center gap-1.5"
        >
          ＋ 新規利用者登録
        </button>
      {/if}
    </div>

    <!-- タブボディ -->
    <div class="bg-slate-900/90 border border-slate-800 rounded-3xl p-6 shadow-xl overflow-hidden">
      
      <!-- 1. 入退室ログ -->
      {#if activeTab === 'logs'}
        {#if logs.length === 0}
          <div class="text-center py-16 text-slate-500 text-sm">打刻ログはまだ記録されていません</div>
        {:else}
          <div class="overflow-x-auto">
            <table class="w-full text-left text-sm text-slate-300">
              <thead class="text-xs text-slate-400 uppercase bg-slate-800/60 border-b border-slate-800">
                <tr>
                  <th class="p-3.5 rounded-l-xl">打刻日時</th>
                  <th class="p-3.5">氏名</th>
                  <th class="p-3.5">区分</th>
                  <th class="p-3.5">学籍/職員番号</th>
                  <th class="p-3.5">ステータス</th>
                  <th class="p-3.5 rounded-r-xl">滞在時間</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-slate-800/60">
                {#each logs as log}
                  <tr class="hover:bg-slate-800/40 transition">
                    <td class="p-3.5 font-mono text-xs text-slate-400">{formatDate(log.timestamp)}</td>
                    <td class="p-3.5 font-bold text-white text-base">{log.userName || '未登録'}</td>
                    <td class="p-3.5">
                      <span class={`px-2.5 py-1 rounded-full text-xs font-semibold ${log.roleCode === 9 ? 'bg-purple-950 text-purple-300 border border-purple-800' : 'bg-slate-800 text-slate-300'}`}>
                        {log.roleName || '-'}
                      </span>
                    </td>
                    <td class="p-3.5 font-mono text-slate-300">{log.studentNo || '-'}</td>
                    <td class="p-3.5">
                      <span class={`inline-flex items-center px-3 py-1 rounded-xl text-xs font-black ${log.eventType === 'entry' ? 'bg-blue-950/80 text-blue-400 border border-blue-700/60' : 'bg-amber-950/80 text-amber-400 border border-amber-700/60'}`}>
                        {log.eventType === 'entry' ? '🔵 入室' : '🟠 退室'}
                      </span>
                    </td>
                    <td class="p-3.5 font-mono text-slate-400 font-semibold">{log.durationText || '-'}</td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {/if}

      <!-- 2. 在室中メンバー -->
      {:else if activeTab === 'inside'}
        {#if insideUsers.length === 0}
          <div class="text-center py-16 text-slate-500 text-sm">現在在室中の利用者はいません</div>
        {:else}
          <div class="overflow-x-auto">
            <table class="w-full text-left text-sm text-slate-300">
              <thead class="text-xs text-slate-400 uppercase bg-slate-800/60 border-b border-slate-800">
                <tr>
                  <th class="p-3.5 rounded-l-xl">氏名</th>
                  <th class="p-3.5">区分</th>
                  <th class="p-3.5">学籍/職員番号</th>
                  <th class="p-3.5">入室時刻</th>
                  <th class="p-3.5 rounded-r-xl">現在の滞在時間</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-slate-800/60">
                {#each insideUsers as u}
                  <tr class="hover:bg-slate-800/40 transition">
                    <td class="p-3.5 font-bold text-white text-base flex items-center gap-2">
                      <span class="w-2.5 h-2.5 rounded-full bg-emerald-400 animate-pulse"></span>
                      {u.userName || '未登録'}
                    </td>
                    <td class="p-3.5">
                      <span class="px-2.5 py-1 rounded-full text-xs font-semibold bg-slate-800 text-slate-300">
                        {u.roleName || '-'}
                      </span>
                    </td>
                    <td class="p-3.5 font-mono text-slate-300">{u.studentNo || '-'}</td>
                    <td class="p-3.5 font-mono text-xs text-slate-400">{formatTime(u.lastEventTime)}</td>
                    <td class="p-3.5 font-bold text-blue-400 text-sm">{u.stayDuration || '数秒'}</td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {/if}

      <!-- 3. 利用者マスター -->
      {:else if activeTab === 'users'}
        {#if users.length === 0}
          <div class="text-center py-16 text-slate-500 text-sm">登録されている利用者はまだいません</div>
        {:else}
          <div class="overflow-x-auto">
            <table class="w-full text-left text-sm text-slate-300">
              <thead class="text-xs text-slate-400 uppercase bg-slate-800/60 border-b border-slate-800">
                <tr>
                  <th class="p-3.5 rounded-l-xl">識別ID (磁気/NFC)</th>
                  <th class="p-3.5">氏名</th>
                  <th class="p-3.5">区分 (コード)</th>
                  <th class="p-3.5">学籍/職員番号</th>
                  <th class="p-3.5 text-right rounded-r-xl">操作</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-slate-800/60">
                {#each users as u}
                  <tr class="hover:bg-slate-800/40 transition">
                    <td class="p-3.5 font-mono text-xs text-slate-400">{u.cardId}</td>
                    <td class="p-3.5 font-bold text-white text-base">{u.name}</td>
                    <td class="p-3.5">
                      <span class={`px-2.5 py-1 rounded-full text-xs font-semibold ${u.roleCode === 9 ? 'bg-purple-950 text-purple-300 border border-purple-800' : 'bg-slate-800 text-slate-300'}`}>
                        {u.roleName} ({u.roleCode})
                      </span>
                    </td>
                    <td class="p-3.5 font-mono text-slate-300">{u.studentNo || '-'}</td>
                    <td class="p-3.5 text-right">
                      <button 
                        on:click={() => openEditUserModal(u)}
                        class="bg-slate-800 hover:bg-blue-600 text-blue-400 hover:text-white px-3 py-1 rounded-lg text-xs font-semibold transition"
                      >
                        編集
                      </button>
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {/if}
      {/if}

    </div>
  </main>

  <!-- ユーザー編集/登録モーダル -->
  {#if isUserModalOpen}
    <div class="fixed inset-0 bg-slate-950/85 backdrop-blur-md flex items-center justify-center p-4 z-50 animate-fade">
      <div class="bg-slate-900 border border-slate-700 rounded-3xl p-7 max-w-lg w-full shadow-2xl">
        <h3 class="text-xl font-bold text-white mb-4">
          {editingUser ? '利用者情報の編集' : '新規利用者の登録'}
        </h3>

        {#if formError}
          <div class="bg-rose-950/70 border border-rose-800 text-rose-300 text-xs p-3 rounded-xl mb-4">
            {formError}
          </div>
        {/if}

        <div class="space-y-4">
          <div>
            <label class="block text-xs font-semibold text-slate-400 uppercase mb-1">
              識別カードID (磁気カード番号 / NFC IDm) *
            </label>
            <input 
              type="text" 
              bind:value={formCardId} 
              disabled={!!editingUser}
              placeholder="カードを通すか直接入力"
              class="w-full bg-slate-800 border border-slate-700 text-white rounded-xl px-3.5 py-2.5 font-mono text-sm focus:outline-none focus:border-blue-500 disabled:opacity-50"
            />
          </div>

          <div>
            <label class="block text-xs font-semibold text-slate-400 uppercase mb-1">氏名 *</label>
            <input 
              type="text" 
              bind:value={formName} 
              placeholder="例: 山田 太郎"
              class="w-full bg-slate-800 border border-slate-700 text-white rounded-xl px-3.5 py-2.5 text-sm focus:outline-none focus:border-blue-500"
            />
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-xs font-semibold text-slate-400 uppercase mb-1">区分名</label>
              <select 
                bind:value={formRoleName} 
                on:change={handleRoleChange}
                class="w-full bg-slate-800 border border-slate-700 text-white rounded-xl px-3.5 py-2.5 text-sm focus:outline-none focus:border-blue-500"
              >
                <option value="学生">学生</option>
                <option value="教職員">教職員</option>
                <option value="スタッフ">スタッフ</option>
                <option value="来客">来客</option>
              </select>
            </div>

            <div>
              <label class="block text-xs font-semibold text-slate-400 uppercase mb-1">ロールコード</label>
              <input 
                type="number" 
                bind:value={formRoleCode} 
                class="w-full bg-slate-800 border border-slate-700 text-white rounded-xl px-3.5 py-2.5 text-sm font-mono focus:outline-none focus:border-blue-500"
              />
            </div>
          </div>

          <div>
            <label class="block text-xs font-semibold text-slate-400 uppercase mb-1">学籍番号 / 職員番号 (任意)</label>
            <input 
              type="text" 
              bind:value={formStudentNo} 
              placeholder="例: B2026001"
              class="w-full bg-slate-800 border border-slate-700 text-white rounded-xl px-3.5 py-2.5 font-mono text-sm focus:outline-none focus:border-blue-500"
            />
          </div>
        </div>

        <div class="flex justify-end gap-3 pt-6 border-t border-slate-800 mt-6">
          <button 
            type="button" 
            on:click={() => isUserModalOpen = false}
            class="px-5 py-2.5 rounded-xl text-xs font-semibold text-slate-400 hover:text-white"
          >
            キャンセル
          </button>
          <button 
            type="button" 
            on:click={handleSaveUser}
            class="bg-blue-600 hover:bg-blue-500 text-white font-bold px-6 py-2.5 rounded-xl text-xs transition shadow-lg shadow-blue-600/30"
          >
            保存
          </button>
        </div>
      </div>
    </div>
  {/if}

  <!-- 高度な操作モーダル -->
  {#if isAdvancedOpen}
    <AdvancedOpsModal 
      onClose={() => isAdvancedOpen = false}
      onRefresh={loadData}
    />
  {/if}

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
