<script>
  import { onMount, onDestroy } from 'svelte';
  import { 
    GetDashboardStats, 
    GetMonthlyStats,
    GetAllUsers, 
    SaveUser, 
    DeleteUser, 
    GetRecentLogs, 
    GetCurrentInsideUsers,
    ToggleFullscreen,
    GetAppVersion
  } from '../adminApi.js';
  import { formatLogTime } from '../utils/date.js';

  import DashboardTab from './admin/DashboardTab.svelte';
  import LogsTab from './admin/LogsTab.svelte';
  import InsideTab from './admin/InsideTab.svelte';
  import UsersTab from './admin/UsersTab.svelte';
  import UserModal from './admin/UserModal.svelte';
  import AdvancedOpsModal from './AdvancedOpsModal.svelte';

  export let onBackToKiosk = () => {};

  // 画面タブ ('dashboard' | 'logs' | 'inside' | 'users')
  let activeTab = 'dashboard';

  // バックエンドデータ状態
  let stats = { currentInsideCount: 0, totalUserCount: 0, todayLogCount: 0, weekLogCount: 0, monthLogCount: 0, yearLogCount: 0 };
  let logs = [];
  let users = [];
  let insideUsers = [];
  let monthlyStatsRows = [];
  let isAdvancedOpen = false;
  let refreshTimer = null;
  let appVersion = 'dev';

  // ユーザーモーダル制御
  let isUserModalOpen = false;
  let modalMode = 'create'; // 'create' | 'edit'
  let selectedUserForEdit = null;

  // 定員設定 (単一施設)
  const ROOM_CAPACITY = 10;

  // ----------------------------------------------------------------
  // データ読み込み (adminApi呼び出しに統一)
  // ----------------------------------------------------------------
  async function loadData() {
    try {
      const [s, l, u, i, m] = await Promise.all([
        GetDashboardStats(),
        GetRecentLogs(500),
        GetAllUsers(),
        GetCurrentInsideUsers(),
        GetMonthlyStats()
      ]);

      stats = s || { currentInsideCount: 0, totalUserCount: 0, todayLogCount: 0, weekLogCount: 0, monthLogCount: 0, yearLogCount: 0 };
      users = u || [];
      logs = l || [];
      insideUsers = i || [];
      monthlyStatsRows = m?.rows || [];
    } catch (e) {
      console.error('Failed to load DB data:', e);
    }
  }

  onMount(async () => {
    try {
      const v = await GetAppVersion();
      if (v) appVersion = v;
    } catch (e) {
      console.warn('Failed to get version:', e);
    }
    loadData();
    refreshTimer = setInterval(loadData, 5000);
  });

  onDestroy(() => {
    if (refreshTimer) clearInterval(refreshTimer);
  });

  // ダッシュボード用の成形済みログ
  $: processedLogs = (logs || []).map(l => {
    const formatted = formatLogTime(l.timestamp);
    const typeLabel = l.eventType === 'entry' ? '入室' : (l.eventType === 'force_exit' ? '強制退室' : '退室');
    const result = (l.userName && l.userName !== '未登録') ? '成功' : '失敗';
    const displayId = l.studentNo || l.cardId || '';
    const purpose = l.purpose || '';
    return { ...l, formatted, typeLabel, result, displayId, purpose };
  });

  // 在室人数カウント
  $: currentInsideNum = insideUsers ? insideUsers.length : (stats?.currentInsideCount ?? 0);

  // ユーザー作成・編集・削除ハンドラ
  function handleOpenCreateUser() {
    modalMode = 'create';
    selectedUserForEdit = null;
    isUserModalOpen = true;
  }

  function handleOpenEditUser(u) {
    modalMode = 'edit';
    selectedUserForEdit = u;
    isUserModalOpen = true;
  }

  async function handleSaveUser(userData) {
    await SaveUser(userData);
    await loadData();
  }

  async function handleDeleteUser(u) {
    const targetId = u.cardId || u.studentNo;
    if (confirm(`「${u.name}（${u.studentNo}）」を削除しますか？\n（関連する打刻ログも削除されます）`)) {
      try {
        await DeleteUser(targetId);
        await loadData();
      } catch (err) {
        alert("削除に失敗しました: " + (err?.message || err));
      }
    }
  }
</script>

<div class="w-full h-screen flex overflow-hidden bg-slate-50 text-slate-800 select-none font-sans">
  
  <!-- ==============================================================
       画面左側：固定サイドバー (濃紺背景)
       ============================================================== -->
  <aside class="w-64 bg-slate-900 text-slate-300 flex flex-col flex-shrink-0 z-20 border-r border-slate-800">
    <!-- システムヘッダー -->
    <div class="h-16 flex items-center px-6 border-b border-slate-800 gap-3">
      <div class="w-8 h-8 rounded-lg bg-blue-600 flex items-center justify-center text-white shadow-sm flex-shrink-0">
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"/>
        </svg>
      </div>
      <div>
        <h1 class="text-sm font-bold text-white tracking-wide">入退室管理システム</h1>
        <p class="text-[11px] text-blue-400">クリエイトセンター</p>
      </div>
    </div>

    <!-- サイドバーメニュー -->
    <nav class="flex-1 px-3 py-4 space-y-1 overflow-y-auto">
      <!-- 1. ダッシュボード -->
      <button 
        on:click={() => activeTab = 'dashboard'}
        class="w-full flex items-center gap-3 px-4 py-3 text-sm font-medium rounded-xl transition-all duration-150 {activeTab === 'dashboard' ? 'bg-blue-600 text-white shadow-sm' : 'text-slate-400 hover:text-white hover:bg-slate-800'}"
      >
        <svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z"/>
        </svg>
        <span>ダッシュボード</span>
      </button>

      <!-- 2. 入退室履歴 -->
      <button 
        on:click={() => activeTab = 'logs'}
        class="w-full flex items-center gap-3 px-4 py-3 text-sm font-medium rounded-xl transition-all duration-150 {activeTab === 'logs' ? 'bg-blue-600 text-white shadow-sm' : 'text-slate-400 hover:text-white hover:bg-slate-800'}"
      >
        <svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
        </svg>
        <span>入退室履歴</span>
      </button>

      <!-- 3. 在室中メンバー -->
      <button 
        on:click={() => activeTab = 'inside'}
        class="w-full flex items-center justify-between px-4 py-3 text-sm font-medium rounded-xl transition-all duration-150 {activeTab === 'inside' ? 'bg-blue-600 text-white shadow-sm' : 'text-slate-400 hover:text-white hover:bg-slate-800'}"
      >
        <div class="flex items-center gap-3">
          <svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"/>
          </svg>
          <span>在室中メンバー</span>
        </div>
        {#if currentInsideNum > 0}
          <span class="px-2 py-0.5 text-xs font-bold rounded-full {activeTab === 'inside' ? 'bg-white text-blue-600' : 'bg-emerald-500 text-white'}">
            {currentInsideNum}
          </span>
        {/if}
      </button>

      <!-- 4. ユーザー管理 (利用者マスター) -->
      <button 
        on:click={() => activeTab = 'users'}
        class="w-full flex items-center gap-3 px-4 py-3 text-sm font-medium rounded-xl transition-all duration-150 {activeTab === 'users' ? 'bg-blue-600 text-white shadow-sm' : 'text-slate-400 hover:text-white hover:bg-slate-800'}"
      >
        <svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z"/>
        </svg>
        <span>ユーザー管理</span>
      </button>

      <!-- 高度な操作 / キオスク復帰 -->
      <div class="pt-4 mt-4 border-t border-slate-800/80 space-y-1">
        <button 
          on:click={() => isAdvancedOpen = true}
          class="w-full flex items-center gap-3 px-4 py-2.5 text-xs font-semibold rounded-xl text-slate-400 hover:text-white hover:bg-slate-800 transition"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
          </svg>
          <span>高度な操作 / CSV</span>
        </button>

        <button 
          on:click={onBackToKiosk}
          class="w-full flex items-center gap-3 px-4 py-2.5 text-xs font-semibold rounded-xl text-slate-400 hover:text-white hover:bg-slate-800 transition"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 17l-5-5m0 0l5-5m-5 5h12"/>
          </svg>
          <span>キオスク画面に戻る</span>
        </button>
      </div>
    </nav>

    <!-- フッター情報 -->
    <div class="p-4 border-t border-slate-800 text-xs text-slate-500 flex items-center justify-between">
      <div class="text-[11px] text-slate-500 font-mono">SQLite DB / Wails</div>
      <div class="text-[11px] text-blue-400 font-mono font-semibold" title="システムバージョン">{appVersion}</div>
    </div>
  </aside>

  <!-- ==============================================================
       メインコンテンツエリア
       ============================================================== -->
  <div class="flex-1 flex flex-col min-w-0 overflow-hidden bg-slate-50">

    <!-- 上部ヘッダー -->
    <header class="h-16 bg-white border-b border-slate-200 flex items-center justify-between px-8 flex-shrink-0 z-10 shadow-xs">
      <div class="flex items-center gap-3">
        <h2 class="text-lg font-bold text-slate-800">
          {activeTab === 'dashboard' ? 'ダッシュボード' : activeTab === 'logs' ? '入退室履歴' : activeTab === 'inside' ? '在室中メンバー' : 'ユーザー管理'}
        </h2>
        <span class="text-xs bg-blue-50 text-blue-700 px-3 py-1 rounded-full font-semibold border border-blue-100">
          対象施設：クリエイトセンター
        </span>
      </div>

      <!-- 右側：全画面切替 & 管理者表示 -->
      <div class="flex items-center gap-4">
        <button 
          on:click={async () => { await ToggleFullscreen(); }}
          class="text-xs text-slate-600 hover:text-slate-900 px-3 py-1.5 rounded-lg border border-slate-200 hover:bg-slate-50 transition font-medium"
          title="フルスクリーン切り替え (05prompt)"
        >
          全画面切替
        </button>

        <div class="flex items-center gap-3 pl-2 border-l border-slate-200">
          <div class="w-9 h-9 rounded-full bg-blue-600 text-white flex items-center justify-center font-bold text-sm shadow-xs">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"/>
            </svg>
          </div>
          <div class="text-sm font-bold text-slate-800">管理者</div>
        </div>
      </div>
    </header>

    <!-- メインスクロール領域 -->
    <main class="flex-1 overflow-y-auto p-8 space-y-6">
      {#if activeTab === 'dashboard'}
        <DashboardTab 
          {stats}
          {insideUsers}
          {processedLogs}
          {monthlyStatsRows}
          onNavigateToLogs={() => activeTab = 'logs'}
        />
      {:else if activeTab === 'logs'}
        <LogsTab 
          {logs}
        />
      {:else if activeTab === 'inside'}
        <InsideTab 
          {insideUsers}
          roomCapacity={ROOM_CAPACITY}
          onOpenAdvanced={() => isAdvancedOpen = true}
        />
      {:else if activeTab === 'users'}
        <UsersTab 
          {users}
          onCreateUser={handleOpenCreateUser}
          onEditUser={handleOpenEditUser}
          onDeleteUser={handleDeleteUser}
        />
      {/if}
    </main>
  </div>

  <!-- ユーザー登録・編集 モーダルウィンドウ -->
  <UserModal 
    isOpen={isUserModalOpen}
    mode={modalMode}
    initialUser={selectedUserForEdit}
    onClose={() => isUserModalOpen = false}
    onSave={handleSaveUser}
  />

  <!-- 高度な操作モーダル -->
  {#if isAdvancedOpen}
    <AdvancedOpsModal 
      onClose={() => { isAdvancedOpen = false; }}
      onRefresh={loadData}
    />
  {/if}

</div>
