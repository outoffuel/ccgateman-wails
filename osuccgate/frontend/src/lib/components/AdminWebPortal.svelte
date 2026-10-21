<script>
  import { onMount, onDestroy } from 'svelte';

  export let onBackToKiosk = null;

  // 画面タブ
  let activeTab = 'dashboard'; // 'dashboard' | 'logs' | 'users'

  // 現在時刻
  let currentTimeString = '';
  let timerId = null;

  function updateTime() {
    const now = new Date();
    const y = now.getFullYear();
    const m = String(now.getMonth() + 1).padStart(2, '0');
    const d = String(now.getDate()).padStart(2, '0');
    const days = ['日', '月', '火', '水', '木', '金', '土'];
    const day = days[now.getDay()];
    const hh = String(now.getHours()).padStart(2, '0');
    const mm = String(now.getMinutes()).padStart(2, '0');
    const ss = String(now.getSeconds()).padStart(2, '0');
    currentTimeString = `${y}/${m}/${d}(${day}) ${hh}:${mm}:${ss}`;
  }

  // トースト通知
  let toasts = [];
  function showToast(message, type = 'info') {
    const id = Date.now() + Math.random();
    toasts = [...toasts, { id, message, type }];
    setTimeout(() => {
      toasts = toasts.filter(t => t.id !== id);
    }, 3000);
  }

  // 1. 対象室の状態（単一室）
  let roomState = {
    name: '研究室',
    isLocked: false,
    capacity: 10,
    currentInside: 2,
    insideUsers: ['山田 太郎', '佐藤 一郎']
  };

  function toggleDoor(toLocked) {
    roomState.isLocked = toLocked;
    if (toLocked) {
      showToast('研究室のドアを【施錠】しました', 'info');
    } else {
      showToast('研究室のドアを【解錠】しました', 'success');
    }
  }

  // 2. 入退室履歴データ
  let logsData = [
    { id: 1, timestamp: "2026/10/22 14:22", name: "山田 太郎", uid: "A0012345", type: "入室", result: "成功", note: "ICカード" },
    { id: 2, timestamp: "2026/10/22 13:58", name: "鈴木 花子", uid: "A0023456", type: "退室", result: "成功", note: "顔認証" },
    { id: 3, timestamp: "2026/10/22 13:41", name: "佐藤 一郎", uid: "A0034567", type: "入室", result: "成功", note: "ICカード" },
    { id: 4, timestamp: "2026/10/22 12:30", name: "高橋 美咲", uid: "A0045678", type: "退室", result: "成功", note: "スマホ(NFC)" },
    { id: 5, timestamp: "2026/10/22 11:05", name: "田中 健二", uid: "A0056789", type: "入室", result: "失敗", note: "未登録カード" },
    { id: 6, timestamp: "2026/10/22 10:45", name: "鈴木 花子", uid: "A0023456", type: "入室", result: "成功", note: "顔認証" },
    { id: 7, timestamp: "2026/10/22 10:15", name: "佐藤 一郎", uid: "A0034567", type: "退室", result: "成功", note: "ICカード" },
    { id: 8, timestamp: "2026/10/22 09:40", name: "渡辺 拓海", uid: "A0078901", type: "入室", result: "成功", note: "暗証番号" },
    { id: 9, timestamp: "2026/10/22 09:10", name: "高橋 美咲", uid: "A0045678", type: "入室", result: "成功", note: "スマホ(NFC)" },
    { id: 10, timestamp: "2026/10/22 08:50", name: "伊藤 翼",   uid: "A0067890", type: "入室", result: "成功", note: "ICカード" },
    { id: 11, timestamp: "2026/10/22 08:30", name: "山田 太郎", uid: "A0012345", type: "退室", result: "成功", note: "ICカード" },
    { id: 12, timestamp: "2026/10/22 08:05", name: "山田 太郎", uid: "A0012345", type: "入室", result: "成功", note: "ICカード" },
    { id: 13, timestamp: "2026/10/21 19:40", name: "佐藤 一郎", uid: "A0034567", type: "退室", result: "成功", note: "ICカード" },
    { id: 14, timestamp: "2026/10/21 18:30", name: "鈴木 花子", uid: "A0023456", type: "退室", result: "成功", note: "顔認証" },
    { id: 15, timestamp: "2026/10/21 17:15", name: "渡辺 拓海", uid: "A0078901", type: "退室", result: "成功", note: "暗証番号" },
    { id: 16, timestamp: "2026/10/21 16:50", name: "伊藤 翼",   uid: "A0067890", type: "退室", result: "成功", note: "ICカード" },
    { id: 17, timestamp: "2026/10/21 15:20", name: "高橋 美咲", uid: "A0045678", type: "入室", result: "成功", note: "スマホ(NFC)" },
    { id: 18, timestamp: "2026/10/21 14:10", name: "田中 健二", uid: "A0056789", type: "入室", result: "失敗", note: "アカウント無効" },
    { id: 19, timestamp: "2026/10/21 13:00", name: "山田 太郎", uid: "A0012345", type: "入室", result: "成功", note: "ICカード" },
    { id: 20, timestamp: "2026/10/21 12:00", name: "鈴木 花子", uid: "A0023456", type: "入室", result: "成功", note: "顔認証" },
    { id: 21, timestamp: "2026/10/21 11:30", name: "佐藤 一郎", uid: "A0034567", type: "入室", result: "成功", note: "ICカード" },
    { id: 22, timestamp: "2026/10/21 10:10", name: "渡辺 拓海", uid: "A0078901", type: "入室", result: "成功", note: "暗証番号" },
    { id: 23, timestamp: "2026/10/20 18:00", name: "山田 太郎", uid: "A0012345", type: "退室", result: "成功", note: "ICカード" },
    { id: 24, timestamp: "2026/10/20 17:30", name: "鈴木 花子", uid: "A0023456", type: "退室", result: "成功", note: "顔認証" },
    { id: 25, timestamp: "2026/10/20 16:20", name: "佐藤 一郎", uid: "A0034567", type: "退室", result: "成功", note: "ICカード" },
    { id: 26, timestamp: "2026/10/20 14:00", name: "伊藤 翼",   uid: "A0067890", type: "入室", result: "成功", note: "ICカード" },
    { id: 27, timestamp: "2026/10/20 10:30", name: "高橋 美咲", uid: "A0045678", type: "入室", result: "成功", note: "スマホ(NFC)" },
    { id: 28, timestamp: "2026/10/20 09:00", name: "山田 太郎", uid: "A0012345", type: "入室", result: "成功", note: "ICカード" }
  ];

  // 履歴フィルター条件
  let filterDateStart = "2026-10-22";
  let filterDateEnd = "2026-10-22";
  let filterType = "all";
  let filterResult = "all";
  let filterKeyword = "";
  let logCurrentPage = 1;
  const LOGS_PER_PAGE = 10;

  $: filteredLogs = logsData.filter(log => {
    const logDate = log.timestamp.split(' ')[0].replace(/\//g, '-');
    if (filterDateStart && logDate < filterDateStart) return false;
    if (filterDateEnd && logDate > filterDateEnd) return false;
    if (filterType !== 'all' && log.type !== filterType) return false;
    if (filterResult !== 'all' && log.result !== filterResult) return false;
    if (filterKeyword.trim()) {
      const kw = filterKeyword.trim().toLowerCase();
      const mName = log.name.toLowerCase().includes(kw);
      const mUid = log.uid.toLowerCase().includes(kw);
      if (!mName && !mUid) return false;
    }
    return true;
  });

  $: totalLogPages = Math.ceil(filteredLogs.length / LOGS_PER_PAGE) || 1;
  $: paginatedLogs = filteredLogs.slice((logCurrentPage - 1) * LOGS_PER_PAGE, logCurrentPage * LOGS_PER_PAGE);

  function clearLogFilters() {
    filterDateStart = "";
    filterDateEnd = "";
    filterType = "all";
    filterResult = "all";
    filterKeyword = "";
    logCurrentPage = 1;
    showToast("検索条件をクリアしました", "info");
  }

  function exportLogsCSV() {
    if (filteredLogs.length === 0) {
      showToast("出力対象の履歴がありません", "error");
      return;
    }
    const header = ["日時", "氏名", "ユーザーID", "種類", "認証結果", "備考"];
    const rows = filteredLogs.map(l => [
      `"${l.timestamp}"`,
      `"${l.name}"`,
      `"${l.uid}"`,
      `"${l.type}"`,
      `"${l.result}"`,
      `"${l.note}"`
    ]);

    const csvContent = "\uFEFF" + [header.join(","), ...rows.map(r => r.join(","))].join("\r\n");
    const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.setAttribute("href", url);
    link.setAttribute("download", `access_history_${new Date().toISOString().slice(0,10)}.csv`);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    showToast("CSVファイルを出力しました", "success");
  }

  // 3. ユーザー管理データ
  let usersData = [
    { id: 1, name: "山田 太郎", uid: "A0012345", role: "管理者", status: "有効" },
    { id: 2, name: "鈴木 花子", uid: "A0023456", role: "一般ユーザー", status: "有効" },
    { id: 3, name: "佐藤 一郎", uid: "A0034567", role: "一般ユーザー", status: "有効" },
    { id: 4, name: "高橋 美咲", uid: "A0045678", role: "一般ユーザー", status: "有効" },
    { id: 5, name: "田中 健二", uid: "A0056789", role: "一般ユーザー", status: "無効" },
    { id: 6, name: "伊藤 翼",   uid: "A0067890", role: "一般ユーザー", status: "有効" },
    { id: 7, name: "渡辺 拓海", uid: "A0078901", role: "管理者", status: "有効" }
  ];

  let userSearch = "";
  let userStatusFilter = "all"; // 'all' | 'active' | 'inactive'

  $: filteredUsers = usersData.filter(u => {
    if (userStatusFilter === 'active' && u.status !== '有効') return false;
    if (userStatusFilter === 'inactive' && u.status !== '無効') return false;
    if (userSearch.trim()) {
      const s = userSearch.trim().toLowerCase();
      if (!u.name.toLowerCase().includes(s) && !u.uid.toLowerCase().includes(s)) return false;
    }
    return true;
  });

  // モーダル管理
  let isUserModalOpen = false;
  let modalMode = 'create'; // 'create' | 'edit'
  let formId = null;
  let formName = '';
  let formUid = '';
  let formRole = '一般ユーザー';
  let formStatus = '有効';

  function openCreateUserModal() {
    modalMode = 'create';
    formId = null;
    formName = '';
    formUid = '';
    formRole = '一般ユーザー';
    formStatus = '有効';
    isUserModalOpen = true;
  }

  function openEditUserModal(user) {
    modalMode = 'edit';
    formId = user.id;
    formName = user.name;
    formUid = user.uid;
    formRole = user.role;
    formStatus = user.status;
    isUserModalOpen = true;
  }

  function closeUserModal() {
    isUserModalOpen = false;
  }

  function handleSaveUser() {
    if (!formName.trim() || !formUid.trim()) {
      alert("氏名とユーザーIDを入力してください。");
      return;
    }

    if (modalMode === 'edit') {
      usersData = usersData.map(u => {
        if (u.id === formId) {
          return {
            ...u,
            name: formName.trim(),
            uid: formUid.trim().toUpperCase(),
            role: formRole,
            status: formStatus
          };
        }
        return u;
      });
      showToast(`ユーザー「${formName}」を更新しました`, 'success');
    } else {
      const nextId = usersData.length > 0 ? Math.max(...usersData.map(u => u.id)) + 1 : 1;
      usersData = [{
        id: nextId,
        name: formName.trim(),
        uid: formUid.trim().toUpperCase(),
        role: formRole,
        status: formStatus
      }, ...usersData];
      showToast(`ユーザー「${formName}」を登録しました`, 'success');
    }
    closeUserModal();
  }

  function handleDeleteUser(user) {
    if (confirm(`ユーザー「${user.name} (${user.uid})」を削除してもよろしいですか？`)) {
      usersData = usersData.filter(u => u.id !== user.id);
      showToast(`ユーザー「${user.name}」を削除しました`, 'info');
    }
  }

  onMount(() => {
    updateTime();
    timerId = setInterval(updateTime, 1000);
  });

  onDestroy(() => {
    if (timerId) clearInterval(timerId);
  });
</script>

<div class="w-full h-screen flex overflow-hidden bg-slate-50 text-slate-800 select-none font-sans">
  
  <!-- 左側：固定サイドバー -->
  <aside class="w-64 bg-slate-900 text-slate-300 flex flex-col flex-shrink-0 z-20 border-r border-slate-800">
    <div class="h-16 flex items-center px-6 border-b border-slate-800 gap-3">
      <div class="w-9 h-9 rounded-lg bg-blue-600 flex items-center justify-center text-white shadow-md flex-shrink-0">
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"/>
        </svg>
      </div>
      <div class="overflow-hidden">
        <h1 class="text-sm font-bold text-white tracking-wide truncate">入退室管理システム</h1>
        <p class="text-[11px] text-blue-400 font-medium">対象室：研究室</p>
      </div>
    </div>

    <!-- ミニバッジ -->
    <div class="px-5 py-4">
      <div class="bg-slate-800/80 border border-slate-700/60 rounded-xl p-3 flex items-center justify-between shadow-xs">
        <div class="flex items-center gap-2">
          <span class="w-2.5 h-2.5 rounded-full {roomState.isLocked ? 'bg-slate-400' : 'bg-emerald-400'}"></span>
          <div>
            <div class="text-xs font-bold text-white">研究室</div>
            <div class="text-[10px] {roomState.isLocked ? 'text-slate-400' : 'text-emerald-400'} font-medium">
              {roomState.isLocked ? '施錠中' : '解錠中'}
            </div>
          </div>
        </div>
        <span class="text-[11px] bg-slate-700/90 text-slate-300 px-2 py-0.5 rounded font-mono font-medium">定員 {roomState.capacity}</span>
      </div>
    </div>

    <!-- メニュー -->
    <nav class="flex-1 px-3 space-y-1.5 overflow-y-auto">
      <button 
        on:click={() => activeTab = 'dashboard'}
        class="w-full flex items-center gap-3 px-4 py-3 text-sm font-semibold rounded-xl transition-all duration-150 {activeTab === 'dashboard' ? 'bg-blue-600 text-white shadow-sm' : 'text-slate-400 hover:text-white hover:bg-slate-800'}"
      >
        <svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z"/>
        </svg>
        <span>ダッシュボード</span>
      </button>

      <button 
        on:click={() => activeTab = 'logs'}
        class="w-full flex items-center gap-3 px-4 py-3 text-sm font-medium rounded-xl transition-all duration-150 {activeTab === 'logs' ? 'bg-blue-600 text-white shadow-sm' : 'text-slate-400 hover:text-white hover:bg-slate-800'}"
      >
        <svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
        </svg>
        <span>入退室履歴</span>
      </button>

      <button 
        on:click={() => activeTab = 'users'}
        class="w-full flex items-center gap-3 px-4 py-3 text-sm font-medium rounded-xl transition-all duration-150 {activeTab === 'users' ? 'bg-blue-600 text-white shadow-sm' : 'text-slate-400 hover:text-white hover:bg-slate-800'}"
      >
        <svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z"/>
        </svg>
        <span>ユーザー管理</span>
      </button>

      {#if onBackToKiosk}
        <div class="pt-4 mt-4 border-t border-slate-800">
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
      {/if}
    </nav>

    <div class="p-4 border-t border-slate-800 text-xs text-slate-400 bg-slate-900/50">
      <div class="flex justify-between items-center">
        <span>システム状態</span>
        <span class="text-emerald-400 font-semibold flex items-center gap-1.5">
          <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
          正常稼働中
        </span>
      </div>
      <div class="text-[11px] text-slate-500 mt-1">大学システム開発課題版</div>
    </div>
  </aside>

  <!-- メインエリア -->
  <div class="flex-1 flex flex-col min-w-0 overflow-hidden">
    <!-- ヘッダー -->
    <header class="h-16 bg-white border-b border-slate-200 flex items-center justify-between px-8 flex-shrink-0 z-10 shadow-xs">
      <div class="flex items-center gap-3">
        <h2 class="text-lg font-bold text-slate-800">
          {activeTab === 'dashboard' ? 'ダッシュボード' : activeTab === 'logs' ? '入退室履歴' : 'ユーザー管理'}
        </h2>
        <span class="text-xs bg-blue-50 text-blue-700 px-3 py-1 rounded-full font-semibold border border-blue-100 flex items-center gap-1.5">
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4"/>
          </svg>
          対象室：研究室
        </span>
      </div>

      <div class="flex items-center gap-6">
        <div class="hidden sm:flex items-center gap-2 text-xs text-slate-600 bg-slate-50 px-3.5 py-1.5 rounded-lg border border-slate-200">
          <svg class="w-3.5 h-3.5 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
          </svg>
          <span class="font-mono font-medium text-slate-700">{currentTimeString}</span>
        </div>

        <div class="flex items-center gap-3 pl-3 border-l border-slate-200">
          <div class="w-9 h-9 rounded-full bg-blue-600 text-white flex items-center justify-center font-bold text-sm shadow-sm ring-2 ring-blue-100">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"/>
            </svg>
          </div>
          <div>
            <div class="text-sm font-bold text-slate-800 leading-tight">管理者</div>
            <div class="text-[11px] text-emerald-600 flex items-center gap-1 font-medium mt-0.5">
              <span class="w-1.5 h-1.5 rounded-full bg-emerald-500"></span> ログイン中
            </div>
          </div>
        </div>
      </div>
    </header>

    <!-- コンテンツ -->
    <main class="flex-1 overflow-y-auto p-8 space-y-6 bg-slate-50">
      
      <!-- 1. ダッシュボード -->
      {#if activeTab === 'dashboard'}
        <div class="space-y-6">
          <!-- 4つの統計カード -->
          <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5">
            <div class="bg-white rounded-xl p-5 border border-slate-200/90 shadow-sm flex items-center justify-between">
              <div>
                <p class="text-xs font-bold text-slate-500 uppercase tracking-wider">現在の在室人数</p>
                <div class="mt-2 flex items-baseline gap-1.5">
                  <span class="text-3xl font-extrabold text-slate-800">{roomState.currentInside}</span>
                  <span class="text-sm text-slate-500 font-bold">人</span>
                </div>
                <p class="text-xs text-blue-600 font-medium mt-1">定員 10人 (利用率 20%)</p>
              </div>
              <div class="w-12 h-12 rounded-xl bg-blue-50 text-blue-600 flex items-center justify-center border border-blue-100">
                <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"/>
                </svg>
              </div>
            </div>

            <div class="bg-white rounded-xl p-5 border border-slate-200/90 shadow-sm flex items-center justify-between">
              <div>
                <p class="text-xs font-bold text-slate-500 uppercase tracking-wider">本日の入室</p>
                <div class="mt-2 flex items-baseline gap-1.5">
                  <span class="text-3xl font-extrabold text-slate-800">12</span>
                  <span class="text-sm text-slate-500 font-bold">回</span>
                </div>
                <p class="text-xs text-emerald-600 font-medium mt-1">最終 14:22 (山田 太郎)</p>
              </div>
              <div class="w-12 h-12 rounded-xl bg-emerald-50 text-emerald-600 flex items-center justify-center border border-emerald-100">
                <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 16l-4-4m0 0l4-4m-4 4h14m-5 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h7a3 3 0 013 3v1"/>
                </svg>
              </div>
            </div>

            <div class="bg-white rounded-xl p-5 border border-slate-200/90 shadow-sm flex items-center justify-between">
              <div>
                <p class="text-xs font-bold text-slate-500 uppercase tracking-wider">本日の退室</p>
                <div class="mt-2 flex items-baseline gap-1.5">
                  <span class="text-3xl font-extrabold text-slate-800">10</span>
                  <span class="text-sm text-slate-500 font-bold">回</span>
                </div>
                <p class="text-xs text-slate-500 font-medium mt-1">最終 13:58 (鈴木 花子)</p>
              </div>
              <div class="w-12 h-12 rounded-xl bg-rose-50 text-rose-600 flex items-center justify-center border border-rose-100">
                <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"/>
                </svg>
              </div>
            </div>

            <div class="bg-white rounded-xl p-5 border border-slate-200/90 shadow-sm flex items-center justify-between">
              <div>
                <p class="text-xs font-bold text-slate-500 uppercase tracking-wider">認証失敗</p>
                <div class="mt-2 flex items-baseline gap-1.5">
                  <span class="text-3xl font-extrabold text-rose-600">1</span>
                  <span class="text-sm text-slate-500 font-bold">件</span>
                </div>
                <p class="text-xs text-rose-500 font-medium mt-1">11:05 未登録カード</p>
              </div>
              <div class="w-12 h-12 rounded-xl bg-amber-50 text-amber-600 flex items-center justify-center border border-amber-100">
                <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
                </svg>
              </div>
            </div>
          </div>

          <!-- 対象室の状態カード -->
          <div class="bg-white rounded-xl border border-slate-200/90 shadow-sm p-6">
            <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-5 border-b border-slate-100">
              <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-lg bg-blue-50 text-blue-600 flex items-center justify-center border border-blue-100">
                  <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4"/>
                  </svg>
                </div>
                <div>
                  <h3 class="text-base font-bold text-slate-800">対象室の状態</h3>
                  <p class="text-xs text-slate-500">管理対象室のリアルタイム施錠・在室情報</p>
                </div>
              </div>

              <!-- 施錠 / 解錠 ボタン -->
              <div class="flex items-center gap-2.5">
                <span class="text-xs text-slate-500 font-semibold mr-1">遠隔操作:</span>
                <button 
                  on:click={() => toggleDoor(true)}
                  class="px-4 py-2 text-xs font-bold rounded-lg border border-slate-300 bg-white hover:bg-slate-50 text-slate-700 transition flex items-center gap-1.5 shadow-xs"
                >
                  <svg class="w-4 h-4 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"/>
                  </svg>
                  施錠する
                </button>
                <button 
                  on:click={() => toggleDoor(false)}
                  class="px-4 py-2 text-xs font-bold rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white transition flex items-center gap-1.5 shadow-sm"
                >
                  <svg class="w-4 h-4 text-emerald-100" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 11V7a4 4 0 118 0m-4 8v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2z"/>
                  </svg>
                  解錠する
                </button>
              </div>
            </div>

            <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6 pt-5">
              <div>
                <p class="text-xs font-bold text-slate-400">対象室</p>
                <p class="text-lg font-bold text-slate-800 mt-1">{roomState.name}</p>
                <p class="text-xs text-slate-500 mt-0.5">管理対象：1部屋のみ</p>
              </div>

              <div>
                <p class="text-xs font-bold text-slate-400">ドア状態</p>
                <div class="mt-1 flex items-center gap-2">
                  <span class="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-bold {roomState.isLocked ? 'bg-slate-100 text-slate-700 border border-slate-300' : 'bg-emerald-100 text-emerald-800 border border-emerald-200'}">
                    <span class="w-2 h-2 rounded-full {roomState.isLocked ? 'bg-slate-500' : 'bg-emerald-500'}"></span>
                    <span>{roomState.isLocked ? '施錠中' : '解錠中'}</span>
                  </span>
                </div>
                <p class="text-xs text-slate-400 mt-1">{roomState.isLocked ? '管理者操作により施錠されました' : '通常利用可能'}</p>
              </div>

              <div>
                <p class="text-xs font-bold text-slate-400">現在の在室人数</p>
                <p class="text-lg font-bold text-slate-800 mt-1">{roomState.currentInside} 人</p>
                <p class="text-xs text-slate-500 mt-0.5">在室者: {roomState.insideUsers.join(', ')}</p>
              </div>

              <div>
                <p class="text-xs font-bold text-slate-400">定員</p>
                <p class="text-lg font-bold text-slate-800 mt-1">{roomState.capacity} 人</p>
                <div class="w-full bg-slate-100 rounded-full h-2 mt-2 overflow-hidden">
                  <div class="bg-blue-600 h-2 rounded-full" style="width: {(roomState.currentInside / roomState.capacity) * 100}%;"></div>
                </div>
              </div>
            </div>
          </div>

          <!-- グラフ & 最近の履歴 -->
          <div class="grid grid-cols-1 lg:grid-cols-12 gap-6">
            <!-- 本日の入退室状況グラフ -->
            <div class="lg:col-span-7 bg-white rounded-xl border border-slate-200/90 shadow-sm p-6 flex flex-col justify-between">
              <div>
                <div class="flex items-center justify-between pb-3 border-b border-slate-100">
                  <div>
                    <h3 class="text-base font-bold text-slate-800">本日の入退室状況</h3>
                    <p class="text-xs text-slate-500">時間帯ごとの入室・退室回数（2時間毎）</p>
                  </div>
                  <div class="flex items-center gap-4 text-xs font-semibold">
                    <div class="flex items-center gap-1.5">
                      <span class="w-3 h-3 rounded-xs bg-blue-500"></span>
                      <span class="text-slate-600">青：入室</span>
                    </div>
                    <div class="flex items-center gap-1.5">
                      <span class="w-3 h-3 rounded-xs bg-rose-500"></span>
                      <span class="text-slate-600">赤：退室</span>
                    </div>
                  </div>
                </div>

                <div class="relative mt-6">
                  <svg class="w-full h-64 overflow-visible" viewBox="0 0 520 220">
                    <g class="text-slate-400 text-[10px] font-mono">
                      <line x1="40" y1="20" x2="500" y2="20" stroke="#f1f5f9" stroke-width="1" />
                      <text x="32" y="24" text-anchor="end" fill="#94a3b8">8回</text>
                      <line x1="40" y1="65" x2="500" y2="65" stroke="#f1f5f9" stroke-width="1" />
                      <text x="32" y="69" text-anchor="end" fill="#94a3b8">6回</text>
                      <line x1="40" y1="110" x2="500" y2="110" stroke="#f1f5f9" stroke-width="1" />
                      <text x="32" y="114" text-anchor="end" fill="#94a3b8">4回</text>
                      <line x1="40" y1="155" x2="500" y2="155" stroke="#f1f5f9" stroke-width="1" />
                      <text x="32" y="159" text-anchor="end" fill="#94a3b8">2回</text>
                      <line x1="40" y1="190" x2="500" y2="190" stroke="#cbd5e1" stroke-width="1.5" />
                      <text x="32" y="194" text-anchor="end" fill="#94a3b8">0</text>
                    </g>
                    <!-- 08:00 -->
                    <g>
                      <rect x="65" y="145" width="16" height="45" rx="3" fill="#3b82f6" />
                      <text x="73" y="140" text-anchor="middle" font-size="10" font-weight="bold" fill="#3b82f6">2</text>
                      <text x="83" y="208" text-anchor="middle" font-size="11" fill="#64748b">08:00</text>
                    </g>
                    <!-- 10:00 -->
                    <g>
                      <rect x="130" y="100" width="16" height="90" rx="3" fill="#3b82f6" />
                      <text x="138" y="95" text-anchor="middle" font-size="10" font-weight="bold" fill="#3b82f6">4</text>
                      <rect x="150" y="167.5" width="16" height="22.5" rx="3" fill="#ef4444" />
                      <text x="158" y="162.5" text-anchor="middle" font-size="10" font-weight="bold" fill="#ef4444">1</text>
                      <text x="148" y="208" text-anchor="middle" font-size="11" fill="#64748b">10:00</text>
                    </g>
                    <!-- 12:00 -->
                    <g>
                      <rect x="195" y="167.5" width="16" height="22.5" rx="3" fill="#3b82f6" />
                      <text x="203" y="162.5" text-anchor="middle" font-size="10" font-weight="bold" fill="#3b82f6">1</text>
                      <rect x="215" y="100" width="16" height="90" rx="3" fill="#ef4444" />
                      <text x="223" y="95" text-anchor="middle" font-size="10" font-weight="bold" fill="#ef4444">4</text>
                      <text x="213" y="208" text-anchor="middle" font-size="11" fill="#64748b">12:00</text>
                    </g>
                    <!-- 14:00 -->
                    <g>
                      <rect x="260" y="77.5" width="16" height="112.5" rx="3" fill="#3b82f6" />
                      <text x="268" y="72.5" text-anchor="middle" font-size="10" font-weight="bold" fill="#3b82f6">5</text>
                      <rect x="280" y="100" width="16" height="90" rx="3" fill="#ef4444" />
                      <text x="288" y="95" text-anchor="middle" font-size="10" font-weight="bold" fill="#ef4444">4</text>
                      <text x="278" y="208" text-anchor="middle" font-size="11" fill="#64748b">14:00</text>
                    </g>
                    <!-- 16:00 -->
                    <g>
                      <rect x="345" y="167.5" width="16" height="22.5" rx="3" fill="#ef4444" />
                      <text x="353" y="162.5" text-anchor="middle" font-size="10" font-weight="bold" fill="#ef4444">1</text>
                      <text x="343" y="208" text-anchor="middle" font-size="11" fill="#64748b">16:00</text>
                    </g>
                    <!-- 18:00 -->
                    <g>
                      <text x="408" y="208" text-anchor="middle" font-size="11" fill="#64748b">18:00</text>
                    </g>
                    <!-- 20:00 -->
                    <g>
                      <text x="473" y="208" text-anchor="middle" font-size="11" fill="#64748b">20:00</text>
                    </g>
                  </svg>
                </div>
              </div>
              <p class="text-[11px] text-slate-400 mt-2 text-right">※各棒にカーソルを合わせると詳細が表示されます</p>
            </div>

            <!-- 最近の入退室履歴 (ダッシュボード右側) -->
            <div class="lg:col-span-5 bg-white rounded-xl border border-slate-200/90 shadow-sm p-6 flex flex-col justify-between">
              <div>
                <div class="flex items-center justify-between pb-3 border-b border-slate-100">
                  <h3 class="text-base font-bold text-slate-800">最近の入退室履歴</h3>
                  <button on:click={() => activeTab = 'logs'} class="text-xs text-blue-600 hover:text-blue-800 font-bold flex items-center gap-1">
                    すべて見る
                    <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
                    </svg>
                  </button>
                </div>

                <div class="overflow-x-auto mt-2">
                  <table class="w-full text-left text-xs">
                    <thead>
                      <tr class="text-slate-400 border-b border-slate-100 font-semibold">
                        <th class="py-2.5">時刻</th>
                        <th class="py-2.5">氏名</th>
                        <th class="py-2.5 text-center">入室 / 退室</th>
                        <th class="py-2.5 text-right">認証結果</th>
                      </tr>
                    </thead>
                    <tbody class="divide-y divide-slate-100 font-medium">
                      {#each logsData.slice(0, 5) as log}
                        <tr class="hover:bg-slate-50 transition">
                          <td class="py-2.5 text-slate-600 font-mono">{log.timestamp.split(' ')[1]}</td>
                          <td class="py-2.5 text-slate-800 font-bold">{log.name}</td>
                          <td class="py-2.5 text-center">
                            <span class="inline-block px-2.5 py-0.5 rounded text-[11px] font-bold {log.type === '入室' ? 'bg-blue-100 text-blue-700' : 'bg-rose-100 text-rose-700'}">
                              {log.type}
                            </span>
                          </td>
                          <td class="py-2.5 text-right">
                            <span class="inline-block px-2.5 py-0.5 rounded text-[11px] font-bold {log.result === '成功' ? 'bg-emerald-100 text-emerald-700' : 'bg-rose-100 text-rose-700'}">
                              {log.result}
                            </span>
                          </td>
                        </tr>
                      {/each}
                    </tbody>
                  </table>
                </div>
              </div>

              <div class="pt-3 border-t border-slate-100 text-center">
                <button on:click={() => activeTab = 'logs'} class="text-xs text-slate-500 hover:text-blue-600 font-semibold">
                  すべての入退室履歴を確認する →
                </button>
              </div>
            </div>
          </div>
        </div>
      {/if}

      <!-- 2. 入退室履歴 -->
      {#if activeTab === 'logs'}
        <div class="space-y-6">
          <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <div>
              <h2 class="text-xl font-bold text-slate-800">入退室履歴</h2>
              <p class="text-xs text-slate-500 mt-0.5">管理している研究室への入退室記録を一覧で確認・検索できます</p>
            </div>
            <button 
              on:click={exportLogsCSV}
              class="px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-xl text-xs font-bold transition shadow-xs flex items-center justify-center gap-2"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
              </svg>
              CSV出力
            </button>
          </div>

          <!-- 検索・絞り込みエリア -->
          <div class="bg-white rounded-xl border border-slate-200/90 shadow-sm p-5">
            <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-12 gap-4 items-end">
              <div class="lg:col-span-4">
                <label class="block text-xs font-bold text-slate-600 mb-1.5">日付指定 (開始日 ～ 終了日)</label>
                <div class="flex items-center gap-2">
                  <input type="date" bind:value={filterDateStart} class="w-full text-xs px-3 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20" />
                  <span class="text-slate-400 text-xs">～</span>
                  <input type="date" bind:value={filterDateEnd} class="w-full text-xs px-3 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20" />
                </div>
              </div>

              <div class="lg:col-span-2">
                <label class="block text-xs font-bold text-slate-600 mb-1.5">種類</label>
                <select bind:value={filterType} class="w-full text-xs px-3 py-2 border border-slate-300 rounded-lg bg-white focus:outline-none focus:ring-2 focus:ring-blue-500/20">
                  <option value="all">すべて</option>
                  <option value="入室">入室</option>
                  <option value="退室">退室</option>
                </select>
              </div>

              <div class="lg:col-span-2">
                <label class="block text-xs font-bold text-slate-600 mb-1.5">認証結果</label>
                <select bind:value={filterResult} class="w-full text-xs px-3 py-2 border border-slate-300 rounded-lg bg-white focus:outline-none focus:ring-2 focus:ring-blue-500/20">
                  <option value="all">すべて</option>
                  <option value="成功">成功</option>
                  <option value="失敗">失敗</option>
                </select>
              </div>

              <div class="lg:col-span-2">
                <label class="block text-xs font-bold text-slate-600 mb-1.5">氏名またはID</label>
                <input type="text" bind:value={filterKeyword} placeholder="氏名・IDで検索" class="w-full text-xs px-3 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20" />
              </div>

              <div class="lg:col-span-2 flex items-center gap-2">
                <button 
                  on:click={() => { logCurrentPage = 1; showToast(`${filteredLogs.length}件の履歴が見つかりました`, "info"); }}
                  class="flex-1 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-xs font-bold transition shadow-xs flex items-center justify-center gap-1"
                >
                  検索
                </button>
                <button 
                  on:click={clearLogFilters}
                  class="px-3 py-2 border border-slate-300 bg-white hover:bg-slate-50 text-slate-700 rounded-lg text-xs font-semibold transition"
                >
                  クリア
                </button>
              </div>
            </div>
          </div>

          <!-- テーブル -->
          <div class="bg-white rounded-xl border border-slate-200/90 shadow-sm overflow-hidden">
            <div class="px-6 py-4 border-b border-slate-100 flex items-center justify-between bg-slate-50/50">
              <span class="text-xs font-bold text-slate-600">
                該当件数：<span class="font-extrabold text-slate-900">{filteredLogs.length}</span> 件
              </span>
              <span class="text-xs text-slate-400">最新順</span>
            </div>

            <div class="overflow-x-auto">
              <table class="w-full text-left text-xs">
                <thead class="bg-slate-50 text-slate-500 font-bold uppercase tracking-wider border-b border-slate-200">
                  <tr>
                    <th class="px-6 py-3.5">日時</th>
                    <th class="px-6 py-3.5">氏名</th>
                    <th class="px-6 py-3.5">ユーザーID</th>
                    <th class="px-6 py-3.5 text-center">種類</th>
                    <th class="px-6 py-3.5 text-center">認証結果</th>
                    <th class="px-6 py-3.5">備考</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-slate-100">
                  {#if paginatedLogs.length === 0}
                    <tr>
                      <td colspan="6" class="px-6 py-10 text-center text-slate-400 font-medium">条件に一致する履歴はありません。</td>
                    </tr>
                  {:else}
                    {#each paginatedLogs as log}
                      <tr class="hover:bg-slate-50/80 transition border-b border-slate-100">
                        <td class="px-6 py-3.5 font-mono text-slate-600">{log.timestamp}</td>
                        <td class="px-6 py-3.5 font-bold text-slate-800">{log.name}</td>
                        <td class="px-6 py-3.5 font-mono text-slate-500">{log.uid}</td>
                        <td class="px-6 py-3.5 text-center">
                          <span class="inline-block px-2.5 py-0.5 rounded text-xs font-bold {log.type === '入室' ? 'bg-blue-100 text-blue-700' : 'bg-rose-100 text-rose-700'}">{log.type}</span>
                        </td>
                        <td class="px-6 py-3.5 text-center">
                          <span class="inline-block px-2.5 py-0.5 rounded text-xs font-bold {log.result === '成功' ? 'bg-emerald-100 text-emerald-700' : 'bg-rose-100 text-rose-700'}">{log.result}</span>
                        </td>
                        <td class="px-6 py-3.5 text-slate-500">{log.note}</td>
                      </tr>
                    {/each}
                  {/if}
                </tbody>
              </table>
            </div>

            <!-- ページネーション -->
            <div class="px-6 py-4 border-t border-slate-100 flex items-center justify-between">
              <div class="text-xs text-slate-500">
                全 {filteredLogs.length} 件中 {filteredLogs.length > 0 ? (logCurrentPage - 1) * LOGS_PER_PAGE + 1 : 0} ～ {Math.min(logCurrentPage * LOGS_PER_PAGE, filteredLogs.length)} 件を表示
              </div>
              {#if totalLogPages > 1}
                <div class="flex items-center gap-1.5 text-xs">
                  <button 
                    disabled={logCurrentPage === 1}
                    on:click={() => logCurrentPage--}
                    class="px-2.5 py-1 rounded border border-slate-300 hover:bg-slate-50 disabled:text-slate-300 disabled:cursor-not-allowed"
                  >前へ</button>

                  {#each Array(totalLogPages) as _, i}
                    <button 
                      on:click={() => logCurrentPage = i + 1}
                      class="px-3 py-1 rounded font-bold {logCurrentPage === i + 1 ? 'bg-blue-600 text-white' : 'border border-slate-200 hover:bg-slate-50 text-slate-700'}"
                    >
                      {i + 1}
                    </button>
                  {/each}

                  <button 
                    disabled={logCurrentPage === totalLogPages}
                    on:click={() => logCurrentPage++}
                    class="px-2.5 py-1 rounded border border-slate-300 hover:bg-slate-50 disabled:text-slate-300 disabled:cursor-not-allowed"
                  >次へ</button>
                </div>
              {/if}
            </div>
          </div>
        </div>
      {/if}

      <!-- 3. ユーザー管理 -->
      {#if activeTab === 'users'}
        <div class="space-y-6">
          <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <div>
              <h2 class="text-xl font-bold text-slate-800">ユーザー管理</h2>
              <p class="text-xs text-slate-500 mt-0.5">研究室の入室を許可するユーザーを管理します</p>
            </div>
            <button 
              on:click={openCreateUserModal}
              class="px-5 py-2.5 bg-blue-600 hover:bg-blue-700 text-white rounded-xl text-xs font-bold transition shadow-sm flex items-center justify-center gap-1.5"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"/>
              </svg>
              ＋ 新規登録
            </button>
          </div>

          <!-- 検索 & 絞り込み -->
          <div class="bg-white rounded-xl border border-slate-200/90 shadow-sm p-5 flex flex-col md:flex-row md:items-center justify-between gap-4">
            <div class="relative w-full md:w-80">
              <span class="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none text-slate-400">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
                </svg>
              </span>
              <input 
                type="text" 
                bind:value={userSearch}
                placeholder="氏名またはユーザーIDで検索..." 
                class="w-full text-xs pl-9 pr-3.5 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20"
              />
            </div>

            <div class="flex items-center gap-2">
              <span class="text-xs font-bold text-slate-500 mr-1">状態絞り込み:</span>
              <div class="inline-flex rounded-lg bg-slate-100 p-0.5 border border-slate-200 text-xs font-medium">
                <button 
                  on:click={() => userStatusFilter = 'all'} 
                  class="px-3.5 py-1.5 rounded-md font-bold transition {userStatusFilter === 'all' ? 'bg-white text-blue-700 shadow-xs' : 'text-slate-600 hover:text-slate-900'}"
                >すべて</button>
                <button 
                  on:click={() => userStatusFilter = 'active'} 
                  class="px-3.5 py-1.5 rounded-md font-bold transition {userStatusFilter === 'active' ? 'bg-white text-emerald-700 shadow-xs' : 'text-slate-600 hover:text-slate-900'}"
                >有効</button>
                <button 
                  on:click={() => userStatusFilter = 'inactive'} 
                  class="px-3.5 py-1.5 rounded-md font-bold transition {userStatusFilter === 'inactive' ? 'bg-white text-rose-700 shadow-xs' : 'text-slate-600 hover:text-slate-900'}"
                >無効</button>
              </div>
            </div>
          </div>

          <!-- テーブル -->
          <div class="bg-white rounded-xl border border-slate-200/90 shadow-sm overflow-hidden">
            <div class="px-6 py-4 border-b border-slate-100 flex items-center justify-between bg-slate-50/50">
              <span class="text-xs font-bold text-slate-600">
                登録ユーザー：<span class="font-extrabold text-slate-900">{filteredUsers.length}</span> 名
              </span>
            </div>

            <div class="overflow-x-auto">
              <table class="w-full text-left text-xs">
                <thead class="bg-slate-50 text-slate-500 font-bold uppercase tracking-wider border-b border-slate-200">
                  <tr>
                    <th class="px-6 py-3.5">氏名</th>
                    <th class="px-6 py-3.5">ユーザーID</th>
                    <th class="px-6 py-3.5">権限</th>
                    <th class="px-6 py-3.5 text-center">状態</th>
                    <th class="px-6 py-3.5 text-right">操作</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-slate-100">
                  {#if filteredUsers.length === 0}
                    <tr>
                      <td colspan="5" class="px-6 py-10 text-center text-slate-400 font-medium">該当するユーザーが見つかりませんでした。</td>
                    </tr>
                  {:else}
                    {#each filteredUsers as user}
                      <tr class="hover:bg-slate-50/80 transition border-b border-slate-100">
                        <td class="px-6 py-3.5 font-bold text-slate-800">{user.name}</td>
                        <td class="px-6 py-3.5 font-mono text-slate-600">{user.uid}</td>
                        <td class="px-6 py-3.5">
                          <span class="inline-block px-2.5 py-0.5 rounded text-xs {user.role === '管理者' ? 'font-bold bg-indigo-50 text-indigo-700 border border-indigo-200' : 'font-medium bg-slate-100 text-slate-700'}">
                            {user.role}
                          </span>
                        </td>
                        <td class="px-6 py-3.5 text-center">
                          <span class="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded text-xs font-bold {user.status === '有効' ? 'bg-emerald-50 text-emerald-700 border border-emerald-200' : 'bg-slate-100 text-slate-500'}">
                            <span class="w-1.5 h-1.5 rounded-full {user.status === '有効' ? 'bg-emerald-500' : 'bg-slate-400'}"></span>
                            {user.status}
                          </span>
                        </td>
                        <td class="px-6 py-3.5 text-right space-x-2">
                          <button 
                            on:click={() => openEditUserModal(user)}
                            class="px-2.5 py-1 text-xs font-bold text-blue-600 hover:text-blue-800 hover:bg-blue-50 rounded transition"
                          >編集</button>
                          <button 
                            on:click={() => handleDeleteUser(user)}
                            class="px-2.5 py-1 text-xs font-bold text-rose-600 hover:text-rose-800 hover:bg-rose-50 rounded transition"
                          >削除</button>
                        </td>
                      </tr>
                    {/each}
                  {/if}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      {/if}

    </main>
  </div>

  <!-- モーダルウィンドウ -->
  {#if isUserModalOpen}
    <div class="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/50 backdrop-blur-xs">
      <div class="bg-white rounded-2xl shadow-2xl border border-slate-200 w-full max-w-md mx-4 overflow-hidden">
        <div class="px-6 py-4 border-b border-slate-100 flex items-center justify-between bg-slate-50">
          <h3 class="text-base font-bold text-slate-800">
            {modalMode === 'create' ? '新規ユーザー登録' : 'ユーザー情報の編集'}
          </h3>
          <button on:click={closeUserModal} class="text-slate-400 hover:text-slate-600 p-1 rounded-lg">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>

        <form on:submit|preventDefault={handleSaveUser} class="p-6 space-y-4">
          <div>
            <label class="block text-xs font-bold text-slate-700 mb-1">氏名 <span class="text-rose-500">*</span></label>
            <input type="text" bind:value={formName} required placeholder="例: 山田 太郎" class="w-full text-xs px-3.5 py-2.5 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20" />
          </div>

          <div>
            <label class="block text-xs font-bold text-slate-700 mb-1">ユーザーID <span class="text-rose-500">*</span></label>
            <input type="text" bind:value={formUid} required placeholder="例: A0012345" class="w-full text-xs px-3.5 py-2.5 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 font-mono uppercase" />
          </div>

          <div>
            <label class="block text-xs font-bold text-slate-700 mb-1.5">権限</label>
            <div class="grid grid-cols-2 gap-3">
              <label class="flex items-center gap-2 p-2.5 border border-slate-200 rounded-lg cursor-pointer hover:bg-slate-50">
                <input type="radio" bind:group={formRole} value="一般ユーザー" class="text-blue-600">
                <span class="text-xs font-semibold text-slate-700">一般ユーザー</span>
              </label>
              <label class="flex items-center gap-2 p-2.5 border border-slate-200 rounded-lg cursor-pointer hover:bg-slate-50">
                <input type="radio" bind:group={formRole} value="管理者" class="text-blue-600">
                <span class="text-xs font-semibold text-slate-700">管理者</span>
              </label>
            </div>
          </div>

          <div>
            <label class="block text-xs font-bold text-slate-700 mb-1.5">状態</label>
            <div class="grid grid-cols-2 gap-3">
              <label class="flex items-center gap-2 p-2.5 border border-slate-200 rounded-lg cursor-pointer hover:bg-slate-50">
                <input type="radio" bind:group={formStatus} value="有効" class="text-emerald-600">
                <span class="text-xs font-bold text-emerald-700">有効</span>
              </label>
              <label class="flex items-center gap-2 p-2.5 border border-slate-200 rounded-lg cursor-pointer hover:bg-slate-50">
                <input type="radio" bind:group={formStatus} value="無効" class="text-rose-600">
                <span class="text-xs font-semibold text-slate-500">無効</span>
              </label>
            </div>
          </div>

          <div class="pt-4 border-t border-slate-100 flex items-center justify-end gap-3">
            <button type="button" on:click={closeUserModal} class="px-4 py-2 border border-slate-300 text-slate-700 rounded-lg text-xs font-semibold hover:bg-slate-50">
              キャンセル
            </button>
            <button type="submit" class="px-5 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-xs font-bold shadow-sm">
              {modalMode === 'create' ? '登録' : '保存'}
            </button>
          </div>
        </form>
      </div>
    </div>
  {/if}

  <!-- トースト通知 -->
  <div class="fixed top-5 right-5 z-50 flex flex-col gap-2 pointer-events-none">
    {#each toasts as t (t.id)}
      <div class="flex items-center gap-2.5 px-4 py-3 rounded-xl border text-xs font-semibold shadow-md pointer-events-auto {t.type === 'success' ? 'bg-slate-900 text-white border-emerald-500/60' : t.type === 'error' ? 'bg-rose-950 text-white border-rose-500/60' : 'bg-slate-800 text-white border-slate-700'}">
        {#if t.type === 'success'}
          <svg class="w-4 h-4 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>
        {:else if t.type === 'error'}
          <svg class="w-4 h-4 text-rose-300" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/></svg>
        {:else}
          <svg class="w-4 h-4 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>
        {/if}
        <span>{t.message}</span>
      </div>
    {/each}
  </div>

</div>

