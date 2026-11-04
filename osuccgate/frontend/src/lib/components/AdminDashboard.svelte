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
    GetAppVersion,
    GetAvailableFiscalYears,
    GetCurrentFiscalYear,
    FindUserInAnyFiscalYear
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

  // 年度管理状態
  let currentFiscalYear = 2026;
  let selectedFiscalYear = 2026;
  let availableFiscalYears = [2026];

  // バックエンドデータ状態
  let stats = {
    fiscalYear: 2026,
    currentInsideCount: 0,
    totalUserCount: 0,
    todayLogCount: 0,
    todayEntryCount: 0,
    todayExitCount: 0,
    weekLogCount: 0,
    weekEntryCount: 0,
    weekExitCount: 0,
    monthLogCount: 0,
    monthEntryCount: 0,
    monthExitCount: 0,
    yearLogCount: 0,
    yearEntryCount: 0,
    yearExitCount: 0,
    weekChart: [],
    monthChart: [],
    yearChart: []
  };
  let logs = [];
  let users = [];
  let insideUsers = [];
  let monthlyStatsRows = [];
  let isAdvancedOpen = false;
  let refreshTimer = null;
  let appVersion = 'dev';
  let isLoading = false;
  let pendingForceReload = false;
  let lastLoadedTab = null;

  // ユーザーモーダル制御
  let isUserModalOpen = false;
  let modalMode = 'create'; // 'create' | 'edit' | 'renewal'
  let selectedUserForEdit = null;

  let mounted = false;

  // ----------------------------------------------------------------
  // データ読み込み (多重実行防止ガード & タブ別オンデマンド取得 & 年度対応)
  // ----------------------------------------------------------------
  async function loadData(forceAll = false) {
    if (isLoading) {
      if (forceAll) {
        pendingForceReload = true;
      }
      return;
    }
    isLoading = true;
    const targetFY = selectedFiscalYear;

    try {
      // 初回またはタブ切り替え、明示的更新時は全データ取得
      if (forceAll || lastLoadedTab !== activeTab) {
        const [s, l, u, i, m, years] = await Promise.all([
          GetDashboardStats(targetFY).catch(e => { console.warn('GetDashboardStats error:', e); return null; }),
          GetRecentLogs(500, targetFY).catch(e => { console.warn('GetRecentLogs error:', e); return null; }),
          GetAllUsers(targetFY).catch(e => { console.warn('GetAllUsers error:', e); return null; }),
          GetCurrentInsideUsers().catch(e => { console.warn('GetCurrentInsideUsers error:', e); return null; }),
          GetMonthlyStats(targetFY).catch(e => { console.warn('GetMonthlyStats error:', e); return null; }),
          GetAvailableFiscalYears().catch(e => { console.warn('GetAvailableFiscalYears error:', e); return [targetFY]; })
        ]);

        if (s) stats = s;
        if (u) users = u;
        if (l) logs = l;
        if (i) insideUsers = i;
        if (m?.rows) monthlyStatsRows = m.rows;
        if (years && years.length > 0) {
          availableFiscalYears = years;
        }
        lastLoadedTab = activeTab;
        return;
      }

      // 5秒ごとの定期ポーリングではアクティブタブに必要なデータのみ最小限に取得
      if (activeTab === 'dashboard') {
        const [s, l, i, m] = await Promise.all([
          GetDashboardStats(targetFY).catch(e => { console.warn(e); return null; }),
          GetRecentLogs(500, targetFY).catch(e => { console.warn(e); return null; }),
          GetCurrentInsideUsers().catch(e => { console.warn(e); return null; }),
          GetMonthlyStats(targetFY).catch(e => { console.warn(e); return null; })
        ]);
        if (s) stats = s;
        if (l) logs = l;
        if (i) insideUsers = i;
        if (m?.rows) monthlyStatsRows = m.rows;
      } else if (activeTab === 'logs') {
        const [l, i] = await Promise.all([
          GetRecentLogs(500, targetFY).catch(e => { console.warn(e); return null; }),
          GetCurrentInsideUsers().catch(e => { console.warn(e); return null; })
        ]);
        if (l) logs = l;
        if (i) insideUsers = i;
      } else if (activeTab === 'inside') {
        const i = await GetCurrentInsideUsers().catch(e => { console.warn(e); return null; });
        if (i) insideUsers = i;
      } else if (activeTab === 'users') {
        const [u, i] = await Promise.all([
          GetAllUsers(targetFY).catch(e => { console.warn(e); return null; }),
          GetCurrentInsideUsers().catch(e => { console.warn(e); return null; })
        ]);
        if (u) users = u;
        if (i) insideUsers = i;
      }
    } catch (e) {
      console.error('Failed to load DB data:', e);
    } finally {
      isLoading = false;
      if (pendingForceReload) {
        pendingForceReload = false;
        loadData(true);
      }
    }
  }

  // タブ切り替え時にオンデマンドでデータ取得
  $: if (activeTab && mounted) {
    loadData();
  }

  onMount(async () => {
    try {
      const v = await GetAppVersion();
      if (v) appVersion = v;
      const cy = await GetCurrentFiscalYear();
      if (cy) {
        currentFiscalYear = cy;
        selectedFiscalYear = cy;
      }
    } catch (e) {
      console.warn('Failed to get initial metadata:', e);
    }
    mounted = true;
    await loadData(true);
    refreshTimer = setInterval(() => loadData(false), 5000);
  });

  onDestroy(() => {
    if (refreshTimer) clearInterval(refreshTimer);
  });

  // ダッシュボード・履歴タブ用の成形済みログ
  $: processedLogs = (logs || []).map(l => {
    const formatted = formatLogTime(l.timestamp);
    const hasRegisteredUser = l.userName && l.userName !== '未登録';

    let typeLabel = '入室';
    let result = hasRegisteredUser ? '成功' : '失敗';

    if (l.eventType === 'exit') {
      typeLabel = '退室';
    } else if (l.eventType === 'force_exit') {
      typeLabel = '強制退室';
    } else if (l.eventType === 'unregistered_renewal') {
      typeLabel = hasRegisteredUser ? '今年度未登録(登録済)' : '今年度未登録';
      result = hasRegisteredUser ? '登録済' : '失敗';
    } else if (l.eventType === 'unregistered') {
      typeLabel = hasRegisteredUser ? '未登録(登録済)' : '未登録';
      result = hasRegisteredUser ? '登録済' : '失敗';
    }
    const displayId = l.studentNo || l.cardId || '';
    const purpose = l.purpose || '';
    return { ...l, formatted, typeLabel, result, displayId, purpose, isRegistered: Boolean(hasRegisteredUser) };
  });

  // 在室人数カウント
  $: currentInsideNum = insideUsers ? insideUsers.length : (stats?.currentInsideCount ?? 0);

  // 表示年度切り替え
  function handleSelectFiscalYear(fy) {
    selectedFiscalYear = fy;
    lastLoadedTab = null;
    loadData(true);
  }

  // ワンクリック新規登録 (未登録カード)
  function handleOpenRegisterUnregistered(log) {
    modalMode = 'create';
    selectedUserForEdit = {
      cardId: log.cardId,
      studentNo: log.studentNo || log.cardId,
      name: '',
      roleName: '学生',
      roleCode: 1,
      adminNo: '',
      contact: '',
      purpose: ''
    };
    isUserModalOpen = true;
  }

  // ワンクリック更新 (今年度未登録カード)
  async function handleOpenRenewalUser(log) {
    const identifier = log.studentNo || log.cardId;
    let pastUser = null;
    try {
      pastUser = await FindUserInAnyFiscalYear(identifier);
    } catch (e) {
      console.warn('Failed to find past user:', e);
    }

    modalMode = 'renewal';
    if (pastUser) {
      selectedUserForEdit = {
        ...pastUser,
        adminNo: '' // 今年度自動採番のためクリア
      };
    } else {
      selectedUserForEdit = {
        cardId: log.cardId,
        studentNo: log.studentNo || log.cardId,
        name: log.userName !== '未登録' ? log.userName : '',
        roleName: log.roleName !== '未登録' ? log.roleName : '学生',
        roleCode: log.roleCode ?? 1,
        purpose: log.purpose || '',
        adminNo: '',
        contact: ''
      };
    }
    isUserModalOpen = true;
  }

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
    // 常に今年度のデータとして登録/更新
    userData.fiscalYear = currentFiscalYear;
    await SaveUser(userData);
    await loadData(true);
  }

  async function handleDeleteUser(u) {
    const targetId = u.cardId || u.studentNo;
    if (confirm(`「${u.name}（${u.studentNo}）」を削除しますか？\n（${selectedFiscalYear}年度の登録データが削除されます）`)) {
      try {
        await DeleteUser(targetId, selectedFiscalYear);
        await loadData(true);
      } catch (err) {
        alert("削除に失敗しました: " + (err?.message || err));
      }
    }
  }
</script>

<div class="w-full h-screen flex overflow-hidden bg-m3-surface text-m3-on-surface select-none font-sans">
  
  <!-- ==============================================================
       画面左側：M3 Navigation Drawer (Surface Container Low + Pill Indicator)
       ============================================================== -->
  <aside class="w-64 bg-m3-surface-container-low text-m3-on-surface flex flex-col flex-shrink-0 z-20 border-r border-m3-outline-variant/50">
    <!-- システムヘッダー -->
    <div class="h-18 py-4 flex items-center px-5 gap-3.5">
      <div class="w-10 h-10 rounded-2xl bg-m3-primary-container text-m3-on-primary-container flex items-center justify-center flex-shrink-0 shadow-m3-1">
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"/>
        </svg>
      </div>
      <div>
        <h1 class="text-sm font-bold text-m3-on-surface tracking-tight">入退室管理システム</h1>
        <p class="text-[11px] font-semibold text-m3-primary">クリエイトセンター</p>
      </div>
    </div>

    <!-- M3 Navigation Drawer Items (Active Pill Indicator) -->
    <nav class="flex-1 px-3 py-2 space-y-1 overflow-y-auto">
      <div class="px-4 py-1.5 text-[11px] font-bold text-m3-on-surface-variant uppercase tracking-wider">
        メインメニュー
      </div>

      <!-- 1. ダッシュボード -->
      <button 
        on:click={() => activeTab = 'dashboard'}
        class="w-full flex items-center gap-3.5 px-4 py-3 text-sm rounded-full transition-colors duration-150 {activeTab === 'dashboard' ? 'bg-m3-secondary-container text-m3-on-secondary-container font-bold' : 'text-m3-on-surface-variant font-medium hover:bg-m3-surface-container-high hover:text-m3-on-surface'}"
      >
        <svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z"/>
        </svg>
        <span>ダッシュボード</span>
      </button>

      <!-- 2. 入退室履歴 -->
      <button 
        on:click={() => activeTab = 'logs'}
        class="w-full flex items-center gap-3.5 px-4 py-3 text-sm rounded-full transition-colors duration-150 {activeTab === 'logs' ? 'bg-m3-secondary-container text-m3-on-secondary-container font-bold' : 'text-m3-on-surface-variant font-medium hover:bg-m3-surface-container-high hover:text-m3-on-surface'}"
      >
        <svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
        </svg>
        <span>入退室履歴</span>
      </button>

      <!-- 3. 在室中メンバー -->
      <button 
        on:click={() => activeTab = 'inside'}
        class="w-full flex items-center justify-between px-4 py-3 text-sm rounded-full transition-colors duration-150 {activeTab === 'inside' ? 'bg-m3-secondary-container text-m3-on-secondary-container font-bold' : 'text-m3-on-surface-variant font-medium hover:bg-m3-surface-container-high hover:text-m3-on-surface'}"
      >
        <div class="flex items-center gap-3.5">
          <svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"/>
          </svg>
          <span>在室中メンバー</span>
        </div>
        {#if currentInsideNum > 0}
          <span class="px-2.5 py-0.5 text-xs font-bold rounded-full {activeTab === 'inside' ? 'bg-m3-primary text-m3-on-primary' : 'bg-m3-tertiary-container text-m3-on-tertiary-container'}">
            {currentInsideNum}
          </span>
        {/if}
      </button>

      <!-- 4. ユーザー管理 (利用者マスター) -->
      <button 
        on:click={() => activeTab = 'users'}
        class="w-full flex items-center gap-3.5 px-4 py-3 text-sm rounded-full transition-colors duration-150 {activeTab === 'users' ? 'bg-m3-secondary-container text-m3-on-secondary-container font-bold' : 'text-m3-on-surface-variant font-medium hover:bg-m3-surface-container-high hover:text-m3-on-surface'}"
      >
        <svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z"/>
        </svg>
        <span>ユーザー管理</span>
      </button>

      <!-- システム・キオスク操作セクション -->
      <div class="pt-4 mt-4 border-t border-m3-outline-variant space-y-1">
        <div class="px-4 py-1 text-[11px] font-bold text-m3-on-surface-variant uppercase tracking-wider">
          システム操作
        </div>

        <button 
          on:click={() => isAdvancedOpen = true}
          class="w-full flex items-center gap-3.5 px-4 py-2.5 text-xs font-semibold rounded-full text-m3-on-surface-variant hover:text-m3-on-surface hover:bg-m3-surface-container-high transition-colors duration-150"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
          </svg>
          <span>高度な操作 / CSV</span>
        </button>

        <button 
          on:click={onBackToKiosk}
          class="w-full flex items-center gap-3.5 px-4 py-2.5 text-xs font-semibold rounded-full text-m3-primary hover:bg-m3-primary-container transition-colors duration-150"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 17l-5-5m0 0l5-5m-5 5h12"/>
          </svg>
          <span>キオスク画面に戻る</span>
        </button>
      </div>
    </nav>

    <!-- フッター情報 -->
    <div class="p-4 mx-3 mb-3 rounded-2xl bg-m3-surface-container flex items-center justify-between text-xs">
      <div class="text-[11px] text-m3-on-surface-variant font-mono">SQLite / Wails</div>
      <div class="text-[11px] px-2 py-0.5 rounded-full bg-m3-primary-container text-m3-on-primary-container font-mono font-bold" title="システムバージョン">
        {appVersion}
      </div>
    </div>
  </aside>

  <!-- ==============================================================
       メインコンテンツエリア (M3 Surface)
       ============================================================== -->
  <div class="flex-1 flex flex-col min-w-0 overflow-hidden bg-m3-surface">

    <!-- 上部 M3 Top App Bar (スクロール負荷軽減のためソリッド背景を適用) -->
    <header class="h-16 bg-m3-surface-container-lowest border-b border-m3-outline-variant/50 flex items-center justify-between px-8 flex-shrink-0 z-10">
      <div class="flex items-center gap-3">
        <h2 class="text-lg font-bold text-m3-on-surface tracking-tight">
          {activeTab === 'dashboard' ? 'ダッシュボード' : activeTab === 'logs' ? '入退室履歴' : activeTab === 'inside' ? '在室中メンバー' : 'ユーザー管理'}
        </h2>
        <!-- M3 Assist Chip: 対象施設 -->
        <span class="inline-flex items-center gap-1.5 text-xs bg-m3-secondary-container/70 text-m3-on-secondary-container px-3.5 py-1 rounded-full font-semibold">
          <span class="w-1.5 h-1.5 rounded-full bg-m3-secondary"></span>
          対象施設：クリエイトセンター
        </span>
        <!-- M3 Tonal Chip: 選択年度 -->
        <span class="hidden sm:inline-flex items-center text-xs bg-m3-primary-container/60 text-m3-on-primary-container px-3 py-1 rounded-full font-bold">
          {selectedFiscalYear}年度
        </span>
      </div>

      <!-- 右側：全画面切替 (M3 Outlined Pill Button) & 管理者チップ -->
      <div class="flex items-center gap-3">
        <button 
          on:click={async () => { await ToggleFullscreen(); }}
          class="m3-btn-outlined py-1.5 px-4 text-xs"
          title="フルスクリーン切り替え (05prompt)"
        >
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4"/>
          </svg>
          <span>全画面切替</span>
        </button>

        <div class="flex items-center gap-2.5 pl-3 border-l border-m3-outline-variant/60">
          <div class="w-9 h-9 rounded-full bg-m3-primary-container text-m3-on-primary-container flex items-center justify-center font-bold text-sm">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"/>
            </svg>
          </div>
          <div class="text-sm font-bold text-m3-on-surface">管理者</div>
        </div>
      </div>
    </header>

    <!-- メインスクロール領域 (GPU合成とスクロール最適化) -->
    <main class="flex-1 overflow-y-auto p-8 space-y-6 [will-change:scroll-position]">
      {#if activeTab === 'dashboard'}
        <DashboardTab 
          {stats}
          {insideUsers}
          {processedLogs}
          {monthlyStatsRows}
          {availableFiscalYears}
          {selectedFiscalYear}
          {currentFiscalYear}
          onSelectFiscalYear={handleSelectFiscalYear}
          onNavigateToLogs={() => activeTab = 'logs'}
          onRenewalUser={handleOpenRenewalUser}
          onRegisterUnregistered={handleOpenRegisterUnregistered}
        />
      {:else if activeTab === 'logs'}
        <LogsTab 
          {logs}
          {processedLogs}
          {availableFiscalYears}
          {selectedFiscalYear}
          onSelectFiscalYear={handleSelectFiscalYear}
          onRenewalUser={handleOpenRenewalUser}
          onRegisterUnregistered={handleOpenRegisterUnregistered}
        />
      {:else if activeTab === 'inside'}
        <InsideTab 
          {insideUsers}
          onOpenAdvanced={() => isAdvancedOpen = true}
        />
      {:else if activeTab === 'users'}
        <UsersTab 
          {users}
          {availableFiscalYears}
          {selectedFiscalYear}
          {currentFiscalYear}
          onSelectFiscalYear={handleSelectFiscalYear}
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
    {currentFiscalYear}
    onClose={() => isUserModalOpen = false}
    onSave={handleSaveUser}
  />

  <!-- 高度な操作モーダル -->
  {#if isAdvancedOpen}
    <AdvancedOpsModal 
      onClose={() => { isAdvancedOpen = false; }}
      onRefresh={() => loadData(true)}
    />
  {/if}

</div>
