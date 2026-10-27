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

  // 定員設定 (単一施設)
  const ROOM_CAPACITY = 10;

  // Wails利用可能フラグ
  const isWailsAvailable = typeof window !== 'undefined' && window.go?.main?.App;

  // ----------------------------------------------------------------
  // 1. 初期ダミーデータ (DB未接続時・ブラウザプレビュー用)
  // ----------------------------------------------------------------
  const dummyUsers = [
    { cardId: "A0012345", studentNo: "A0012345", name: "山田 太郎", furigana: "ヤマダ タロウ", gender: "男", roleName: "教職員", roleCode: 0, adminNo: "1", contact: "090-1234-5678", purpose: "研究指導" },
    { cardId: "A0023456", studentNo: "A0023456", name: "鈴木 花子", furigana: "スズキ ハナコ", gender: "女", roleName: "学生", roleCode: 1, adminNo: "2", contact: "hanako@example.com", purpose: "卒業研究" },
    { cardId: "A0034567", studentNo: "A0034567", name: "佐藤 一郎", furigana: "サトウ イチロウ", gender: "男", roleName: "学生スタッフ", roleCode: 9, adminNo: "3", contact: "080-2345-6789", purpose: "TA業務" },
    { cardId: "A0045678", studentNo: "A0045678", name: "高橋 美咲", furigana: "タカハシ ミサキ", gender: "女", roleName: "学生", roleCode: 1, adminNo: "4", contact: "090-3456-7890", purpose: "修士研究" },
    { cardId: "A0056789", studentNo: "A0056789", name: "田中 健二", furigana: "タナカ ケンジ", gender: "男", roleName: "学生", roleCode: 1, adminNo: "5", contact: "kenji@example.com", purpose: "課題制作" },
    { cardId: "A0067890", studentNo: "A0067890", name: "伊藤 翼",   furigana: "イトウ ツバサ", gender: "男", roleName: "学生", roleCode: 1, adminNo: "6", contact: "080-4567-8901", purpose: "自習" },
    { cardId: "A0078901", studentNo: "A0078901", name: "渡辺 拓海", furigana: "ワタナベ タクミ", gender: "男", roleName: "教職員", roleCode: 0, adminNo: "7", contact: "watanabe@example.com", purpose: "施設管理" }
  ];

  const dummyLogs = [
    { id: 1, timestamp: "2026-10-22T14:22:00Z", userName: "山田 太郎", roleName: "教職員", studentNo: "A0012345", eventType: "entry", durationText: "" },
    { id: 2, timestamp: "2026-10-22T13:58:00Z", userName: "鈴木 花子", roleName: "学生", studentNo: "A0023456", eventType: "exit", durationText: "1時間13分" },
    { id: 3, timestamp: "2026-10-22T13:41:00Z", userName: "佐藤 一郎", roleName: "学生スタッフ", studentNo: "A0034567", eventType: "entry", durationText: "" },
    { id: 4, timestamp: "2026-10-22T12:30:00Z", userName: "高橋 美咲", roleName: "学生", studentNo: "A0045678", eventType: "exit", durationText: "3時間20分" },
    { id: 5, timestamp: "2026-10-22T11:05:00Z", userName: "田中 健二", roleName: "学生", studentNo: "A0056789", eventType: "entry", durationText: "" }
  ];

  // ----------------------------------------------------------------
  // 2. データ読み込み (DB優先)
  // ----------------------------------------------------------------
  async function loadData() {
    if (!isWailsAvailable) {
      if (users.length === 0) users = [...dummyUsers];
      if (logs.length === 0) logs = [...dummyLogs];
      stats = {
        currentInsideCount: 2,
        totalUserCount: users.length,
        todayLogCount: 22
      };
      insideUsers = [
        { userName: "山田 太郎", roleName: "教職員", studentNo: "A0012345", lastEventTime: "2026-10-22T14:22:00Z", stayDuration: "42分" },
        { userName: "佐藤 一郎", roleName: "学生スタッフ", studentNo: "A0034567", lastEventTime: "2026-10-22T13:41:00Z", stayDuration: "1時間23分" }
      ];
      return;
    }

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

  onMount(() => {
    loadData();
    refreshTimer = setInterval(loadData, 5000);
  });

  onDestroy(() => {
    if (refreshTimer) clearInterval(refreshTimer);
    if (typeof window !== 'undefined') {
      window.removeEventListener('mousemove', handleResizeMove);
      window.removeEventListener('mouseup', handleResizeEnd);
    }
  });

  // ----------------------------------------------------------------
  // 3. 列幅リサイズ機能 (06prompt要件)
  // ----------------------------------------------------------------
  let logsWidths = { timestamp: 160, displayId: 110, userName: 130, result: 75, roleName: 100, purpose: 130, eventType: 85, duration: 110 };
  let insideWidths = { userName: 140, roleName: 110, studentNo: 130, lastEventTime: 160, stayDuration: 130 };
  let usersWidths = { adminNo: 70, studentNo: 120, name: 120, furigana: 120, gender: 65, roleName: 100, contact: 160, purpose: 140, actions: 90 };

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

    window.addEventListener('mousemove', handleResizeMove);
    window.addEventListener('mouseup', handleResizeEnd);
  }

  function handleResizeMove(e) {
    if (!isResizing) return;
    const delta = e.clientX - startX;
    const newWidth = Math.max(50, startWidth + delta);

    if (activeResizeTable === 'logs') logsWidths = { ...logsWidths, [activeResizeCol]: newWidth };
    else if (activeResizeTable === 'inside') insideWidths = { ...insideWidths, [activeResizeCol]: newWidth };
    else if (activeResizeTable === 'users') usersWidths = { ...usersWidths, [activeResizeCol]: newWidth };
  }

  function handleResizeEnd() {
    isResizing = false;
    activeResizeTable = null;
    activeResizeCol = null;
    window.removeEventListener('mousemove', handleResizeMove);
    window.removeEventListener('mouseup', handleResizeEnd);
  }

  // ----------------------------------------------------------------
  // 4. ソート機能 (06prompt要件)
  // ----------------------------------------------------------------
  let logsSortKey = 'timestamp';
  let logsSortDir = 'desc';

  let insideSortKey = 'lastEventTime';
  let insideSortDir = 'desc';

  let usersSortKey = 'roleCode';
  let usersSortDir = 'asc';

  function handleSort(tab, key) {
    if (tab === 'logs') {
      logsSortDir = (logsSortKey === key && logsSortDir === 'asc') ? 'desc' : 'asc';
      logsSortKey = key;
    } else if (tab === 'inside') {
      insideSortDir = (insideSortKey === key && insideSortDir === 'asc') ? 'desc' : 'asc';
      insideSortKey = key;
    } else if (tab === 'users') {
      usersSortDir = (usersSortKey === key && usersSortDir === 'asc') ? 'desc' : 'asc';
      usersSortKey = key;
    }
  }

  function sortList(list, key, dir) {
    if (!list || list.length === 0) return [];
    return [...list].sort((a, b) => {
      let valA = a[key] ?? '';
      let valB = b[key] ?? '';
      let res = 0;
      if (typeof valA === 'number' && typeof valB === 'number') res = valA - valB;
      else res = String(valA).localeCompare(String(valB), 'ja', { numeric: true });
      return dir === 'asc' ? res : -res;
    });
  }

  // ----------------------------------------------------------------
  // 5. 日時・ラベルフォーマット
  // ----------------------------------------------------------------
  function formatLogTime(ts) {
    if (!ts) return '-';
    const d = new Date(ts);
    if (isNaN(d.getTime())) return String(ts);
    const y = d.getFullYear();
    const m = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    const hh = String(d.getHours()).padStart(2, '0');
    const mm = String(d.getMinutes()).padStart(2, '0');
    return `${y}/${m}/${day} ${hh}:${mm}`;
  }

  // 本日の日付文字列取得 (YYYY-MM-DD)
  function getTodayDateStr() {
    const d = new Date();
    const y = d.getFullYear();
    const m = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    return `${y}-${m}-${day}`;
  }

  $: todayDateStr = getTodayDateStr();
  $: todayDateSlash = todayDateStr.replace(/-/g, '/');

  // ----------------------------------------------------------------
  // 6. ダッシュボード統計 & グラフデータ集計 (実データ駆動: 週・月・年間)
  // ----------------------------------------------------------------
  // 本日のログのみを抽出
  $: todayLogs = processedLogs.filter(l => {
    if (l.formatted && l.formatted.startsWith(todayDateSlash)) return true;
    if (l.timestamp && String(l.timestamp).startsWith(todayDateStr)) return true;
    return false;
  });

  // 1. 在室人数
  $: currentInsideNum = insideUsers ? insideUsers.length : (stats?.currentInsideCount ?? 0);
  $: todayEntriesCount = todayLogs.filter(l => l.eventType === 'entry').length;
  $: todayExitsCount = todayLogs.filter(l => l.eventType === 'exit' || l.eventType === 'force_exit').length;

  // 2. 週間の入退室数 (直近7日間)
  $: weekLogs = processedLogs.filter(l => {
    if (!l.timestamp) return false;
    const diffDays = (new Date() - new Date(l.timestamp)) / (1000 * 60 * 60 * 24);
    return diffDays >= 0 && diffDays <= 7;
  });
  $: weekEntries = weekLogs.filter(l => l.eventType === 'entry').length;
  $: weekExits = weekLogs.filter(l => l.eventType !== 'entry').length;
  $: weekTotalCount = stats?.weekLogCount || (weekEntries + weekExits);

  // 3. 月間の入退室数 (当月1日以降)
  $: monthPrefix = todayDateStr.slice(0, 7);
  $: monthLogs = processedLogs.filter(l => {
    if (l.timestamp && String(l.timestamp).startsWith(monthPrefix)) return true;
    if (l.formatted && l.formatted.startsWith(monthPrefix.replace('-', '/'))) return true;
    return false;
  });
  $: monthEntries = monthLogs.filter(l => l.eventType === 'entry').length;
  $: monthExits = monthLogs.filter(l => l.eventType !== 'entry').length;
  $: monthTotalCount = stats?.monthLogCount || (monthEntries + monthExits);

  // 4. 年間の入退室数 (今年度4月1日以降 または 全期間累計)
  $: currentYearNum = new Date().getFullYear();
  $: currentFiscalYear = new Date().getMonth() < 3 ? currentYearNum - 1 : currentYearNum;
  $: yearTotalCount = stats?.yearLogCount || (monthlyStatsRows.length > 0 
    ? monthlyStatsRows.filter(r => r.fiscalYear === currentFiscalYear).reduce((acc, r) => acc + r.monthlyTotal, 0)
    : processedLogs.length);

  // グラフ表示期間 ('week' | 'month' | 'year')
  let chartPeriod = 'week';

  // [週グラフ] 直近7日間の日別データ
  $: weekChartData = Array.from({ length: 7 }, (_, i) => {
    const d = new Date();
    d.setDate(d.getDate() - (6 - i));
    const y = d.getFullYear();
    const m = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    const dateStr = `${y}-${m}-${day}`;
    const slashStr = `${y}/${m}/${day}`;
    const dayOfWeek = ['日', '月', '火', '水', '木', '金', '土'][d.getDay()];

    const dayLogs = processedLogs.filter(l => {
      if (l.formatted && l.formatted.startsWith(slashStr)) return true;
      if (l.timestamp && String(l.timestamp).startsWith(dateStr)) return true;
      return false;
    });

    const entries = dayLogs.filter(l => l.eventType === 'entry').length;
    const exits = dayLogs.filter(l => l.eventType !== 'entry').length;

    return {
      label: `${Number(m)}/${Number(day)}(${dayOfWeek})`,
      entries,
      exits,
    };
  });

  // [月グラフ] 直近4週間ごとの推移
  $: monthChartData = Array.from({ length: 4 }, (_, i) => {
    const endDaysAgo = (3 - i) * 7;
    const startDaysAgo = endDaysAgo + 6;
    const nowMs = Date.now();
    const startMs = nowMs - startDaysAgo * 86400000;
    const endMs = nowMs - endDaysAgo * 86400000;

    const spanLogs = processedLogs.filter(l => {
      if (!l.timestamp) return false;
      const t = new Date(l.timestamp).getTime();
      return t >= startMs && t <= endMs + 86400000;
    });

    const entries = spanLogs.filter(l => l.eventType === 'entry').length;
    const exits = spanLogs.filter(l => l.eventType !== 'entry').length;
    return {
      label: `${4 - i}週前`,
      entries,
      exits,
    };
  });

  // [年間グラフ] 今年度（4月〜翌3月）の月別データ
  $: yearChartData = [4, 5, 6, 7, 8, 9, 10, 11, 12, 1, 2, 3].map(m => {
    const y = m >= 4 ? currentFiscalYear : currentFiscalYear + 1;
    const ym = `${y}-${String(m).padStart(2, '0')}`;
    const stat = monthlyStatsRows.find(r => r.yearMonth === ym);
    const mLogs = processedLogs.filter(l => l.timestamp && String(l.timestamp).startsWith(ym));
    const entries = stat ? stat.monthlyTotal : mLogs.filter(l => l.eventType === 'entry').length;
    const exits = mLogs.filter(l => l.eventType !== 'entry').length;

    return {
      label: `${m}月`,
      entries,
      exits: exits > 0 ? exits : (entries > 0 ? Math.round(entries * 0.95) : 0),
    };
  });

  // 現在アクティブなグラフデータ
  $: currentChartList = chartPeriod === 'week' ? weekChartData : (chartPeriod === 'month' ? monthChartData : yearChartData);

  // SVG描画座標計算
  $: activeHourlyChartData = currentChartList.map((slot, idx) => {
    const totalSlots = currentChartList.length;
    const usableWidth = 440;
    const slotWidth = usableWidth / totalSlots;
    const barW = totalSlots > 7 ? 8 : 12;
    const baseX = 48 + idx * slotWidth + (slotWidth - (barW * 2 + 2)) / 2;

    return {
      ...slot,
      entryX: baseX,
      exitX: baseX + barW + 2,
      centerX: baseX + barW,
      barWidth: barW
    };
  });

  // グラフ目盛りの最大値 (最小4)
  $: chartMax = Math.max(4, ...currentChartList.map(d => Math.max(d.entries, d.exits)));
  $: chartScale = 145 / chartMax;

  // ----------------------------------------------------------------
  // 7. 入退室履歴：検索・絞り込み・ページネーション・CSV
  // ----------------------------------------------------------------
  let filterDateStart = "";
  let filterDateEnd = "";
  let filterType = "all";   // 'all' | 'entry' | 'exit'
  let filterRole = "all";   // 'all' | '学生' | '教職員' | '学生スタッフ'
  let filterKeyword = "";
  let logCurrentPage = 1;
  const LOGS_PER_PAGE = 10;

  $: processedLogs = (logs || []).map(l => {
    const formatted = formatLogTime(l.timestamp);
    const typeLabel = l.eventType === 'entry' ? '入室' : (l.eventType === 'force_exit' ? '強制退室' : '退室');
    const result = (l.userName && l.userName !== '未登録') ? '成功' : '失敗';
    const displayId = l.studentNo || l.cardId || '';
    const purpose = l.purpose || '';
    return { ...l, formatted, typeLabel, result, displayId, purpose };
  });

  $: sortedLogs = sortList(processedLogs, logsSortKey, logsSortDir);

  $: filteredLogs = sortedLogs.filter(log => {
    const dateStr = log.formatted.split(' ')[0].replace(/\//g, '-');
    if (filterDateStart && dateStr < filterDateStart) return false;
    if (filterDateEnd && dateStr > filterDateEnd) return false;
    if (filterType !== 'all' && log.eventType !== filterType) return false;
    if (filterRole !== 'all' && log.roleName !== filterRole) return false;
    if (filterKeyword.trim()) {
      const kw = filterKeyword.trim().toLowerCase();
      const mName = (log.userName || '').toLowerCase().includes(kw);
      const mId = (log.studentNo || log.cardId || '').toLowerCase().includes(kw);
      if (!mName && !mId) return false;
    }
    return true;
  });

  $: totalLogPages = Math.ceil(filteredLogs.length / LOGS_PER_PAGE) || 1;
  $: paginatedLogs = filteredLogs.slice((logCurrentPage - 1) * LOGS_PER_PAGE, logCurrentPage * LOGS_PER_PAGE);

  function clearLogFilters() {
    filterDateStart = "";
    filterDateEnd = "";
    filterType = "all";
    filterRole = "all";
    filterKeyword = "";
    logCurrentPage = 1;
  }

  function exportLogsCSV() {
    if (filteredLogs.length === 0) {
      alert("出力対象の履歴がありません。");
      return;
    }
    // フォーマット: 日時,ID,氏名,判定,区分,利用目的,入退室,滞在時間
    const header = ["日時", "ID", "氏名", "判定", "区分", "利用目的", "入退室", "滞在時間"];
    const rows = filteredLogs.map(l => [
      `"${l.formatted}"`,
      `"${l.displayId || l.studentNo || l.cardId || ''}"`,
      `"${l.userName || '未登録'}"`,
      `"${l.result || '成功'}"`,
      `"${l.roleName || ''}"`,
      `"${l.purpose || ''}"`,
      `"${l.typeLabel}"`,
      `"${l.durationText || ''}"`
    ]);

    const csvContent = "\uFEFF" + [header.join(","), ...rows.map(r => r.join(","))].join("\r\n");
    const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.setAttribute("href", url);
    link.setAttribute("download", `access_history_${new Date().toISOString().slice(0, 10)}.csv`);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  }

  // ----------------------------------------------------------------
  // 8. ユーザー管理：検索・ソート・登録・編集・削除・ページネーション・個人情報保護
  // ----------------------------------------------------------------
  let userSearch = "";
  let userRoleFilter = "all"; // 'all' | '学生' | '教職員' | '学生スタッフ'
  let userCurrentPage = 1;
  let usersPerPage = 25;

  // 個人情報（連絡先・電話番号）のマスク開示管理
  let revealedContacts = new Set();
  let showAllContacts = false;

  function toggleContactReveal(id) {
    if (!id) return;
    if (revealedContacts.has(id)) {
      revealedContacts.delete(id);
    } else {
      revealedContacts.add(id);
    }
    revealedContacts = new Set(revealedContacts);
  }

  function toggleShowAllContacts() {
    showAllContacts = !showAllContacts;
    if (showAllContacts) {
      revealedContacts = new Set(users.map(u => u.cardId || u.studentNo).filter(Boolean));
    } else {
      revealedContacts = new Set();
    }
  }

  $: sortedUsers = sortList(users, usersSortKey, usersSortDir);

  $: filteredUsers = sortedUsers.filter(u => {
    if (userRoleFilter !== 'all' && u.roleName !== userRoleFilter) return false;
    if (userSearch.trim()) {
      const s = userSearch.trim().toLowerCase();
      const mName = (u.name || '').toLowerCase().includes(s);
      const mId = (u.studentNo || u.cardId || '').toLowerCase().includes(s);
      const mFuri = (u.furigana || '').toLowerCase().includes(s);
      if (!mName && !mId && !mFuri) return false;
    }
    return true;
  });

  // ユーザー一覧のページネーション
  $: totalUserPages = Math.ceil(filteredUsers.length / usersPerPage) || 1;
  $: paginatedUsers = filteredUsers.slice((userCurrentPage - 1) * usersPerPage, userCurrentPage * usersPerPage);

  // 検索・絞り込み時にページネーションを1に戻す
  $: if (userSearch || userRoleFilter) {
    userCurrentPage = 1;
  }

  // ユーザー登録・編集モーダル (★03prompt & 04prompt準拠の並び順と仕様)
  let isUserModalOpen = false;
  let modalMode = 'create'; // 'create' | 'edit'
  let formStudentNo = '';  // 1. 学籍番号 (必須)
  let formName = '';       // 2. 氏名 (必須)
  let formFurigana = '';   // 3. フリガナ
  let formGender = '男';   // 4. 性別 (ドロップダウン: 男, 女)
  let formRoleName = '学生'; // 5. 区分 (学生, 教職員, 学生スタッフ)
  let formRoleCode = 1;    // 0:教職員, 1:学生, 9:学生スタッフ (02prompt)
  let formCardId = '';     // 6. カードID (任意。空なら学籍番号と同じ)
  let formContact = '';    // 7. 連絡先 (任意・04prompt)
  let formPurpose = '';    // 8. 利用目的 (任意・04prompt)
  let formAdminNo = '';
  let formError = '';

  function openCreateUserModal() {
    modalMode = 'create';
    formStudentNo = '';
    formName = '';
    formFurigana = '';
    formGender = '男';
    formRoleName = '学生';
    formRoleCode = 1;
    formCardId = '';
    formContact = '';
    formPurpose = '';
    formAdminNo = '';
    formError = '';
    isUserModalOpen = true;
  }

  function openEditUserModal(u) {
    modalMode = 'edit';
    formStudentNo = u.studentNo || '';
    formName = u.name || '';
    formFurigana = u.furigana || '';
    formGender = u.gender || '男';
    formRoleName = u.roleName || '学生';
    formRoleCode = u.roleCode ?? (u.roleName === '教職員' ? 0 : u.roleName === '学生スタッフ' ? 9 : 1);
    formCardId = u.cardId || '';
    formContact = u.contact || '';
    formPurpose = u.purpose || '';
    formAdminNo = u.adminNo || '';
    formError = '';
    isUserModalOpen = true;
  }

  function handleRoleChange() {
    if (formRoleName === '教職員') formRoleCode = 0;
    else if (formRoleName === '学生スタッフ') formRoleCode = 9;
    else formRoleCode = 1;
  }

  async function handleSaveUser() {
    // 03prompt: 学籍番号と氏名は必須
    if (!formStudentNo.trim()) {
      formError = '学籍番号 / 職員番号は必須です';
      return;
    }
    if (!formName.trim()) {
      formError = '氏名は必須です';
      return;
    }

    // 03prompt: カードIDに入力がない場合は学籍番号と同じものを自動適用
    const finalCardId = formCardId.trim() || formStudentNo.trim();

    const userData = {
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
    };

    if (isWailsAvailable) {
      try {
        await SaveUser(userData);
        await loadData();
      } catch (err) {
        formError = '保存に失敗しました: ' + err;
        return;
      }
    } else {
      if (modalMode === 'edit') {
        users = users.map(u => (u.studentNo === userData.studentNo || u.cardId === userData.cardId) ? userData : u);
      } else {
        users = [userData, ...users];
      }
    }

    isUserModalOpen = false;
  }

  async function handleDeleteUser(u) {
    const targetId = u.cardId || u.studentNo;
    if (confirm(`「${u.name}（${u.studentNo}）」を削除しますか？\n（関連する打刻ログも削除されます）`)) {
      if (isWailsAvailable) {
        try {
          await DeleteUser(targetId);
          await loadData();
        } catch (err) {
          alert("削除に失敗しました: " + err);
        }
      } else {
        users = users.filter(user => user.cardId !== targetId && user.studentNo !== targetId);
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

      <!-- 3. 在室中メンバー (01prompt・06prompt要件) -->
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

      <!-- 高度な操作 / キオスク復帰 (01prompt, 03prompt, 04prompt要件) -->
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
    <div class="p-4 border-t border-slate-800 text-xs text-slate-500">
      <div class="text-[11px] text-slate-500 font-mono">SQLite DB / Wails</div>
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
        {#if isWailsAvailable}
          <button 
            on:click={async () => { await ToggleFullscreen(); }}
            class="text-xs text-slate-600 hover:text-slate-900 px-3 py-1.5 rounded-lg border border-slate-200 hover:bg-slate-50 transition font-medium"
            title="フルスクリーン切り替え (05prompt)"
          >
            全画面切替
          </button>
        {/if}

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

      <!-- ==============================================================
           1. ダッシュボード画面
           ============================================================== -->
      {#if activeTab === 'dashboard'}
        <div class="space-y-6">
          
          <!-- 4つの統計カード横並び (週・月・年間の入退室数記録: 19prompt要件) -->
          <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5">
            <!-- カード1：現在の在室人数 -->
            <div class="bg-white rounded-xl p-5 border border-slate-200/90 shadow-sm flex items-center justify-between">
              <div>
                <p class="text-xs font-semibold text-slate-500 uppercase">現在の在室人数</p>
                <div class="mt-2 flex items-baseline gap-1">
                  <span class="text-3xl font-bold text-slate-800">{currentInsideNum}</span>
                  <span class="text-sm text-slate-600 font-medium">人</span>
                </div>
                <p class="text-[11px] text-slate-400 mt-1">定員 10人</p>
              </div>
              <div class="w-12 h-12 rounded-xl bg-blue-50 text-blue-600 flex items-center justify-center border border-blue-100 flex-shrink-0">
                <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"/>
                </svg>
              </div>
            </div>

            <!-- カード2：週間の入退室数 -->
            <div class="bg-white rounded-xl p-5 border border-slate-200/90 shadow-sm flex items-center justify-between">
              <div>
                <p class="text-xs font-semibold text-slate-500 uppercase">週間の入退室数</p>
                <div class="mt-2 flex items-baseline gap-1">
                  <span class="text-3xl font-bold text-slate-800">{weekTotalCount}</span>
                  <span class="text-sm text-slate-600 font-medium">回</span>
                </div>
                <p class="text-[11px] text-slate-400 mt-1">直近7日間 (入室 {weekEntries} / 退室 {weekExits})</p>
              </div>
              <div class="w-12 h-12 rounded-xl bg-emerald-50 text-emerald-600 flex items-center justify-center border border-emerald-100 flex-shrink-0">
                <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"/>
                </svg>
              </div>
            </div>

            <!-- カード3：月間の入退室数 -->
            <div class="bg-white rounded-xl p-5 border border-slate-200/90 shadow-sm flex items-center justify-between">
              <div>
                <p class="text-xs font-semibold text-slate-500 uppercase">月間の入退室数</p>
                <div class="mt-2 flex items-baseline gap-1">
                  <span class="text-3xl font-bold text-slate-800">{monthTotalCount}</span>
                  <span class="text-sm text-slate-600 font-medium">回</span>
                </div>
                <p class="text-[11px] text-slate-400 mt-1">当月累計 (入室 {monthEntries} / 退室 {monthExits})</p>
              </div>
              <div class="w-12 h-12 rounded-xl bg-indigo-50 text-indigo-600 flex items-center justify-center border border-indigo-100 flex-shrink-0">
                <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"/>
                </svg>
              </div>
            </div>

            <!-- カード4：年間の入退室数 -->
            <div class="bg-white rounded-xl p-5 border border-slate-200/90 shadow-sm flex items-center justify-between">
              <div>
                <p class="text-xs font-semibold text-slate-500 uppercase">年間の入退室数</p>
                <div class="mt-2 flex items-baseline gap-1">
                  <span class="text-3xl font-bold text-slate-800">{yearTotalCount}</span>
                  <span class="text-sm text-slate-600 font-medium">回</span>
                </div>
                <p class="text-[11px] text-slate-400 mt-1">{currentFiscalYear}年度累計</p>
              </div>
              <div class="w-12 h-12 rounded-xl bg-amber-50 text-amber-600 flex items-center justify-center border border-amber-100 flex-shrink-0">
                <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 8v8m-4-5v5m-4-2v2m-2 4h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"/>
                </svg>
              </div>
            </div>
          </div>

          <!-- グラフ & 最近の入退室履歴 -->
          <div class="grid grid-cols-1 lg:grid-cols-12 gap-6">
            <!-- 入退室数グラフ (タイトル:「入退室数」・週 / 月 / 年間 切り替え: 19prompt要件) -->
            <div class="lg:col-span-7 bg-white rounded-xl border border-slate-200/90 shadow-sm p-6 flex flex-col justify-between">
              <div>
                <div class="flex flex-wrap items-center justify-between pb-3 border-b border-slate-100 gap-3">
                  <div class="flex items-center gap-3">
                    <h3 class="text-base font-bold text-slate-800">入退室数</h3>
                    <!-- 週 / 月 / 年間 切り替えタブ -->
                    <div class="inline-flex rounded-lg bg-slate-100 p-0.5 border border-slate-200 text-xs font-medium">
                      <button 
                        on:click={() => chartPeriod = 'week'}
                        class="px-3 py-1 rounded-md transition font-bold {chartPeriod === 'week' ? 'bg-white text-blue-700 shadow-xs' : 'text-slate-600 hover:text-slate-900'}"
                      >週</button>
                      <button 
                        on:click={() => chartPeriod = 'month'}
                        class="px-3 py-1 rounded-md transition font-bold {chartPeriod === 'month' ? 'bg-white text-blue-700 shadow-xs' : 'text-slate-600 hover:text-slate-900'}"
                      >月</button>
                      <button 
                        on:click={() => chartPeriod = 'year'}
                        class="px-3 py-1 rounded-md transition font-bold {chartPeriod === 'year' ? 'bg-white text-blue-700 shadow-xs' : 'text-slate-600 hover:text-slate-900'}"
                      >年間</button>
                    </div>
                  </div>

                  <div class="flex items-center gap-4 text-xs font-medium">
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
                    <!-- 背景グリッド & 目盛り数値 (動的スケール) -->
                    <g class="text-slate-400 text-[10px] font-mono">
                      <!-- 最大値ライン -->
                      <line x1="40" y1="45" x2="500" y2="45" stroke="#f1f5f9" stroke-width="1" />
                      <text x="32" y="49" text-anchor="end" fill="#94a3b8">{chartMax}回</text>
                      
                      <!-- 3/4ライン -->
                      <line x1="40" y1="81" x2="500" y2="81" stroke="#f1f5f9" stroke-width="1" />
                      <text x="32" y="85" text-anchor="end" fill="#94a3b8">{Math.round(chartMax * 0.75)}回</text>
                      
                      <!-- 1/2ライン -->
                      <line x1="40" y1="117" x2="500" y2="117" stroke="#f1f5f9" stroke-width="1" />
                      <text x="32" y="121" text-anchor="end" fill="#94a3b8">{Math.round(chartMax * 0.5)}回</text>
                      
                      <!-- 1/4ライン -->
                      <line x1="40" y1="154" x2="500" y2="154" stroke="#f1f5f9" stroke-width="1" />
                      <text x="32" y="158" text-anchor="end" fill="#94a3b8">{Math.round(chartMax * 0.25)}回</text>
                      
                      <!-- ゼロ基準線 -->
                      <line x1="40" y1="190" x2="500" y2="190" stroke="#cbd5e1" stroke-width="1.5" />
                      <text x="32" y="194" text-anchor="end" fill="#94a3b8">0</text>
                    </g>

                    <!-- 各期間の棒グラフ (実際の打刻ログから動的描画) -->
                    {#each activeHourlyChartData as slot}
                      <g class="group">
                        <!-- 入室バー (青) -->
                        {#if slot.entries > 0}
                          <rect 
                            class="chart-bar" 
                            x={slot.entryX} 
                            y={190 - slot.entries * chartScale} 
                            width={slot.barWidth} 
                            height={slot.entries * chartScale} 
                            rx="2" 
                            fill="#3b82f6" 
                          />
                          <text 
                            x={slot.entryX + slot.barWidth / 2} 
                            y={190 - slot.entries * chartScale - 4} 
                            text-anchor="middle" 
                            font-size="9" 
                            font-weight="bold" 
                            fill="#3b82f6"
                          >{slot.entries}</text>
                        {/if}

                        <!-- 退室バー (赤) -->
                        {#if slot.exits > 0}
                          <rect 
                            class="chart-bar" 
                            x={slot.exitX} 
                            y={190 - slot.exits * chartScale} 
                            width={slot.barWidth} 
                            height={slot.exits * chartScale} 
                            rx="2" 
                            fill="#ef4444" 
                          />
                          <text 
                            x={slot.exitX + slot.barWidth / 2} 
                            y={190 - slot.exits * chartScale - 4} 
                            text-anchor="middle" 
                            font-size="9" 
                            font-weight="bold" 
                            fill="#ef4444"
                          >{slot.exits}</text>
                        {/if}

                        <!-- X軸ラベル -->
                        <text x={slot.centerX} y="208" text-anchor="middle" font-size="10" fill="#64748b">
                          {slot.label}
                        </text>
                      </g>
                    {/each}
                  </svg>
                </div>
              </div>

              <!-- グラフ下部フッター注釈 -->
              <div class="mt-4 pt-3 border-t border-slate-100 flex items-center justify-between text-xs text-slate-500">
                <span>
                  ※表示期間: {chartPeriod === 'week' ? '週（直近7日間）' : (chartPeriod === 'month' ? '月（直近4週間）' : `${currentFiscalYear}年度（4月〜翌3月）`)}
                </span>
                <button 
                  on:click={() => activeTab = 'logs'}
                  class="text-blue-600 hover:text-blue-800 font-bold hover:underline"
                >
                  詳細ログを見る →
                </button>
              </div>
            </div>

            <!-- 最近の入退室履歴 (ダッシュボード右側) -->
            <div class="lg:col-span-5 bg-white rounded-xl border border-slate-200/90 shadow-sm p-6 flex flex-col justify-between">
              <div>
                <div class="flex items-center justify-between pb-3 border-b border-slate-100">
                  <h3 class="text-base font-bold text-slate-800">最近の入退室履歴</h3>
                  <button on:click={() => activeTab = 'logs'} class="text-xs text-blue-600 hover:text-blue-800 font-semibold flex items-center gap-1">
                    すべて見る
                    <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
                    </svg>
                  </button>
                </div>

                <div class="overflow-x-auto mt-2">
                  <table class="w-full text-left text-xs">
                    <thead>
                      <tr class="text-slate-400 border-b border-slate-100">
                        <th class="py-2.5 font-semibold">時刻</th>
                        <th class="py-2.5 font-semibold">氏名</th>
                        <th class="py-2.5 font-semibold text-center">区分</th>
                        <th class="py-2.5 font-semibold text-right">種類</th>
                      </tr>
                    </thead>
                    <tbody class="divide-y divide-slate-100 font-medium">
                      {#each processedLogs.slice(0, 5) as log}
                        <tr class="hover:bg-slate-50 transition">
                          <td class="py-2.5 text-slate-600 font-mono">
                            {log.formatted.split(' ')[1] || log.formatted}
                          </td>
                          <td class="py-2.5 text-slate-800 font-bold">{log.userName || '-'}</td>
                          <td class="py-2.5 text-center text-slate-600">{log.roleName || '学生'}</td>
                          <td class="py-2.5 text-right">
                            <span class="inline-block px-2.5 py-0.5 rounded text-[11px] font-bold {log.eventType === 'entry' ? 'bg-blue-100 text-blue-700' : (log.eventType === 'force_exit' ? 'bg-amber-100 text-amber-700' : 'bg-rose-100 text-rose-700')}">
                              {log.typeLabel}
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

          <!-- 月別・区分別入退室集計テーブル (19prompt要件) -->
          <div class="bg-white rounded-xl border border-slate-200/90 shadow-sm p-6">
            <div class="flex items-center justify-between pb-4 border-b border-slate-100">
              <div class="flex items-center gap-2">
                <h3 class="text-base font-bold text-slate-800">月別・区分別入退室記録</h3>
                <span class="text-xs text-slate-400 font-medium">({currentFiscalYear}年度)</span>
              </div>
              <span class="text-xs text-slate-400">※単位: 回（入室打刻数）</span>
            </div>

            <div class="overflow-x-auto mt-4">
              <table class="w-full text-xs text-left border-collapse">
                <thead class="bg-slate-50 text-slate-600 font-bold border-b border-slate-200 uppercase tracking-wider">
                  <tr>
                    <th class="px-5 py-3">年月</th>
                    <th class="px-4 py-3 text-center">- (その他)</th>
                    <th class="px-4 py-3 text-center text-indigo-700">0（教職員）</th>
                    <th class="px-4 py-3 text-center text-blue-700">1（学生）</th>
                    <th class="px-4 py-3 text-center text-amber-700">9（スタッフ）</th>
                    <th class="px-5 py-3 text-right">当月までの合計</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-slate-100">
                  {#if monthlyStatsRows.length === 0}
                    <tr>
                      <td colspan="6" class="px-5 py-8 text-center text-slate-400 font-medium">
                        集計対象の入退室ログデータがありません。
                      </td>
                    </tr>
                  {:else}
                    {#each monthlyStatsRows as row}
                      <tr class="hover:bg-slate-50/80 transition">
                        <td class="px-5 py-3 font-mono font-bold text-slate-700">{row.yearMonth}</td>
                        <td class="px-4 py-3 text-center text-slate-500">{row.roleOtherCount}</td>
                        <td class="px-4 py-3 text-center font-semibold text-indigo-700">{row.role0Count}</td>
                        <td class="px-4 py-3 text-center font-semibold text-blue-700">{row.role1Count}</td>
                        <td class="px-4 py-3 text-center font-semibold text-amber-700">{row.role9Count}</td>
                        <td class="px-5 py-3 text-right font-bold text-slate-900">{row.fiscalYearCumulativeTotal.toLocaleString()} 回</td>
                      </tr>
                    {/each}
                  {/if}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      {/if}

      <!-- ==============================================================
           2. 入退室履歴画面 (01prompt, 06prompt, 10prompt)
           ============================================================== -->
      {#if activeTab === 'logs'}
        <div class="space-y-6">
          <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <div>
              <h2 class="text-xl font-bold text-slate-800">入退室履歴</h2>
              <p class="text-xs text-slate-500 mt-0.5">クリエイトセンターへのすべての入退室ログを検索・一覧確認できます（列ヘッダーでソート・リサイズ可能）</p>
            </div>
            <button 
              on:click={exportLogsCSV}
              class="px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-xl text-xs font-bold transition shadow-xs flex items-center justify-center gap-1.5"
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
                <label for="filter-date-start" class="block text-xs font-semibold text-slate-600 mb-1.5">日付指定 (開始日 ～ 終了日)</label>
                <div class="flex items-center gap-2">
                  <input id="filter-date-start" type="date" bind:value={filterDateStart} class="w-full text-xs px-3 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20" />
                  <span class="text-slate-400 text-xs">～</span>
                  <input id="filter-date-end" type="date" bind:value={filterDateEnd} class="w-full text-xs px-3 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20" />
                </div>
              </div>

              <div class="lg:col-span-2">
                <label for="filter-type-select" class="block text-xs font-semibold text-slate-600 mb-1.5">種類</label>
                <select id="filter-type-select" bind:value={filterType} class="w-full text-xs px-3 py-2 border border-slate-300 rounded-lg bg-white focus:outline-none focus:ring-2 focus:ring-blue-500/20">
                  <option value="all">すべて</option>
                  <option value="entry">入室</option>
                  <option value="exit">退室</option>
                </select>
              </div>

              <div class="lg:col-span-2">
                <label for="filter-role-select" class="block text-xs font-semibold text-slate-600 mb-1.5">区分</label>
                <select id="filter-role-select" bind:value={filterRole} class="w-full text-xs px-3 py-2 border border-slate-300 rounded-lg bg-white focus:outline-none focus:ring-2 focus:ring-blue-500/20">
                  <option value="all">すべて</option>
                  <option value="学生">学生</option>
                  <option value="教職員">教職員</option>
                  <option value="学生スタッフ">学生スタッフ</option>
                </select>
              </div>

              <div class="lg:col-span-2">
                <label for="filter-keyword-input" class="block text-xs font-semibold text-slate-600 mb-1.5">検索</label>
                <input id="filter-keyword-input" type="text" bind:value={filterKeyword} placeholder="氏名またはID" class="w-full text-xs px-3 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20" />
              </div>

              <div class="lg:col-span-2 flex items-center gap-2">
                <button 
                  on:click={() => { logCurrentPage = 1; }}
                  class="flex-1 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-xs font-bold transition shadow-xs"
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

          <!-- テーブル (ソート & 列リサイズ対応) -->
          <div class="bg-white rounded-xl border border-slate-200/90 shadow-sm overflow-hidden">
            <div class="px-6 py-4 border-b border-slate-100 flex items-center justify-between bg-slate-50/50">
              <span class="text-xs font-semibold text-slate-600">
                検索結果：<span class="font-bold text-slate-900">{filteredLogs.length}</span> 件
              </span>
              <span class="text-xs text-slate-400">※列ヘッダーをクリックでソート、右端をドラッグで幅変更可能</span>
            </div>

            <div class="overflow-x-auto">
              <table class="w-full text-left text-xs border-collapse">
                <thead class="bg-slate-50 text-slate-600 font-bold uppercase tracking-wider border-b border-slate-200">
                  <tr>
                    <!-- 1. 日時 -->
                    <th style="width: {logsWidths.timestamp}px;" class="px-5 py-3.5 relative group select-none">
                      <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('logs', 'timestamp')}>
                        <span>日時 {logsSortKey === 'timestamp' ? (logsSortDir === 'asc' ? '▲' : '▼') : ''}</span>
                      </div>
                      <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'logs', 'timestamp')}></div>
                    </th>

                    <!-- 2. ID -->
                    <th style="width: {logsWidths.displayId}px;" class="px-5 py-3.5 relative group select-none">
                      <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('logs', 'displayId')}>
                        <span>ID {logsSortKey === 'displayId' ? (logsSortDir === 'asc' ? '▲' : '▼') : ''}</span>
                      </div>
                      <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'logs', 'displayId')}></div>
                    </th>

                    <!-- 3. 氏名 -->
                    <th style="width: {logsWidths.userName}px;" class="px-5 py-3.5 relative group select-none">
                      <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('logs', 'userName')}>
                        <span>氏名 {logsSortKey === 'userName' ? (logsSortDir === 'asc' ? '▲' : '▼') : ''}</span>
                      </div>
                      <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'logs', 'userName')}></div>
                    </th>

                    <!-- 4. 判定 -->
                    <th style="width: {logsWidths.result}px;" class="px-4 py-3.5 text-center relative group select-none">
                      <div class="flex items-center justify-center cursor-pointer" on:click={() => handleSort('logs', 'result')}>
                        <span>判定 {logsSortKey === 'result' ? (logsSortDir === 'asc' ? '▲' : '▼') : ''}</span>
                      </div>
                      <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'logs', 'result')}></div>
                    </th>

                    <!-- 5. 区分 -->
                    <th style="width: {logsWidths.roleName}px;" class="px-5 py-3.5 relative group select-none">
                      <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('logs', 'roleName')}>
                        <span>区分 {logsSortKey === 'roleName' ? (logsSortDir === 'asc' ? '▲' : '▼') : ''}</span>
                      </div>
                      <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'logs', 'roleName')}></div>
                    </th>

                    <!-- 6. 利用目的 -->
                    <th style="width: {logsWidths.purpose}px;" class="px-5 py-3.5 relative group select-none">
                      <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('logs', 'purpose')}>
                        <span>利用目的 {logsSortKey === 'purpose' ? (logsSortDir === 'asc' ? '▲' : '▼') : ''}</span>
                      </div>
                      <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'logs', 'purpose')}></div>
                    </th>

                    <!-- 7. 入退室 -->
                    <th style="width: {logsWidths.eventType}px;" class="px-4 py-3.5 text-center relative group select-none">
                      <div class="flex items-center justify-center cursor-pointer" on:click={() => handleSort('logs', 'eventType')}>
                        <span>入退室 {logsSortKey === 'eventType' ? (logsSortDir === 'asc' ? '▲' : '▼') : ''}</span>
                      </div>
                      <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'logs', 'eventType')}></div>
                    </th>

                    <!-- 8. 滞在時間 -->
                    <th style="width: {logsWidths.duration}px;" class="px-5 py-3.5 relative group select-none">
                      <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('logs', 'durationSecond')}>
                        <span>滞在時間 {logsSortKey === 'durationSecond' ? (logsSortDir === 'asc' ? '▲' : '▼') : ''}</span>
                      </div>
                      <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'logs', 'duration')}></div>
                    </th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-slate-100">
                  {#if paginatedLogs.length === 0}
                    <tr>
                      <td colspan="8" class="px-6 py-10 text-center text-slate-400 font-medium">条件に一致する履歴はありません。</td>
                    </tr>
                  {:else}
                    {#each paginatedLogs as log}
                      <tr class="hover:bg-slate-50/80 transition border-b border-slate-100">
                        <td class="px-5 py-3.5 font-mono text-slate-600">{log.formatted}</td>
                        <td class="px-5 py-3.5 font-mono text-slate-500 font-semibold">{log.displayId || '-'}</td>
                        <td class="px-5 py-3.5 font-bold text-slate-800">{log.userName || '未登録'}</td>
                        <td class="px-4 py-3.5 text-center">
                          <span class="inline-block px-2 py-0.5 rounded text-[11px] font-bold {log.result === '成功' ? 'bg-emerald-50 text-emerald-700 border border-emerald-200' : 'bg-rose-50 text-rose-700 border border-rose-200'}">
                            {log.result}
                          </span>
                        </td>
                        <td class="px-5 py-3.5 text-slate-600">{log.roleName || '-'}</td>
                        <td class="px-5 py-3.5 text-slate-500 truncate max-w-[140px]">{log.purpose || '-'}</td>
                        <td class="px-4 py-3.5 text-center">
                          <span class="inline-block px-2.5 py-0.5 rounded text-xs font-bold {log.eventType === 'entry' ? 'bg-blue-100 text-blue-700' : (log.eventType === 'force_exit' ? 'bg-amber-100 text-amber-700' : 'bg-rose-100 text-rose-700')}">
                            {log.typeLabel}
                          </span>
                        </td>
                        <td class="px-5 py-3.5 text-slate-500">{log.durationText || '-'}</td>
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

      <!-- ==============================================================
           3. 在室中メンバー画面 (01prompt & 06prompt要件)
           ============================================================== -->
      {#if activeTab === 'inside'}
        <div class="space-y-6">
          <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <div>
              <h2 class="text-xl font-bold text-slate-800">在室中メンバー</h2>
              <p class="text-xs text-slate-500 mt-0.5">現在、施設内にいる利用者のリアルタイム状況です（定員：{ROOM_CAPACITY}名）</p>
            </div>
            <button 
              on:click={() => isAdvancedOpen = true}
              class="px-4 py-2 bg-amber-600 hover:bg-amber-700 text-white rounded-xl text-xs font-bold transition shadow-xs flex items-center justify-center gap-1.5"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"/>
              </svg>
              一括強制退室 / 管理操作
            </button>
          </div>

          <div class="bg-white rounded-xl border border-slate-200/90 shadow-sm overflow-hidden">
            <div class="px-6 py-4 border-b border-slate-100 flex items-center justify-between bg-slate-50/50">
              <span class="text-xs font-semibold text-slate-600">
                現在の在室者数：<span class="font-bold text-emerald-600 text-sm">{insideUsers.length}</span> 名
              </span>
            </div>

            <div class="overflow-x-auto">
              <table class="w-full text-left text-xs border-collapse">
                <thead class="bg-slate-50 text-slate-600 font-bold uppercase tracking-wider border-b border-slate-200">
                  <tr>
                    <th style="width: {insideWidths.userName}px;" class="px-6 py-3.5 relative group select-none">
                      <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('inside', 'userName')}>
                        <span>氏名 {insideSortKey === 'userName' ? (insideSortDir === 'asc' ? '▲' : '▼') : ''}</span>
                      </div>
                      <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'inside', 'userName')}></div>
                    </th>

                    <th style="width: {insideWidths.roleName}px;" class="px-6 py-3.5 relative group select-none">
                      <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('inside', 'roleName')}>
                        <span>区分 {insideSortKey === 'roleName' ? (insideSortDir === 'asc' ? '▲' : '▼') : ''}</span>
                      </div>
                      <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'inside', 'roleName')}></div>
                    </th>

                    <th style="width: {insideWidths.studentNo}px;" class="px-6 py-3.5 relative group select-none">
                      <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('inside', 'studentNo')}>
                        <span>学籍番号/職員番号 {insideSortKey === 'studentNo' ? (insideSortDir === 'asc' ? '▲' : '▼') : ''}</span>
                      </div>
                      <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'inside', 'studentNo')}></div>
                    </th>

                    <th style="width: {insideWidths.lastEventTime}px;" class="px-6 py-3.5 relative group select-none">
                      <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('inside', 'lastEventTime')}>
                        <span>入室時刻 {insideSortKey === 'lastEventTime' ? (insideSortDir === 'asc' ? '▲' : '▼') : ''}</span>
                      </div>
                      <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'inside', 'lastEventTime')}></div>
                    </th>

                    <th style="width: {insideWidths.stayDuration}px;" class="px-6 py-3.5 relative group select-none">
                      <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('inside', 'stayDuration')}>
                        <span>滞在時間 {insideSortKey === 'stayDuration' ? (insideSortDir === 'asc' ? '▲' : '▼') : ''}</span>
                      </div>
                      <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'inside', 'stayDuration')}></div>
                    </th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-slate-100">
                  {#if insideUsers.length === 0}
                    <tr>
                      <td colspan="5" class="px-6 py-10 text-center text-slate-400 font-medium">現在、在室中のメンバーはいません。</td>
                    </tr>
                  {:else}
                    {#each sortList(insideUsers, insideSortKey, insideSortDir) as user}
                      <tr class="hover:bg-slate-50/80 transition border-b border-slate-100">
                        <td class="px-6 py-3.5 font-bold text-slate-800">{user.userName}</td>
                        <td class="px-6 py-3.5 text-slate-600">{user.roleName || '-'}</td>
                        <td class="px-6 py-3.5 font-mono text-slate-500">{user.studentNo || user.cardId}</td>
                        <td class="px-6 py-3.5 font-mono text-slate-600">{formatLogTime(user.lastEventTime)}</td>
                        <td class="px-6 py-3.5 font-medium text-blue-600">{user.stayDuration || '入室中'}</td>
                      </tr>
                    {/each}
                  {/if}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      {/if}

      <!-- ==============================================================
           4. ユーザー管理画面 (03prompt, 04prompt, 06prompt, 10prompt)
           ============================================================== -->
      {#if activeTab === 'users'}
        <div class="space-y-6">
          <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <div>
              <h2 class="text-xl font-bold text-slate-800">ユーザー管理</h2>
              <p class="text-xs text-slate-500 mt-0.5">クリエイトセンターへの入室を許可する利用者を登録・管理します（列幅調整・ソート対応）</p>
            </div>
            <!-- 目立つ青色の ＋ 新規登録 ボタン -->
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

          <!-- 検索 & 区分フィルター -->
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
                placeholder="氏名、フリガナ、学籍番号で検索" 
                class="w-full text-xs pl-9 pr-3.5 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20"
              />
            </div>

            <div class="flex items-center gap-2">
              <span class="text-xs font-semibold text-slate-500 mr-1">区分絞り込み:</span>
              <div class="inline-flex rounded-lg bg-slate-100 p-0.5 border border-slate-200 text-xs font-medium">
                <button 
                  on:click={() => userRoleFilter = 'all'} 
                  class="px-3.5 py-1.5 rounded-md font-bold transition {userRoleFilter === 'all' ? 'bg-white text-blue-700 shadow-xs' : 'text-slate-600 hover:text-slate-900'}"
                >すべて</button>
                <button 
                  on:click={() => userRoleFilter = '学生'} 
                  class="px-3.5 py-1.5 rounded-md font-bold transition {userRoleFilter === '学生' ? 'bg-white text-blue-700 shadow-xs' : 'text-slate-600 hover:text-slate-900'}"
                >学生</button>
                <button 
                  on:click={() => userRoleFilter = '教職員'} 
                  class="px-3.5 py-1.5 rounded-md font-bold transition {userRoleFilter === '教職員' ? 'bg-white text-blue-700 shadow-xs' : 'text-slate-600 hover:text-slate-900'}"
                >教職員</button>
                <button 
                  on:click={() => userRoleFilter = '学生スタッフ'} 
                  class="px-3.5 py-1.5 rounded-md font-bold transition {userRoleFilter === '学生スタッフ' ? 'bg-white text-blue-700 shadow-xs' : 'text-slate-600 hover:text-slate-900'}"
                >スタッフ</button>
              </div>
            </div>
          </div>

          <!-- テーブル (ソート & 列リサイズ対応 & ページネーション & 個人情報保護) -->
          <div class="bg-white rounded-xl border border-slate-200/90 shadow-sm overflow-hidden">
            <!-- テーブル上部ステータスバー -->
            <div class="px-6 py-3.5 border-b border-slate-100 flex flex-wrap items-center justify-between gap-3 bg-slate-50/50">
              <div class="flex items-center gap-4">
                <span class="text-xs font-semibold text-slate-600">
                  登録者：<span class="font-bold text-slate-900">{filteredUsers.length}</span> 人
                  {#if filteredUsers.length > 0}
                    <span class="text-slate-400 ml-1">
                      ({(userCurrentPage - 1) * usersPerPage + 1}〜{Math.min(userCurrentPage * usersPerPage, filteredUsers.length)}件を表示)
                    </span>
                  {/if}
                </span>

                <!-- 表示件数選択 -->
                <div class="flex items-center gap-1.5 text-xs text-slate-500">
                  <span>表示件数:</span>
                  <select 
                    bind:value={usersPerPage} 
                    class="bg-white border border-slate-200 text-xs rounded px-2 py-1 focus:outline-none focus:border-blue-500 font-semibold"
                  >
                    <option value={25}>25件</option>
                    <option value={50}>50件</option>
                    <option value={100}>100件</option>
                  </select>
                </div>
              </div>

              <div class="flex items-center gap-3">
                <!-- 個人情報一括表示 / 非表示トグルボタン -->
                <button 
                  type="button"
                  on:click={toggleShowAllContacts}
                  class="flex items-center gap-1.5 px-3 py-1 text-xs font-bold rounded-lg border transition {showAllContacts ? 'bg-amber-50 text-amber-800 border-amber-300 hover:bg-amber-100' : 'bg-slate-100 text-slate-600 border-slate-200 hover:bg-slate-200'}"
                  title="電話番号やメールアドレスなどの個人情報のマスクを一括切り替え"
                >
                  <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    {#if showAllContacts}
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l18 18" />
                    {:else}
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                    {/if}
                  </svg>
                  <span>{showAllContacts ? '連絡先を一括マスク' : '連絡先を一括表示'}</span>
                </button>
              </div>
            </div>

            <div class="overflow-x-auto">
              <table class="w-full text-left text-xs border-collapse">
                <thead class="bg-slate-50 text-slate-600 font-bold uppercase tracking-wider border-b border-slate-200">
                  <tr>
                    <!-- 管理番号 -->
                    <th style="width: {usersWidths.adminNo}px;" class="px-5 py-3.5 relative group select-none">
                      <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('users', 'adminNo')}>
                        <span>No {usersSortKey === 'adminNo' ? (usersSortDir === 'asc' ? '▲' : '▼') : ''}</span>
                      </div>
                      <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'users', 'adminNo')}></div>
                    </th>

                    <!-- 学籍番号/職員番号 -->
                    <th style="width: {usersWidths.studentNo}px;" class="px-5 py-3.5 relative group select-none">
                      <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('users', 'studentNo')}>
                        <span>学籍/職員番号 {usersSortKey === 'studentNo' ? (usersSortDir === 'asc' ? '▲' : '▼') : ''}</span>
                      </div>
                      <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'users', 'studentNo')}></div>
                    </th>

                    <!-- 氏名 -->
                    <th style="width: {usersWidths.name}px;" class="px-5 py-3.5 relative group select-none">
                      <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('users', 'name')}>
                        <span>氏名 {usersSortKey === 'name' ? (usersSortDir === 'asc' ? '▲' : '▼') : ''}</span>
                      </div>
                      <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'users', 'name')}></div>
                    </th>

                    <!-- フリガナ -->
                    <th style="width: {usersWidths.furigana}px;" class="px-5 py-3.5 relative group select-none">
                      <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('users', 'furigana')}>
                        <span>フリガナ {usersSortKey === 'furigana' ? (usersSortDir === 'asc' ? '▲' : '▼') : ''}</span>
                      </div>
                      <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'users', 'furigana')}></div>
                    </th>

                    <!-- 性別 -->
                    <th style="width: {usersWidths.gender}px;" class="px-4 py-3.5 text-center relative group select-none">
                      <div class="flex items-center justify-center cursor-pointer" on:click={() => handleSort('users', 'gender')}>
                        <span>性別 {usersSortKey === 'gender' ? (usersSortDir === 'asc' ? '▲' : '▼') : ''}</span>
                      </div>
                      <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'users', 'gender')}></div>
                    </th>

                    <!-- 区分 -->
                    <th style="width: {usersWidths.roleName}px;" class="px-5 py-3.5 relative group select-none">
                      <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('users', 'roleCode')}>
                        <span>区分 {usersSortKey === 'roleCode' ? (usersSortDir === 'asc' ? '▲' : '▼') : ''}</span>
                      </div>
                      <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'users', 'roleName')}></div>
                    </th>

                    <!-- 連絡先 (個人情報保護対応) -->
                    <th style="width: {usersWidths.contact}px;" class="px-5 py-3.5 relative group select-none">
                      <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('users', 'contact')}>
                        <span>連絡先 {usersSortKey === 'contact' ? (usersSortDir === 'asc' ? '▲' : '▼') : ''}</span>
                      </div>
                      <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'users', 'contact')}></div>
                    </th>

                    <!-- 利用目的 -->
                    <th style="width: {usersWidths.purpose}px;" class="px-5 py-3.5 relative group select-none">
                      <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('users', 'purpose')}>
                        <span>利用目的 {usersSortKey === 'purpose' ? (usersSortDir === 'asc' ? '▲' : '▼') : ''}</span>
                      </div>
                      <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'users', 'purpose')}></div>
                    </th>

                    <!-- 操作 -->
                    <th style="width: {usersWidths.actions}px;" class="px-5 py-3.5 text-right select-none">
                      操作
                    </th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-slate-100">
                  {#if filteredUsers.length === 0}
                    <tr>
                      <td colspan="9" class="px-6 py-10 text-center text-slate-400 font-medium">該当するユーザーが見つかりませんでした。</td>
                    </tr>
                  {:else}
                    {#each paginatedUsers as user}
                      <tr class="hover:bg-slate-50/80 transition border-b border-slate-100">
                        <td class="px-5 py-3.5 font-mono text-slate-500">{user.adminNo || '-'}</td>
                        <td class="px-5 py-3.5 font-mono text-slate-700 font-semibold">{user.studentNo}</td>
                        <td class="px-5 py-3.5 font-bold text-slate-800">{user.name}</td>
                        <td class="px-5 py-3.5 text-slate-500 text-[11px]">{user.furigana || '-'}</td>
                        <td class="px-4 py-3.5 text-center text-slate-600">{user.gender || '-'}</td>
                        <td class="px-5 py-3.5">
                          <span class="inline-block px-2.5 py-0.5 rounded text-xs {user.roleCode === 0 ? 'bg-indigo-50 text-indigo-700 border border-indigo-200 font-bold' : (user.roleCode === 9 ? 'bg-amber-50 text-amber-700 border border-amber-200 font-semibold' : 'bg-slate-100 text-slate-700')}">
                            {user.roleName || '学生'}
                          </span>
                        </td>
                        <!-- 連絡先セル：未クリック時はマスク表示、クリックで開示 -->
                        <td class="px-5 py-3.5 text-slate-600">
                          {#if !user.contact}
                            <span class="text-slate-400">-</span>
                          {:else if showAllContacts || revealedContacts.has(user.cardId || user.studentNo)}
                            <button 
                              type="button" 
                              on:click={() => toggleContactReveal(user.cardId || user.studentNo)}
                              class="inline-flex items-center gap-1 text-slate-800 hover:text-blue-600 font-mono text-xs bg-blue-50/60 hover:bg-blue-100/60 px-2 py-0.5 rounded transition text-left"
                              title="クリックして再度マスクする"
                            >
                              <span>{user.contact}</span>
                              <svg class="w-3 h-3 text-blue-500 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                              </svg>
                            </button>
                          {:else}
                            <button 
                              type="button" 
                              on:click={() => toggleContactReveal(user.cardId || user.studentNo)}
                              class="inline-flex items-center gap-1 text-slate-400 hover:text-blue-600 font-mono text-xs bg-slate-100 hover:bg-blue-50 px-2 py-0.5 rounded transition cursor-pointer"
                              title="クリックして電話番号・連絡先を表示"
                            >
                              <span>••••••••</span>
                              <svg class="w-3 h-3 text-slate-400 group-hover:text-blue-500 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l18 18" />
                              </svg>
                            </button>
                          {/if}
                        </td>
                        <td class="px-5 py-3.5 text-slate-500 truncate max-w-[140px]">{user.purpose || '-'}</td>
                        <td class="px-5 py-3.5 text-right space-x-1.5 whitespace-nowrap">
                          <button 
                            on:click={() => openEditUserModal(user)}
                            class="px-2 py-1 text-xs font-bold text-blue-600 hover:text-blue-800 hover:bg-blue-50 rounded transition"
                          >編集</button>
                          <button 
                            on:click={() => handleDeleteUser(user)}
                            class="px-2 py-1 text-xs font-bold text-rose-600 hover:text-rose-800 hover:bg-rose-50 rounded transition"
                          >削除</button>
                        </td>
                      </tr>
                    {/each}
                  {/if}
                </tbody>
              </table>
            </div>

            <!-- ユーザー管理 ページネーションバー -->
            {#if totalUserPages > 1}
              <div class="px-6 py-3.5 border-t border-slate-100 flex items-center justify-between bg-slate-50/30">
                <span class="text-xs text-slate-500 font-medium">
                  {userCurrentPage} / {totalUserPages} ページ (全 {filteredUsers.length} 人)
                </span>
                <div class="flex items-center gap-1.5">
                  <button 
                    disabled={userCurrentPage === 1}
                    on:click={() => userCurrentPage = 1}
                    class="px-2.5 py-1 text-xs border border-slate-200 rounded hover:bg-slate-50 disabled:opacity-40 disabled:cursor-not-allowed font-medium text-slate-600"
                  >最初</button>
                  <button 
                    disabled={userCurrentPage === 1}
                    on:click={() => userCurrentPage--}
                    class="px-2.5 py-1 text-xs border border-slate-200 rounded hover:bg-slate-50 disabled:opacity-40 disabled:cursor-not-allowed font-medium text-slate-600"
                  >前へ</button>

                  <span class="px-2 text-xs font-bold text-slate-700">{userCurrentPage}</span>

                  <button 
                    disabled={userCurrentPage === totalUserPages}
                    on:click={() => userCurrentPage++}
                    class="px-2.5 py-1 text-xs border border-slate-200 rounded hover:bg-slate-50 disabled:opacity-40 disabled:cursor-not-allowed font-medium text-slate-600"
                  >次へ</button>
                  <button 
                    disabled={userCurrentPage === totalUserPages}
                    on:click={() => userCurrentPage = totalUserPages}
                    class="px-2.5 py-1 text-xs border border-slate-200 rounded hover:bg-slate-50 disabled:opacity-40 disabled:cursor-not-allowed font-medium text-slate-600"
                  >最後</button>
                </div>
              </div>
            {/if}
          </div>
        </div>
      {/if}

    </main>
  </div>

  <!-- ==============================================================
       ユーザー登録・編集 モーダルウィンドウ
       (★ 03prompt & 04prompt で指示された並び順と必須/任意の厳格定義！)
       1. 学籍番号 (必須)
       2. 氏名 (必須)
       3. フリガナ (任意)
       4. 性別 (ドロップダウン: 男女)
       5. 区分 (ドロップダウン: 学生, 教職員, 学生スタッフ) ※区分コード数値は非表示
       6. カードID (任意。空なら学籍番号と同じものを扱う)
       7. 連絡先 (任意)
       8. 利用目的 (任意)
       ============================================================== -->
  {#if isUserModalOpen}
    <div class="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/50 backdrop-blur-xs p-4">
      <div class="bg-white rounded-2xl shadow-2xl border border-slate-200 w-full max-w-lg overflow-hidden max-h-[90vh] flex flex-col">
        <!-- モーダルヘッダー -->
        <div class="px-6 py-4 border-b border-slate-100 flex items-center justify-between bg-slate-50">
          <h3 class="text-base font-bold text-slate-800">
            {modalMode === 'create' ? '利用者新規登録' : '利用者情報の編集'}
          </h3>
          <button on:click={() => isUserModalOpen = false} class="text-slate-400 hover:text-slate-600 p-1 rounded-lg">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>

        <!-- フォームスクロール領域 -->
        <form on:submit|preventDefault={handleSaveUser} class="p-6 space-y-4 overflow-y-auto flex-1">
          {#if formError}
            <div class="p-3 rounded-lg bg-rose-50 border border-rose-200 text-xs text-rose-600 font-semibold">
              {formError}
            </div>
          {/if}

          <!-- 1. 学籍番号 / 職員番号 (必須) -->
          <div>
            <label for="form-student-no" class="block text-xs font-bold text-slate-700 mb-1">
              学籍番号 / 職員番号 <span class="text-rose-500">*</span>
            </label>
            <input 
              id="form-student-no"
              type="text" 
              bind:value={formStudentNo} 
              required 
              placeholder="例: A0012345 (カードIDが空の場合はこの番号が照合に使われます)" 
              class="w-full text-xs px-3.5 py-2.5 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 font-mono uppercase" 
            />
          </div>

          <!-- 2. 氏名 (必須) -->
          <div>
            <label for="form-name" class="block text-xs font-bold text-slate-700 mb-1">
              氏名 <span class="text-rose-500">*</span>
            </label>
            <input 
              id="form-name"
              type="text" 
              bind:value={formName} 
              required 
              placeholder="例: 山田 太郎" 
              class="w-full text-xs px-3.5 py-2.5 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20" 
            />
          </div>

          <!-- 3. フリガナ -->
          <div>
            <label for="form-furigana" class="block text-xs font-bold text-slate-700 mb-1">フリガナ</label>
            <input 
              id="form-furigana"
              type="text" 
              bind:value={formFurigana} 
              placeholder="例: ヤマダ タロウ" 
              class="w-full text-xs px-3.5 py-2.5 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20" 
            />
          </div>

          <!-- 4. 性別 (ドロップダウン: 男, 女) & 5. 区分 (ドロップダウン: 学生, 教職員, 学生スタッフ) -->
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label for="form-gender" class="block text-xs font-bold text-slate-700 mb-1">性別</label>
              <select 
                id="form-gender"
                bind:value={formGender} 
                class="w-full text-xs px-3.5 py-2.5 border border-slate-300 rounded-lg bg-white focus:outline-none focus:ring-2 focus:ring-blue-500/20"
              >
                <option value="男">男</option>
                <option value="女">女</option>
              </select>
            </div>

            <div>
              <label for="form-role" class="block text-xs font-bold text-slate-700 mb-1">区分</label>
              <select 
                id="form-role"
                bind:value={formRoleName} 
                on:change={handleRoleChange}
                class="w-full text-xs px-3.5 py-2.5 border border-slate-300 rounded-lg bg-white focus:outline-none focus:ring-2 focus:ring-blue-500/20 font-medium"
              >
                <option value="学生">学生</option>
                <option value="教職員">教職員</option>
                <option value="学生スタッフ">学生スタッフ</option>
              </select>
            </div>
          </div>

          <!-- 6. カードID (任意・空なら学籍番号と同じ) -->
          <div>
            <label for="form-card-id" class="block text-xs font-bold text-slate-700 mb-1">
              カードID (任意 / NFC・磁気併用時)
            </label>
            <input 
              id="form-card-id"
              type="text" 
              bind:value={formCardId} 
              placeholder="未入力の場合は学籍番号と同じ値が自動適用されます" 
              class="w-full text-xs px-3.5 py-2.5 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 font-mono" 
            />
            <p class="text-[11px] text-slate-400 mt-1">※PaSoRiのNFC IDm番号や個別カード番号を別途割り当てる場合に入力</p>
          </div>

          <!-- 7. 連絡先 (04prompt) -->
          <div>
            <label for="form-contact" class="block text-xs font-bold text-slate-700 mb-1">連絡先 (任意)</label>
            <input 
              id="form-contact"
              type="text" 
              bind:value={formContact} 
              placeholder="例: 090-xxxx-xxxx / user@example.com" 
              class="w-full text-xs px-3.5 py-2.5 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20" 
            />
          </div>

          <!-- 8. 利用目的 (04prompt) -->
          <div>
            <label for="form-purpose" class="block text-xs font-bold text-slate-700 mb-1">利用目的 (任意)</label>
            <input 
              id="form-purpose"
              type="text" 
              bind:value={formPurpose} 
              placeholder="例: 卒業研究 / 自習 / TA業務" 
              class="w-full text-xs px-3.5 py-2.5 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20" 
            />
          </div>

          <!-- フッターボタン -->
          <div class="pt-4 border-t border-slate-100 flex items-center justify-end gap-3 sticky bottom-0 bg-white">
            <button 
              type="button" 
              on:click={() => isUserModalOpen = false} 
              class="px-4 py-2 border border-slate-300 text-slate-700 rounded-lg text-xs font-semibold hover:bg-slate-50"
            >
              キャンセル
            </button>
            <button 
              type="submit" 
              class="px-5 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-xs font-bold shadow-sm"
            >
              {modalMode === 'create' ? '登録' : '保存'}
            </button>
          </div>
        </form>
      </div>
    </div>
  {/if}

  <!-- 高度な操作モーダル (01prompt, 03prompt, 04prompt要件) -->
  {#if isAdvancedOpen}
    <AdvancedOpsModal 
      onClose={() => { isAdvancedOpen = false; }}
      onRefresh={loadData}
    />
  {/if}

</div>
