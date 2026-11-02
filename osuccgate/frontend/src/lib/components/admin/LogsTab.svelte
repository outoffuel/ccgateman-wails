<script>
  import { onDestroy } from 'svelte';
  import { sortList } from '../../utils/sorting.js';
  import { clampPage, getPageItems } from '../../utils/pagination.js';
  import { formatLogTime } from '../../utils/date.js';

  export let logs = [];
  export let processedLogs = null;
  export let availableFiscalYears = [2026];
  export let selectedFiscalYear = 2026;
  export let onSelectFiscalYear = () => {};
  export let onRenewalUser = () => {};
  export let onRegisterUnregistered = () => {};

  // 列幅リサイズ
  let logsWidths = { timestamp: 155, displayId: 105, userName: 120, result: 70, roleName: 95, purpose: 120, eventType: 95, duration: 95, action: 105 };
  let isResizing = false;
  let activeResizeCol = null;
  let startX = 0;
  let startWidth = 0;

  function startResize(e, colKey) {
    e.stopPropagation();
    e.preventDefault();
    isResizing = true;
    activeResizeCol = colKey;
    startX = e.clientX;
    startWidth = logsWidths[colKey];

    window.addEventListener('mousemove', handleResizeMove);
    window.addEventListener('mouseup', handleResizeEnd);
  }

  function handleResizeMove(e) {
    if (!isResizing) return;
    const delta = e.clientX - startX;
    const newWidth = Math.max(50, startWidth + delta);
    logsWidths = { ...logsWidths, [activeResizeCol]: newWidth };
  }

  function handleResizeEnd() {
    isResizing = false;
    activeResizeCol = null;
    window.removeEventListener('mousemove', handleResizeMove);
    window.removeEventListener('mouseup', handleResizeEnd);
  }

  onDestroy(() => {
    if (typeof window !== 'undefined') {
      window.removeEventListener('mousemove', handleResizeMove);
      window.removeEventListener('mouseup', handleResizeEnd);
    }
  });

  // ソート
  let logsSortKey = 'timestamp';
  let logsSortDir = 'desc';

  function handleSort(key) {
    logsSortDir = (logsSortKey === key && logsSortDir === 'asc') ? 'desc' : 'asc';
    logsSortKey = key;
  }

  // 検索・絞り込み・ページネーション・CSV
  let filterDateStart = "";
  let filterDateEnd = "";
  let filterType = "all";   // 'all' | 'entry' | 'exit'
  let filterRole = "all";   // 'all' | '学生' | '教職員' | '学生スタッフ'
  let filterKeyword = "";
  let logCurrentPage = 1;
  let logPageInput = 1;
  const LOGS_PER_PAGE = 10;

  $: effectiveLogs = processedLogs ?? (logs || []).map(l => {
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

  $: sortedLogs = sortList(effectiveLogs, logsSortKey, logsSortDir);

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
  $: logPageItems = getPageItems(totalLogPages, logCurrentPage);
  $: if (logCurrentPage > totalLogPages) setLogPage(totalLogPages);
  $: paginatedLogs = filteredLogs.slice((logCurrentPage - 1) * LOGS_PER_PAGE, logCurrentPage * LOGS_PER_PAGE);

  function setLogPage(page) {
    logCurrentPage = clampPage(page, totalLogPages);
    logPageInput = logCurrentPage;
  }

  function clearLogFilters() {
    filterDateStart = "";
    filterDateEnd = "";
    filterType = "all";
    filterRole = "all";
    filterKeyword = "";
    setLogPage(1);
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
    URL.revokeObjectURL(url);
  }
</script>

<div class="space-y-6">
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
    <div>
      <h2 class="text-xl font-bold text-slate-800">入退室履歴</h2>
      <p class="text-xs text-slate-500 mt-0.5">クリエイトセンターへのすべての入退室ログを検索・一覧確認できます（列ヘッダーでソート・リサイズ可能）</p>
    </div>
    <div class="flex items-center gap-3">
      <!-- 表示年度プルダウン (要件4) -->
      <div class="flex items-center gap-1.5 bg-white border border-slate-300 rounded-xl px-3 py-1.5 shadow-xs">
        <label for="logs-fiscal-year-select" class="text-xs font-bold text-slate-600 whitespace-nowrap">表示年度:</label>
        <select 
          id="logs-fiscal-year-select" 
          value={selectedFiscalYear} 
          on:change={(e) => onSelectFiscalYear(Number(e.target.value))}
          class="text-xs font-bold text-blue-700 bg-transparent focus:outline-none cursor-pointer"
        >
          {#each availableFiscalYears as fy}
            <option value={fy}>{fy}年度</option>
          {/each}
        </select>
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
          on:click={() => setLogPage(1)}
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
              <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('timestamp')}>
                <span>日時 {logsSortKey === 'timestamp' ? (logsSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'timestamp')}></div>
            </th>

            <!-- 2. ID -->
            <th style="width: {logsWidths.displayId}px;" class="px-5 py-3.5 relative group select-none">
              <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('displayId')}>
                <span>ID {logsSortKey === 'displayId' ? (logsSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'displayId')}></div>
            </th>

            <!-- 3. 氏名 -->
            <th style="width: {logsWidths.userName}px;" class="px-5 py-3.5 relative group select-none">
              <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('userName')}>
                <span>氏名 {logsSortKey === 'userName' ? (logsSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'userName')}></div>
            </th>

            <!-- 4. 判定 -->
            <th style="width: {logsWidths.result}px;" class="px-4 py-3.5 text-center relative group select-none">
              <div class="flex items-center justify-center cursor-pointer" on:click={() => handleSort('result')}>
                <span>判定 {logsSortKey === 'result' ? (logsSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'result')}></div>
            </th>

            <!-- 5. 区分 -->
            <th style="width: {logsWidths.roleName}px;" class="px-5 py-3.5 relative group select-none">
              <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('roleName')}>
                <span>区分 {logsSortKey === 'roleName' ? (logsSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'roleName')}></div>
            </th>

            <!-- 6. 利用目的 -->
            <th style="width: {logsWidths.purpose}px;" class="px-5 py-3.5 relative group select-none">
              <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('purpose')}>
                <span>利用目的 {logsSortKey === 'purpose' ? (logsSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'purpose')}></div>
            </th>

            <!-- 7. 入退室 -->
            <th style="width: {logsWidths.eventType}px;" class="px-4 py-3.5 text-center relative group select-none">
              <div class="flex items-center justify-center cursor-pointer" on:click={() => handleSort('eventType')}>
                <span>入退室 {logsSortKey === 'eventType' ? (logsSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'eventType')}></div>
            </th>

            <!-- 8. 滞在時間 -->
            <th style="width: {logsWidths.duration}px;" class="px-5 py-3.5 relative group select-none">
              <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('durationSecond')}>
                <span>滞在時間 {logsSortKey === 'durationSecond' ? (logsSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'duration')}></div>
            </th>

            <!-- 9. 操作 -->
            <th style="width: {logsWidths.action}px;" class="px-4 py-3.5 text-center relative group select-none">
              <div class="flex items-center justify-center">
                <span>操作</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'action')}></div>
            </th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-100">
          {#if paginatedLogs.length === 0}
            <tr>
              <td colspan="9" class="px-6 py-10 text-center text-slate-400 font-medium">条件に一致する履歴はありません。</td>
            </tr>
          {:else}
            {#each paginatedLogs as log}
              <tr class="hover:bg-slate-50/80 transition border-b border-slate-100">
                <td class="px-5 py-3.5 font-mono text-slate-600">{log.formatted}</td>
                <td class="px-5 py-3.5 font-mono text-slate-500 font-semibold">{log.displayId || '-'}</td>
                <td class="px-5 py-3.5 font-bold text-slate-800">{log.userName || '未登録'}</td>
                <td class="px-4 py-3.5 text-center">
                  <span class="inline-block px-2 py-0.5 rounded text-[11px] font-bold {(log.result === '成功' || log.result === '登録済') ? 'bg-emerald-50 text-emerald-700 border border-emerald-200' : 'bg-rose-50 text-rose-700 border border-rose-200'}">
                    {log.result}
                  </span>
                </td>
                <td class="px-5 py-3.5 text-slate-600">{log.roleName || '-'}</td>
                <td class="px-5 py-3.5 text-slate-500 truncate max-w-[140px]">{log.purpose || '-'}</td>
                <td class="px-4 py-3.5 text-center">
                  {#if log.eventType === 'unregistered_renewal'}
                    <span class="inline-block px-2.5 py-0.5 rounded text-[11px] font-bold {log.isRegistered ? 'bg-sky-100 text-sky-800 border border-sky-300' : 'bg-amber-100 text-amber-800 border border-amber-300'}">
                      {log.typeLabel}
                    </span>
                  {:else if log.eventType === 'unregistered'}
                    <span class="inline-block px-2.5 py-0.5 rounded text-[11px] font-bold {log.isRegistered ? 'bg-sky-100 text-sky-800 border border-sky-300' : 'bg-rose-100 text-rose-700 border border-rose-300'}">
                      {log.typeLabel}
                    </span>
                  {:else}
                    <span class="inline-block px-2.5 py-0.5 rounded text-xs font-bold {log.eventType === 'entry' ? 'bg-blue-100 text-blue-700' : (log.eventType === 'force_exit' ? 'bg-amber-100 text-amber-700' : 'bg-rose-100 text-rose-700')}">
                      {log.typeLabel}
                    </span>
                  {/if}
                </td>
                <td class="px-5 py-3.5 text-slate-500">{log.durationText || '-'}</td>
                <td class="px-4 py-3.5 text-center whitespace-nowrap">
                  {#if log.eventType === 'unregistered_renewal'}
                    {#if log.isRegistered}
                      <span class="inline-flex items-center gap-1 text-[11px] font-bold text-emerald-600 bg-emerald-50 px-2 py-0.5 rounded border border-emerald-200">
                        ✓ 登録済
                      </span>
                    {:else}
                      <button
                        on:click={() => onRenewalUser(log)}
                        class="inline-flex items-center gap-1 px-2.5 py-1 bg-amber-500 hover:bg-amber-600 text-white rounded-lg text-[11px] font-bold shadow-xs transition"
                        title="前年度の登録情報を引き継いで今年度登録を更新"
                      >
                        <span>🔄</span> 登録更新
                      </button>
                    {/if}
                  {:else if log.eventType === 'unregistered'}
                    {#if log.isRegistered}
                      <span class="inline-flex items-center gap-1 text-[11px] font-bold text-emerald-600 bg-emerald-50 px-2 py-0.5 rounded border border-emerald-200">
                        ✓ 登録済
                      </span>
                    {:else}
                      <button
                        on:click={() => onRegisterUnregistered(log)}
                        class="inline-flex items-center gap-1 px-2.5 py-1 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-[11px] font-bold shadow-xs transition"
                        title="読み取られたカードIDで新規登録"
                      >
                        <span>＋</span> 新規登録
                      </button>
                    {/if}
                  {:else}
                    <span class="text-slate-400">-</span>
                  {/if}
                </td>
              </tr>
            {/each}
          {/if}
        </tbody>
      </table>
    </div>

    <!-- ページネーション -->
    <div class="px-4 sm:px-6 py-4 border-t border-slate-100">
      <div class="grid grid-cols-1 md:grid-cols-[1fr_auto_1fr] items-center gap-3">
        <div class="text-xs text-slate-500 text-center md:text-left">
          全 {filteredLogs.length} 件中 {filteredLogs.length > 0 ? (logCurrentPage - 1) * LOGS_PER_PAGE + 1 : 0} ～ {Math.min(logCurrentPage * LOGS_PER_PAGE, filteredLogs.length)} 件を表示
        </div>
        {#if totalLogPages > 1}
          <div class="min-w-0 max-w-full justify-self-center flex flex-wrap items-center justify-center gap-3">
            <nav aria-label="入退室履歴のページ" class="flex max-w-full flex-wrap items-center justify-center gap-1 text-xs">
              <button 
                disabled={logCurrentPage === 1}
                aria-label="前のページ"
                on:click={() => setLogPage(logCurrentPage - 1)}
                class="h-8 w-8 rounded border border-slate-300 hover:bg-slate-50 disabled:text-slate-300 disabled:cursor-not-allowed"
              >‹</button>

              {#each logPageItems as item (item.key)}
                {#if item.type === 'ellipsis'}
                  <span aria-hidden="true" class="w-5 text-center text-slate-500">…</span>
                {:else}
                  <button
                    aria-current={logCurrentPage === item.page ? 'page' : undefined}
                    on:click={() => setLogPage(item.page)}
                    class="h-8 min-w-8 rounded px-1.5 font-bold {logCurrentPage === item.page ? 'bg-blue-600 text-white' : 'border border-slate-200 hover:bg-slate-50 text-slate-700'}"
                  >
                    {item.page}
                  </button>
                {/if}
              {/each}

              <button 
                disabled={logCurrentPage === totalLogPages}
                aria-label="次のページ"
                on:click={() => setLogPage(logCurrentPage + 1)}
                class="h-8 w-8 rounded border border-slate-300 hover:bg-slate-50 disabled:text-slate-300 disabled:cursor-not-allowed"
              >›</button>
            </nav>
            <form class="flex items-center justify-center gap-2 text-xs" on:submit|preventDefault={() => setLogPage(logPageInput)}>
              <label for="log-page-input" class="text-slate-600">ページへ移動</label>
              <input
                id="log-page-input"
                type="number"
                min="1"
                max={totalLogPages}
                bind:value={logPageInput}
                on:change={(event) => setLogPage(event.currentTarget.value)}
                class="h-8 w-16 rounded border border-slate-300 text-center text-xs focus:outline-none focus:ring-2 focus:ring-blue-500/20"
              />
              <span class="text-slate-500">/ {totalLogPages}</span>
              <button type="submit" class="h-8 rounded border border-slate-300 px-2 font-semibold text-slate-700 hover:bg-slate-50">移動</button>
            </form>
          </div>
        {/if}
        <div class="hidden md:block"></div>
      </div>
    </div>
  </div>
</div>

