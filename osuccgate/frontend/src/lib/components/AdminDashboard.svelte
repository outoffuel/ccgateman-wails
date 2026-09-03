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
    ToggleFullscreen
  } from '../../../wailsjs/go/main/App.js';
  import AdvancedOpsModal from './AdvancedOpsModal.svelte';

  export let onBackToKiosk = () => {};

  let activeTab = 'logs'; // 'logs' | 'inside' | 'users' | 'monthly'
  let stats = { currentInsideCount: 0, totalUserCount: 0, todayLogCount: 0 };
  let logs = [];
  let users = [];
  let insideUsers = [];
  let monthlyStats = { rows: [] };
  let isAdvancedOpen = false;

  // 1. ソートステート
  let logsSortKey = 'timestamp';
  let logsSortDir = 'desc'; // デフォルト: 打刻日時の新しい順

  let insideSortKey = 'lastEventTime';
  let insideSortDir = 'desc'; // デフォルト: 入室日時の新しい順

  let usersSortKey = 'roleCode';
  let usersSortDir = 'asc'; // デフォルト: 区分順

  let monthlySortKey = 'yearMonth';
  let monthlySortDir = 'desc'; // デフォルト: 年月の新しい順

  // 2. 列幅ステート (px)
  let logsWidths = {
    timestamp: 180,
    userName: 140,
    roleName: 110,
    studentNo: 140,
    eventType: 130,
    method: 140,
    duration: 120
  };

  let insideWidths = {
    userName: 200,
    roleName: 140,
    studentNo: 160,
    lastEventTime: 160,
    stayDuration: 180
  };

  let usersWidths = {
    adminNo: 80,
    studentNo: 140,
    name: 150,
    furigana: 150,
    gender: 90,
    roleName: 130,
    contact: 160,
    purpose: 180,
    cardId: 150,
    actions: 90
  };

  let monthlyWidths = {
    yearMonth: 130,
    roleOther: 80,
    role0: 130,
    role1: 130,
    role9: 130,
    monthlyTotal: 100,
    quarterTotal: 160,
    fiscalYearCumulativeTotal: 160
  };

  // リサイズドラッグハンドラー
  let isResizing = false;
  let activeResizeTable = null;
  let activeResizeCol = null;
  let startX = 0;
  let startWidth = 0;

  function startResize(e, table, colKey) {
    e.stopPropagation();
    e.preventDefault();
    isResizing = true;
    activeResizeTable = table;
    activeResizeCol = colKey;
    startX = e.clientX;

    if (table === 'logs') startWidth = logsWidths[colKey];
    else if (table === 'inside') startWidth = insideWidths[colKey];
    else if (table === 'users') startWidth = usersWidths[colKey];
    else if (table === 'monthly') startWidth = monthlyWidths[colKey];

    window.addEventListener('mousemove', handleResizeMove);
    window.addEventListener('mouseup', handleResizeEnd);
  }

  function handleResizeMove(e) {
    if (!isResizing) return;
    const delta = e.clientX - startX;
    const newWidth = Math.max(50, startWidth + delta);

    if (activeResizeTable === 'logs') {
      logsWidths = { ...logsWidths, [activeResizeCol]: newWidth };
    } else if (activeResizeTable === 'inside') {
      insideWidths = { ...insideWidths, [activeResizeCol]: newWidth };
    } else if (activeResizeTable === 'users') {
      usersWidths = { ...usersWidths, [activeResizeCol]: newWidth };
    } else if (activeResizeTable === 'monthly') {
      monthlyWidths = { ...monthlyWidths, [activeResizeCol]: newWidth };
    }
  }

  function handleResizeEnd() {
    isResizing = false;
    activeResizeTable = null;
    activeResizeCol = null;
    window.removeEventListener('mousemove', handleResizeMove);
    window.removeEventListener('mouseup', handleResizeEnd);
  }

  // 汎用ソート処理関数
  function sortData(list, key, dir, customExtractors = {}) {
    if (!list || list.length === 0) return [];
    const sorted = [...list];
    sorted.sort((a, b) => {
      let valA = customExtractors[key] ? customExtractors[key](a) : a[key];
      let valB = customExtractors[key] ? customExtractors[key](b) : b[key];

      if (valA === undefined || valA === null) valA = '';
      if (valB === undefined || valB === null) valB = '';

      let comparison = 0;
      if (typeof valA === 'number' && typeof valB === 'number') {
        comparison = valA - valB;
      } else {
        const numA = Number(valA);
        const numB = Number(valB);
        if (!isNaN(numA) && !isNaN(numB) && String(valA).trim() !== '' && String(valB).trim() !== '') {
          comparison = numA - numB;
        } else {
          comparison = String(valA).localeCompare(String(valB), 'ja', { numeric: true, sensitivity: 'base' });
        }
      }

      return dir === 'asc' ? comparison : -comparison;
    });
    return sorted;
  }

  function handleSort(tab, key) {
    if (tab === 'logs') {
      if (logsSortKey === key) {
        logsSortDir = logsSortDir === 'asc' ? 'desc' : 'asc';
      } else {
        logsSortKey = key;
        logsSortDir = 'asc';
      }
    } else if (tab === 'inside') {
      if (insideSortKey === key) {
        insideSortDir = insideSortDir === 'asc' ? 'desc' : 'asc';
      } else {
        insideSortKey = key;
        insideSortDir = 'asc';
      }
    } else if (tab === 'users') {
      if (usersSortKey === key) {
        usersSortDir = usersSortDir === 'asc' ? 'desc' : 'asc';
      } else {
        usersSortKey = key;
        usersSortDir = 'asc';
      }
    } else if (tab === 'monthly') {
      if (monthlySortKey === key) {
        monthlySortDir = monthlySortDir === 'asc' ? 'desc' : 'asc';
      } else {
        monthlySortKey = key;
        monthlySortDir = 'asc';
      }
    }
  }

  // ソート済みデータ
  $: sortedLogs = sortData(logs, logsSortKey, logsSortDir, {
    timestamp: item => new Date(item.timestamp).getTime(),
    duration: item => item.durationSecond || 0
  });

  $: sortedInsideUsers = sortData(insideUsers, insideSortKey, insideSortDir, {
    lastEventTime: item => new Date(item.lastEventTime).getTime(),
    stayDuration: item => {
      const t = new Date(item.lastEventTime).getTime();
      return isNaN(t) ? 0 : Date.now() - t;
    }
  });

  $: sortedUsers = sortData(users, usersSortKey, usersSortDir, {
    adminNo: item => {
      const n = parseInt(item.adminNo, 10);
      return isNaN(n) ? item.adminNo || '' : n;
    }
  });

  $: sortedMonthlyRows = sortData(monthlyStats?.rows || [], monthlySortKey, monthlySortDir);

  // ユーザー編集/登録用モーダルステート
  let isUserModalOpen = false;
  let editingUser = null;
  let formStudentNo = '';
  let formName = '';
  let formFurigana = '';
  let formGender = '男';
  let formRoleName = '学生';
  let formRoleCode = 1;
  let formCardId = '';
  let formAdminNo = '';
  let formContact = '';
  let formPurpose = '';
  let formError = '';

  let refreshTimer = null;

  async function loadData() {
    try {
      const [s, l, u, i, m] = await Promise.all([
        GetDashboardStats(),
        GetRecentLogs(50),
        GetAllUsers(),
        GetCurrentInsideUsers(),
        GetMonthlyStats()
      ]);
      stats = s || { currentInsideCount: 0, totalUserCount: 0, todayLogCount: 0 };
      logs = l || [];
      users = u || [];
      insideUsers = i || [];
      monthlyStats = m || { rows: [] };
    } catch (e) {
      console.error('Failed to load dashboard data:', e);
    }
  }

  onMount(() => {
    loadData();
    refreshTimer = setInterval(loadData, 10000);
    return () => {
      if (refreshTimer) clearInterval(refreshTimer);
      if (typeof window !== 'undefined') {
        window.removeEventListener('mousemove', handleResizeMove);
        window.removeEventListener('mouseup', handleResizeEnd);
      }
    };
  });

  onDestroy(() => {
    if (typeof window !== 'undefined') {
      window.removeEventListener('mousemove', handleResizeMove);
      window.removeEventListener('mouseup', handleResizeEnd);
    }
  });

  function openNewUserModal() {
    editingUser = null;
    formStudentNo = '';
    formName = '';
    formFurigana = '';
    formGender = '男';
    formRoleName = '学生';
    formRoleCode = 1;
    formCardId = '';
    formAdminNo = '';
    formContact = '';
    formPurpose = '';
    formError = '';
    isUserModalOpen = true;
  }

  function openEditUserModal(u) {
    editingUser = u;
    formStudentNo = u.studentNo || '';
    formName = u.name || '';
    formFurigana = u.furigana || '';
    formGender = u.gender || '男';
    formRoleName = u.roleName || '学生';
    formRoleCode = u.roleCode ?? 1;
    formCardId = u.cardId || '';
    formAdminNo = u.adminNo || '';
    formContact = u.contact || '';
    formPurpose = u.purpose || '';
    formError = '';
    isUserModalOpen = true;
  }

  function handleRoleChange() {
    if (formRoleName === '教職員') formRoleCode = 0;
    else if (formRoleName === '学生') formRoleCode = 1;
    else if (formRoleName === '学生スタッフ') formRoleCode = 9;
    else formRoleCode = 1;
  }

  async function handleSaveUser() {
    if (!formStudentNo.trim()) {
      formError = '学籍番号 / 職員番号は必須です';
      return;
    }
    if (!formName.trim()) {
      formError = '氏名は必須です';
      return;
    }

    // カードIDが空の場合は学籍番号と同一の値を自動適用
    const finalCardId = formCardId.trim() || formStudentNo.trim();

    try {
      await SaveUser({
        cardId: finalCardId,
        studentNo: formStudentNo.trim(),
        name: formName.trim(),
        furigana: formFurigana.trim(),
        gender: formGender,
        roleName: formRoleName,
        roleCode: Number(formRoleCode),
        adminNo: formAdminNo.trim(),
        contact: formContact.trim(),
        purpose: formPurpose.trim()
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
        on:click={async () => { await ToggleFullscreen(); }}
        class="bg-slate-800 hover:bg-slate-700 text-slate-300 px-3.5 py-2 rounded-xl text-xs font-semibold transition flex items-center gap-1"
        title="フルスクリーン表示のON/OFF"
      >
        🖥️ フルスクリーン切替
      </button>

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
        <button 
          on:click={() => activeTab = 'monthly'}
          class={`px-5 py-2.5 rounded-2xl font-bold text-xs transition ${activeTab === 'monthly' ? 'bg-blue-600 text-white shadow-lg shadow-blue-600/30' : 'bg-slate-900 text-slate-400 hover:text-white border border-slate-800'}`}
        >
          📊 月別統計
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
            <table class="text-left text-sm text-slate-300 border-collapse" style="table-layout: fixed; width: max-content; min-width: 100%;">
              <colgroup>
                <col style="width: {logsWidths.timestamp}px;">
                <col style="width: {logsWidths.userName}px;">
                <col style="width: {logsWidths.roleName}px;">
                <col style="width: {logsWidths.studentNo}px;">
                <col style="width: {logsWidths.eventType}px;">
                <col style="width: {logsWidths.method}px;">
                <col style="width: {logsWidths.duration}px;">
              </colgroup>
              <thead class="text-xs text-slate-400 uppercase bg-slate-800/60 border-b border-slate-800 select-none">
                <tr>
                  <th 
                    class="p-3.5 rounded-l-xl cursor-pointer hover:bg-slate-800/90 transition relative group"
                    on:click={() => handleSort('logs', 'timestamp')}
                  >
                    <div class="flex items-center justify-between pr-2">
                      <span>打刻日時</span>
                      <span class="text-xs font-mono {logsSortKey === 'timestamp' ? 'text-blue-400 font-bold' : 'text-slate-600 group-hover:text-slate-400'}">
                        {logsSortKey === 'timestamp' ? (logsSortDir === 'asc' ? '▲' : '▼') : '↕'}
                      </span>
                    </div>
                    <div class="resizer" on:mousedown={(e) => startResize(e, 'logs', 'timestamp')} />
                  </th>

                  <th 
                    class="p-3.5 cursor-pointer hover:bg-slate-800/90 transition relative group"
                    on:click={() => handleSort('logs', 'userName')}
                  >
                    <div class="flex items-center justify-between pr-2">
                      <span>氏名</span>
                      <span class="text-xs font-mono {logsSortKey === 'userName' ? 'text-blue-400 font-bold' : 'text-slate-600 group-hover:text-slate-400'}">
                        {logsSortKey === 'userName' ? (logsSortDir === 'asc' ? '▲' : '▼') : '↕'}
                      </span>
                    </div>
                    <div class="resizer" on:mousedown={(e) => startResize(e, 'logs', 'userName')} />
                  </th>

                  <th 
                    class="p-3.5 cursor-pointer hover:bg-slate-800/90 transition relative group"
                    on:click={() => handleSort('logs', 'roleName')}
                  >
                    <div class="flex items-center justify-between pr-2">
                      <span>区分</span>
                      <span class="text-xs font-mono {logsSortKey === 'roleName' ? 'text-blue-400 font-bold' : 'text-slate-600 group-hover:text-slate-400'}">
                        {logsSortKey === 'roleName' ? (logsSortDir === 'asc' ? '▲' : '▼') : '↕'}
                      </span>
                    </div>
                    <div class="resizer" on:mousedown={(e) => startResize(e, 'logs', 'roleName')} />
                  </th>

                  <th 
                    class="p-3.5 cursor-pointer hover:bg-slate-800/90 transition relative group"
                    on:click={() => handleSort('logs', 'studentNo')}
                  >
                    <div class="flex items-center justify-between pr-2">
                      <span>学籍/職員番号</span>
                      <span class="text-xs font-mono {logsSortKey === 'studentNo' ? 'text-blue-400 font-bold' : 'text-slate-600 group-hover:text-slate-400'}">
                        {logsSortKey === 'studentNo' ? (logsSortDir === 'asc' ? '▲' : '▼') : '↕'}
                      </span>
                    </div>
                    <div class="resizer" on:mousedown={(e) => startResize(e, 'logs', 'studentNo')} />
                  </th>

                  <th 
                    class="p-3.5 cursor-pointer hover:bg-slate-800/90 transition relative group"
                    on:click={() => handleSort('logs', 'eventType')}
                  >
                    <div class="flex items-center justify-between pr-2">
                      <span>ステータス</span>
                      <span class="text-xs font-mono {logsSortKey === 'eventType' ? 'text-blue-400 font-bold' : 'text-slate-600 group-hover:text-slate-400'}">
                        {logsSortKey === 'eventType' ? (logsSortDir === 'asc' ? '▲' : '▼') : '↕'}
                      </span>
                    </div>
                    <div class="resizer" on:mousedown={(e) => startResize(e, 'logs', 'eventType')} />
                  </th>

                  <th 
                    class="p-3.5 cursor-pointer hover:bg-slate-800/90 transition relative group"
                    on:click={() => handleSort('logs', 'eventType')}
                  >
                    <div class="flex items-center justify-between pr-2">
                      <span>入力方法</span>
                      <span class="text-xs font-mono text-slate-600 group-hover:text-slate-400">↕</span>
                    </div>
                    <div class="resizer" on:mousedown={(e) => startResize(e, 'logs', 'method')} />
                  </th>

                  <th 
                    class="p-3.5 rounded-r-xl cursor-pointer hover:bg-slate-800/90 transition relative group"
                    on:click={() => handleSort('logs', 'duration')}
                  >
                    <div class="flex items-center justify-between pr-2">
                      <span>滞在時間</span>
                      <span class="text-xs font-mono {logsSortKey === 'duration' ? 'text-blue-400 font-bold' : 'text-slate-600 group-hover:text-slate-400'}">
                        {logsSortKey === 'duration' ? (logsSortDir === 'asc' ? '▲' : '▼') : '↕'}
                      </span>
                    </div>
                    <div class="resizer" on:mousedown={(e) => startResize(e, 'logs', 'duration')} />
                  </th>
                </tr>
              </thead>
              <tbody class="divide-y divide-slate-800/60">
                {#each sortedLogs as log}
                  <tr class="hover:bg-slate-800/40 transition">
                    <td class="p-3.5 font-mono text-xs text-slate-400 truncate">{formatDate(log.timestamp)}</td>
                    <td class="p-3.5 font-bold text-white text-base truncate">{log.userName || '未登録'}</td>
                    <td class="p-3.5 truncate">
                      <span class={`px-2.5 py-1 rounded-full text-xs font-semibold ${log.roleCode === 0 ? 'bg-purple-950 text-purple-300 border border-purple-800' : (log.roleCode === 9 ? 'bg-emerald-950 text-emerald-300 border border-emerald-800' : 'bg-slate-800 text-slate-300')}`}>
                        {log.roleName || '-'}
                      </span>
                    </td>
                    <td class="p-3.5 font-mono text-slate-300 truncate">{log.studentNo || '-'}</td>
                    <td class="p-3.5 truncate">
                      {#if log.eventType === 'entry'}
                        <span class="inline-flex items-center px-3 py-1 rounded-xl text-xs font-black bg-blue-950/80 text-blue-400 border border-blue-700/60">
                          🔵 入室
                        </span>
                      {:else if log.eventType === 'exit'}
                        <span class="inline-flex items-center px-3 py-1 rounded-xl text-xs font-black bg-amber-950/80 text-amber-400 border border-amber-700/60">
                          🟠 退室
                        </span>
                      {:else if log.eventType === 'force_exit'}
                        <span class="inline-flex items-center px-3 py-1 rounded-xl text-xs font-black bg-rose-950/80 text-rose-400 border border-rose-700/60">
                          ⚠️ 強制退室
                        </span>
                      {/if}
                    </td>
                    <td class="p-3.5 truncate">
                      {#if log.eventType === 'force_exit'}
                        <span class="text-xs text-rose-300 font-semibold flex items-center gap-1">
                          ⚙️ システム自動 (23:00)
                        </span>
                      {:else}
                        <span class="text-xs text-slate-400">
                          🪪 カード読み取り
                        </span>
                      {/if}
                    </td>
                    <td class="p-3.5 font-mono text-slate-400 font-semibold truncate">{log.durationText || '-'}</td>
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
            <table class="text-left text-sm text-slate-300 border-collapse" style="table-layout: fixed; width: max-content; min-width: 100%;">
              <colgroup>
                <col style="width: {insideWidths.userName}px;">
                <col style="width: {insideWidths.roleName}px;">
                <col style="width: {insideWidths.studentNo}px;">
                <col style="width: {insideWidths.lastEventTime}px;">
                <col style="width: {insideWidths.stayDuration}px;">
              </colgroup>
              <thead class="text-xs text-slate-400 uppercase bg-slate-800/60 border-b border-slate-800 select-none">
                <tr>
                  <th 
                    class="p-3.5 rounded-l-xl cursor-pointer hover:bg-slate-800/90 transition relative group"
                    on:click={() => handleSort('inside', 'userName')}
                  >
                    <div class="flex items-center justify-between pr-2">
                      <span>氏名</span>
                      <span class="text-xs font-mono {insideSortKey === 'userName' ? 'text-blue-400 font-bold' : 'text-slate-600 group-hover:text-slate-400'}">
                        {insideSortKey === 'userName' ? (insideSortDir === 'asc' ? '▲' : '▼') : '↕'}
                      </span>
                    </div>
                    <div class="resizer" on:mousedown={(e) => startResize(e, 'inside', 'userName')} />
                  </th>

                  <th 
                    class="p-3.5 cursor-pointer hover:bg-slate-800/90 transition relative group"
                    on:click={() => handleSort('inside', 'roleName')}
                  >
                    <div class="flex items-center justify-between pr-2">
                      <span>区分</span>
                      <span class="text-xs font-mono {insideSortKey === 'roleName' ? 'text-blue-400 font-bold' : 'text-slate-600 group-hover:text-slate-400'}">
                        {insideSortKey === 'roleName' ? (insideSortDir === 'asc' ? '▲' : '▼') : '↕'}
                      </span>
                    </div>
                    <div class="resizer" on:mousedown={(e) => startResize(e, 'inside', 'roleName')} />
                  </th>

                  <th 
                    class="p-3.5 cursor-pointer hover:bg-slate-800/90 transition relative group"
                    on:click={() => handleSort('inside', 'studentNo')}
                  >
                    <div class="flex items-center justify-between pr-2">
                      <span>学籍/職員番号</span>
                      <span class="text-xs font-mono {insideSortKey === 'studentNo' ? 'text-blue-400 font-bold' : 'text-slate-600 group-hover:text-slate-400'}">
                        {insideSortKey === 'studentNo' ? (insideSortDir === 'asc' ? '▲' : '▼') : '↕'}
                      </span>
                    </div>
                    <div class="resizer" on:mousedown={(e) => startResize(e, 'inside', 'studentNo')} />
                  </th>

                  <th 
                    class="p-3.5 cursor-pointer hover:bg-slate-800/90 transition relative group"
                    on:click={() => handleSort('inside', 'lastEventTime')}
                  >
                    <div class="flex items-center justify-between pr-2">
                      <span>入室時刻</span>
                      <span class="text-xs font-mono {insideSortKey === 'lastEventTime' ? 'text-blue-400 font-bold' : 'text-slate-600 group-hover:text-slate-400'}">
                        {insideSortKey === 'lastEventTime' ? (insideSortDir === 'asc' ? '▲' : '▼') : '↕'}
                      </span>
                    </div>
                    <div class="resizer" on:mousedown={(e) => startResize(e, 'inside', 'lastEventTime')} />
                  </th>

                  <th 
                    class="p-3.5 rounded-r-xl cursor-pointer hover:bg-slate-800/90 transition relative group"
                    on:click={() => handleSort('inside', 'stayDuration')}
                  >
                    <div class="flex items-center justify-between pr-2">
                      <span>現在の滞在時間</span>
                      <span class="text-xs font-mono {insideSortKey === 'stayDuration' ? 'text-blue-400 font-bold' : 'text-slate-600 group-hover:text-slate-400'}">
                        {insideSortKey === 'stayDuration' ? (insideSortDir === 'asc' ? '▲' : '▼') : '↕'}
                      </span>
                    </div>
                    <div class="resizer" on:mousedown={(e) => startResize(e, 'inside', 'stayDuration')} />
                  </th>
                </tr>
              </thead>
              <tbody class="divide-y divide-slate-800/60">
                {#each sortedInsideUsers as u}
                  <tr class="hover:bg-slate-800/40 transition">
                    <td class="p-3.5 font-bold text-white text-base flex items-center gap-2 truncate">
                      <span class="w-2.5 h-2.5 rounded-full bg-emerald-400 animate-pulse shrink-0"></span>
                      <span class="truncate">{u.userName || '未登録'}</span>
                    </td>
                    <td class="p-3.5 truncate">
                      <span class="px-2.5 py-1 rounded-full text-xs font-semibold bg-slate-800 text-slate-300">
                        {u.roleName || '-'}
                      </span>
                    </td>
                    <td class="p-3.5 font-mono text-slate-300 truncate">{u.studentNo || '-'}</td>
                    <td class="p-3.5 font-mono text-xs text-slate-400 truncate">{formatTime(u.lastEventTime)}</td>
                    <td class="p-3.5 font-bold text-blue-400 text-sm truncate">{u.stayDuration || '数秒'}</td>
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
            <table class="text-left text-sm text-slate-300 border-collapse" style="table-layout: fixed; width: max-content; min-width: 100%;">
              <colgroup>
                <col style="width: {usersWidths.adminNo}px;">
                <col style="width: {usersWidths.studentNo}px;">
                <col style="width: {usersWidths.name}px;">
                <col style="width: {usersWidths.furigana}px;">
                <col style="width: {usersWidths.gender}px;">
                <col style="width: {usersWidths.roleName}px;">
                <col style="width: {usersWidths.contact}px;">
                <col style="width: {usersWidths.purpose}px;">
                <col style="width: {usersWidths.cardId}px;">
                <col style="width: {usersWidths.actions}px;">
              </colgroup>
              <thead class="text-xs text-slate-400 uppercase bg-slate-800/60 border-b border-slate-800 select-none">
                <tr>
                  <th 
                    class="p-3.5 rounded-l-xl cursor-pointer hover:bg-slate-800/90 transition relative group"
                    on:click={() => handleSort('users', 'adminNo')}
                  >
                    <div class="flex items-center justify-between pr-2">
                      <span>No.</span>
                      <span class="text-xs font-mono {usersSortKey === 'adminNo' ? 'text-blue-400 font-bold' : 'text-slate-600 group-hover:text-slate-400'}">
                        {usersSortKey === 'adminNo' ? (usersSortDir === 'asc' ? '▲' : '▼') : '↕'}
                      </span>
                    </div>
                    <div class="resizer" on:mousedown={(e) => startResize(e, 'users', 'adminNo')} />
                  </th>

                  <th 
                    class="p-3.5 cursor-pointer hover:bg-slate-800/90 transition relative group"
                    on:click={() => handleSort('users', 'studentNo')}
                  >
                    <div class="flex items-center justify-between pr-2">
                      <span>学籍/職員番号</span>
                      <span class="text-xs font-mono {usersSortKey === 'studentNo' ? 'text-blue-400 font-bold' : 'text-slate-600 group-hover:text-slate-400'}">
                        {usersSortKey === 'studentNo' ? (usersSortDir === 'asc' ? '▲' : '▼') : '↕'}
                      </span>
                    </div>
                    <div class="resizer" on:mousedown={(e) => startResize(e, 'users', 'studentNo')} />
                  </th>

                  <th 
                    class="p-3.5 cursor-pointer hover:bg-slate-800/90 transition relative group"
                    on:click={() => handleSort('users', 'name')}
                  >
                    <div class="flex items-center justify-between pr-2">
                      <span>氏名</span>
                      <span class="text-xs font-mono {usersSortKey === 'name' ? 'text-blue-400 font-bold' : 'text-slate-600 group-hover:text-slate-400'}">
                        {usersSortKey === 'name' ? (usersSortDir === 'asc' ? '▲' : '▼') : '↕'}
                      </span>
                    </div>
                    <div class="resizer" on:mousedown={(e) => startResize(e, 'users', 'name')} />
                  </th>

                  <th 
                    class="p-3.5 cursor-pointer hover:bg-slate-800/90 transition relative group"
                    on:click={() => handleSort('users', 'furigana')}
                  >
                    <div class="flex items-center justify-between pr-2">
                      <span>フリガナ</span>
                      <span class="text-xs font-mono {usersSortKey === 'furigana' ? 'text-blue-400 font-bold' : 'text-slate-600 group-hover:text-slate-400'}">
                        {usersSortKey === 'furigana' ? (usersSortDir === 'asc' ? '▲' : '▼') : '↕'}
                      </span>
                    </div>
                    <div class="resizer" on:mousedown={(e) => startResize(e, 'users', 'furigana')} />
                  </th>

                  <th 
                    class="p-3.5 cursor-pointer hover:bg-slate-800/90 transition relative group"
                    on:click={() => handleSort('users', 'gender')}
                  >
                    <div class="flex items-center justify-between pr-2">
                      <span>性別</span>
                      <span class="text-xs font-mono {usersSortKey === 'gender' ? 'text-blue-400 font-bold' : 'text-slate-600 group-hover:text-slate-400'}">
                        {usersSortKey === 'gender' ? (usersSortDir === 'asc' ? '▲' : '▼') : '↕'}
                      </span>
                    </div>
                    <div class="resizer" on:mousedown={(e) => startResize(e, 'users', 'gender')} />
                  </th>

                  <th 
                    class="p-3.5 cursor-pointer hover:bg-slate-800/90 transition relative group"
                    on:click={() => handleSort('users', 'roleCode')}
                  >
                    <div class="flex items-center justify-between pr-2">
                      <span>区分</span>
                      <span class="text-xs font-mono {usersSortKey === 'roleCode' ? 'text-blue-400 font-bold' : 'text-slate-600 group-hover:text-slate-400'}">
                        {usersSortKey === 'roleCode' ? (usersSortDir === 'asc' ? '▲' : '▼') : '↕'}
                      </span>
                    </div>
                    <div class="resizer" on:mousedown={(e) => startResize(e, 'users', 'roleCode')} />
                  </th>

                  <th 
                    class="p-3.5 cursor-pointer hover:bg-slate-800/90 transition relative group"
                    on:click={() => handleSort('users', 'contact')}
                  >
                    <div class="flex items-center justify-between pr-2">
                      <span>連絡先</span>
                      <span class="text-xs font-mono {usersSortKey === 'contact' ? 'text-blue-400 font-bold' : 'text-slate-600 group-hover:text-slate-400'}">
                        {usersSortKey === 'contact' ? (usersSortDir === 'asc' ? '▲' : '▼') : '↕'}
                      </span>
                    </div>
                    <div class="resizer" on:mousedown={(e) => startResize(e, 'users', 'contact')} />
                  </th>

                  <th 
                    class="p-3.5 cursor-pointer hover:bg-slate-800/90 transition relative group"
                    on:click={() => handleSort('users', 'purpose')}
                  >
                    <div class="flex items-center justify-between pr-2">
                      <span>利用目的</span>
                      <span class="text-xs font-mono {usersSortKey === 'purpose' ? 'text-blue-400 font-bold' : 'text-slate-600 group-hover:text-slate-400'}">
                        {usersSortKey === 'purpose' ? (usersSortDir === 'asc' ? '▲' : '▼') : '↕'}
                      </span>
                    </div>
                    <div class="resizer" on:mousedown={(e) => startResize(e, 'users', 'purpose')} />
                  </th>

                  <th 
                    class="p-3.5 cursor-pointer hover:bg-slate-800/90 transition relative group"
                    on:click={() => handleSort('users', 'cardId')}
                  >
                    <div class="flex items-center justify-between pr-2">
                      <span>識別カードID</span>
                      <span class="text-xs font-mono {usersSortKey === 'cardId' ? 'text-blue-400 font-bold' : 'text-slate-600 group-hover:text-slate-400'}">
                        {usersSortKey === 'cardId' ? (usersSortDir === 'asc' ? '▲' : '▼') : '↕'}
                      </span>
                    </div>
                    <div class="resizer" on:mousedown={(e) => startResize(e, 'users', 'cardId')} />
                  </th>

                  <th class="p-3.5 text-right rounded-r-xl relative">
                    <span>操作</span>
                  </th>
                </tr>
              </thead>
              <tbody class="divide-y divide-slate-800/60">
                {#each sortedUsers as u}
                  <tr class="hover:bg-slate-800/40 transition">
                    <td class="p-3.5 font-mono text-xs text-slate-400 truncate">{u.adminNo || '-'}</td>
                    <td class="p-3.5 font-mono font-bold text-white text-sm truncate">{u.studentNo || '-'}</td>
                    <td class="p-3.5 font-bold text-white text-base truncate">{u.name}</td>
                    <td class="p-3.5 text-xs text-slate-400 truncate">{u.furigana || '-'}</td>
                    <td class="p-3.5 text-xs truncate">
                      {#if u.gender}
                        <span class={`px-2 py-0.5 rounded-full text-xs font-semibold ${u.gender === '男' ? 'bg-blue-950 text-blue-300 border border-blue-800' : (u.gender === '女' ? 'bg-pink-950 text-pink-300 border border-pink-800' : 'bg-slate-800 text-slate-300')}`}>
                          {u.gender}
                        </span>
                      {:else}
                        <span class="text-slate-500">-</span>
                      {/if}
                    </td>
                    <td class="p-3.5 truncate">
                      <span class={`px-2.5 py-1 rounded-full text-xs font-semibold ${u.roleCode === 0 ? 'bg-purple-950 text-purple-300 border border-purple-800' : (u.roleCode === 9 ? 'bg-emerald-950 text-emerald-300 border border-emerald-800' : 'bg-slate-800 text-slate-300')}`}>
                        {u.roleName}
                      </span>
                    </td>
                    <td class="p-3.5 text-xs text-slate-300 truncate" title={u.contact || ''}>{u.contact || '-'}</td>
                    <td class="p-3.5 text-xs text-slate-300 truncate" title={u.purpose || ''}>{u.purpose || '-'}</td>
                    <td class="p-3.5 font-mono text-xs text-slate-400 truncate">{u.cardId}</td>
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

      <!-- 4. 月別統計 -->
      {#if activeTab === 'monthly'}
        {#if !monthlyStats.rows || monthlyStats.rows.length === 0}
          <div class="text-center py-16 text-slate-500 text-sm">月別統計データはまだ集計されていません</div>
        {:else}
          <div class="overflow-x-auto">
            <table class="text-left text-sm text-slate-300 border-collapse" style="table-layout: fixed; width: max-content; min-width: 100%;">
              <colgroup>
                <col style="width: {monthlyWidths.yearMonth}px;">
                <col style="width: {monthlyWidths.roleOther}px;">
                <col style="width: {monthlyWidths.role0}px;">
                <col style="width: {monthlyWidths.role1}px;">
                <col style="width: {monthlyWidths.role9}px;">
                <col style="width: {monthlyWidths.monthlyTotal}px;">
                <col style="width: {monthlyWidths.quarterTotal}px;">
                <col style="width: {monthlyWidths.fiscalYearCumulativeTotal}px;">
              </colgroup>
              <thead class="text-xs text-slate-400 uppercase bg-slate-800/60 border-b border-slate-800 select-none">
                <tr>
                  <th 
                    class="p-3.5 rounded-l-xl cursor-pointer hover:bg-slate-800/90 transition relative group"
                    on:click={() => handleSort('monthly', 'yearMonth')}
                  >
                    <div class="flex items-center justify-between pr-2">
                      <span>年月</span>
                      <span class="text-xs font-mono {monthlySortKey === 'yearMonth' ? 'text-blue-400 font-bold' : 'text-slate-600 group-hover:text-slate-400'}">
                        {monthlySortKey === 'yearMonth' ? (monthlySortDir === 'asc' ? '▲' : '▼') : '↕'}
                      </span>
                    </div>
                    <div class="resizer" on:mousedown={(e) => startResize(e, 'monthly', 'yearMonth')} />
                  </th>

                  <th 
                    class="p-3.5 text-center cursor-pointer hover:bg-slate-800/90 transition relative group"
                    on:click={() => handleSort('monthly', 'roleOtherCount')}
                  >
                    <div class="flex items-center justify-center gap-1">
                      <span>-</span>
                      <span class="text-xs font-mono {monthlySortKey === 'roleOtherCount' ? 'text-blue-400 font-bold' : 'text-slate-600 group-hover:text-slate-400'}">
                        {monthlySortKey === 'roleOtherCount' ? (monthlySortDir === 'asc' ? '▲' : '▼') : '↕'}
                      </span>
                    </div>
                    <div class="resizer" on:mousedown={(e) => startResize(e, 'monthly', 'roleOther')} />
                  </th>

                  <th 
                    class="p-3.5 text-center cursor-pointer hover:bg-slate-800/90 transition relative group"
                    on:click={() => handleSort('monthly', 'role0Count')}
                  >
                    <div class="flex items-center justify-center gap-1">
                      <span>0（教職員）</span>
                      <span class="text-xs font-mono {monthlySortKey === 'role0Count' ? 'text-blue-400 font-bold' : 'text-slate-600 group-hover:text-slate-400'}">
                        {monthlySortKey === 'role0Count' ? (monthlySortDir === 'asc' ? '▲' : '▼') : '↕'}
                      </span>
                    </div>
                    <div class="resizer" on:mousedown={(e) => startResize(e, 'monthly', 'role0')} />
                  </th>

                  <th 
                    class="p-3.5 text-center cursor-pointer hover:bg-slate-800/90 transition relative group"
                    on:click={() => handleSort('monthly', 'role1Count')}
                  >
                    <div class="flex items-center justify-center gap-1">
                      <span>1（学生）</span>
                      <span class="text-xs font-mono {monthlySortKey === 'role1Count' ? 'text-blue-400 font-bold' : 'text-slate-600 group-hover:text-slate-400'}">
                        {monthlySortKey === 'role1Count' ? (monthlySortDir === 'asc' ? '▲' : '▼') : '↕'}
                      </span>
                    </div>
                    <div class="resizer" on:mousedown={(e) => startResize(e, 'monthly', 'role1')} />
                  </th>

                  <th 
                    class="p-3.5 text-center cursor-pointer hover:bg-slate-800/90 transition relative group"
                    on:click={() => handleSort('monthly', 'role9Count')}
                  >
                    <div class="flex items-center justify-center gap-1">
                      <span>9（スタッフ）</span>
                      <span class="text-xs font-mono {monthlySortKey === 'role9Count' ? 'text-blue-400 font-bold' : 'text-slate-600 group-hover:text-slate-400'}">
                        {monthlySortKey === 'role9Count' ? (monthlySortDir === 'asc' ? '▲' : '▼') : '↕'}
                      </span>
                    </div>
                    <div class="resizer" on:mousedown={(e) => startResize(e, 'monthly', 'role9')} />
                  </th>

                  <th 
                    class="p-3.5 text-center bg-slate-800/40 cursor-pointer hover:bg-slate-800/90 transition relative group"
                    on:click={() => handleSort('monthly', 'monthlyTotal')}
                  >
                    <div class="flex items-center justify-center gap-1">
                      <span>月計</span>
                      <span class="text-xs font-mono {monthlySortKey === 'monthlyTotal' ? 'text-blue-400 font-bold' : 'text-slate-600 group-hover:text-slate-400'}">
                        {monthlySortKey === 'monthlyTotal' ? (monthlySortDir === 'asc' ? '▲' : '▼') : '↕'}
                      </span>
                    </div>
                    <div class="resizer" on:mousedown={(e) => startResize(e, 'monthly', 'monthlyTotal')} />
                  </th>

                  <th 
                    class="p-3.5 text-center cursor-pointer hover:bg-slate-800/90 transition relative group"
                    on:click={() => handleSort('monthly', 'quarterTotal')}
                  >
                    <div class="flex items-center justify-between pr-2">
                      <span>四半期</span>
                      <span class="text-xs font-mono {monthlySortKey === 'quarterTotal' ? 'text-blue-400 font-bold' : 'text-slate-600 group-hover:text-slate-400'}">
                        {monthlySortKey === 'quarterTotal' ? (monthlySortDir === 'asc' ? '▲' : '▼') : '↕'}
                      </span>
                    </div>
                    <div class="resizer" on:mousedown={(e) => startResize(e, 'monthly', 'quarterTotal')} />
                  </th>

                  <th 
                    class="p-3.5 text-right rounded-r-xl cursor-pointer hover:bg-slate-800/90 transition relative group"
                    on:click={() => handleSort('monthly', 'fiscalYearCumulativeTotal')}
                  >
                    <div class="flex items-center justify-end gap-1">
                      <span>当月までの合計</span>
                      <span class="text-xs font-mono {monthlySortKey === 'fiscalYearCumulativeTotal' ? 'text-blue-400 font-bold' : 'text-slate-600 group-hover:text-slate-400'}">
                        {monthlySortKey === 'fiscalYearCumulativeTotal' ? (monthlySortDir === 'asc' ? '▲' : '▼') : '↕'}
                      </span>
                    </div>
                    <div class="resizer" on:mousedown={(e) => startResize(e, 'monthly', 'fiscalYearCumulativeTotal')} />
                  </th>
                </tr>
              </thead>
              <tbody class="divide-y divide-slate-800/60">
                {#each sortedMonthlyRows as r}
                  <tr class="hover:bg-slate-800/40 transition">
                    <td class="p-3.5 font-mono font-bold text-white text-base truncate">{r.yearMonth}</td>
                    <td class="p-3.5 text-center font-mono text-slate-400 truncate">{r.roleOtherCount || ''}</td>
                    <td class="p-3.5 text-center font-mono font-semibold text-purple-300 truncate">{r.role0Count || ''}</td>
                    <td class="p-3.5 text-center font-mono font-semibold text-blue-300 truncate">{r.role1Count || ''}</td>
                    <td class="p-3.5 text-center font-mono font-semibold text-emerald-300 truncate">{r.role9Count || ''}</td>
                    <td class="p-3.5 text-center font-mono font-bold text-white bg-slate-800/30 truncate">{r.monthlyTotal}</td>
                    <td class="p-3.5 text-center truncate">
                      {#if r.quarterPeriod}
                        <div class="flex items-center justify-center gap-2">
                          <span class="text-xs text-slate-400">{r.quarterPeriod}</span>
                          <span class="font-mono font-bold text-slate-200">{r.quarterTotal}</span>
                        </div>
                      {/if}
                    </td>
                    <td class="p-3.5 text-right font-mono font-bold text-emerald-400 text-base truncate">{r.fiscalYearCumulativeTotal}</td>
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
    <div class="fixed inset-0 bg-slate-950/80 backdrop-blur-md flex items-center justify-center p-4 z-50 animate-fade">
      <div class="bg-slate-900 border border-slate-700 rounded-3xl p-6 sm:p-8 max-w-lg w-full shadow-2xl relative max-h-[90vh] overflow-y-auto">
        <h3 class="text-xl font-black text-white mb-2 flex items-center gap-2">
          {editingUser ? '👤 利用者情報の編集' : '✨ 新規利用者の登録'}
        </h3>
        <p class="text-xs text-slate-400 mb-6">
          {editingUser ? '登録情報を更新します' : '新しい利用者をマスターデータへ追加します'}
        </p>

        {#if formError}
          <div class="bg-rose-950/60 border border-rose-800 text-rose-300 text-xs px-4 py-3 rounded-xl mb-4">
            {formError}
          </div>
        {/if}

        <div class="space-y-4">
          <!-- 1. 学籍番号 (必須) -->
          <div>
            <label for="form-student-no" class="block text-xs font-semibold text-slate-400 uppercase mb-1">
              学籍番号 / 職員番号 <span class="text-rose-400">*</span>
            </label>
            <input 
              id="form-student-no"
              type="text" 
              bind:value={formStudentNo} 
              placeholder="例: B2026001"
              class="w-full bg-slate-800 border border-slate-700 text-white rounded-xl px-3.5 py-2.5 font-mono text-sm focus:outline-none focus:border-blue-500"
            />
          </div>

          <!-- 2. 氏名 (必須) -->
          <div>
            <label for="form-user-name" class="block text-xs font-semibold text-slate-400 uppercase mb-1">
              氏名 <span class="text-rose-400">*</span>
            </label>
            <input 
              id="form-user-name"
              type="text" 
              bind:value={formName} 
              placeholder="例: 山田 太郎"
              class="w-full bg-slate-800 border border-slate-700 text-white rounded-xl px-3.5 py-2.5 text-sm focus:outline-none focus:border-blue-500"
            />
          </div>

          <!-- 3. フリガナ -->
          <div>
            <label for="form-furigana" class="block text-xs font-semibold text-slate-400 uppercase mb-1">
              フリガナ
            </label>
            <input 
              id="form-furigana"
              type="text" 
              bind:value={formFurigana} 
              placeholder="例: ヤマダ タロウ"
              class="w-full bg-slate-800 border border-slate-700 text-white rounded-xl px-3.5 py-2.5 text-sm focus:outline-none focus:border-blue-500"
            />
          </div>

          <!-- 4. 性別 & 5. 区分 -->
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label for="form-gender" class="block text-xs font-semibold text-slate-400 uppercase mb-1">性別</label>
              <select 
                id="form-gender"
                bind:value={formGender} 
                class="w-full bg-slate-800 border border-slate-700 text-white rounded-xl px-3.5 py-2.5 text-sm focus:outline-none focus:border-blue-500"
              >
                <option value="男">男</option>
                <option value="女">女</option>
              </select>
            </div>

            <div>
              <label for="form-role-name" class="block text-xs font-semibold text-slate-400 uppercase mb-1">区分</label>
              <select 
                id="form-role-name"
                bind:value={formRoleName} 
                on:change={handleRoleChange}
                class="w-full bg-slate-800 border border-slate-700 text-white rounded-xl px-3.5 py-2.5 text-sm focus:outline-none focus:border-blue-500"
              >
                <option value="学生">学生</option>
                <option value="教職員">教職員</option>
                <option value="学生スタッフ">学生スタッフ</option>
              </select>
            </div>
          </div>

          <!-- 6. カードID (任意) -->
          <div>
            <div class="flex items-center justify-between mb-1">
              <label for="form-card-id" class="block text-xs font-semibold text-slate-400 uppercase">
                識別カードID (NFC IDm / 磁気カード番号)
              </label>
              <span class="text-[10px] text-slate-400 bg-slate-800 px-2 py-0.5 rounded">任意 (未入力時は学籍番号)</span>
            </div>
            <input 
              id="form-card-id"
              type="text" 
              bind:value={formCardId} 
              placeholder="空欄の場合は学籍番号と同じ値を扱います (NFC/磁気併用時に指定)"
              class="w-full bg-slate-800 border border-slate-700 text-white rounded-xl px-3.5 py-2.5 font-mono text-sm focus:outline-none focus:border-blue-500"
            />
            <p class="text-[11px] text-slate-400 mt-1">
              ※カードIDに入力がある場合、NFCと磁気リーダーの双方で学籍番号・カードIDから認証できます。
            </p>
          </div>

          <!-- 7. 連絡先 (任意) -->
          <div>
            <div class="flex items-center justify-between mb-1">
              <label for="form-contact" class="block text-xs font-semibold text-slate-400 uppercase">
                連絡先
              </label>
              <span class="text-[10px] text-slate-400 bg-slate-800 px-2 py-0.5 rounded">任意</span>
            </div>
            <input 
              id="form-contact"
              type="text" 
              bind:value={formContact} 
              placeholder="例: 090-1234-5678, user@example.com"
              class="w-full bg-slate-800 border border-slate-700 text-white rounded-xl px-3.5 py-2.5 text-sm focus:outline-none focus:border-blue-500"
            />
          </div>

          <!-- 8. 利用目的 (任意) -->
          <div>
            <div class="flex items-center justify-between mb-1">
              <label for="form-purpose" class="block text-xs font-semibold text-slate-400 uppercase">
                利用目的
              </label>
              <span class="text-[10px] text-slate-400 bg-slate-800 px-2 py-0.5 rounded">任意</span>
            </div>
            <input 
              id="form-purpose"
              type="text" 
              bind:value={formPurpose} 
              placeholder="例: 自習、プロジェクト活動、研究利用"
              class="w-full bg-slate-800 border border-slate-700 text-white rounded-xl px-3.5 py-2.5 text-sm focus:outline-none focus:border-blue-500"
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

  .resizer {
    position: absolute;
    top: 0;
    right: 0;
    width: 6px;
    height: 100%;
    cursor: col-resize;
    user-select: none;
    touch-action: none;
    z-index: 10;
  }

  .resizer:hover {
    background-color: #3b82f6;
  }
</style>
