<script>
  import { onDestroy } from 'svelte';
  import { sortList } from '../../utils/sorting.js';
  import { clampPage, getPageItems } from '../../utils/pagination.js';
  import { formatLogTime } from '../../utils/date.js';
  import { SaveCSVContent } from '../../adminApi.js';

  export let logs = [];
  export let processedLogs = null;
  export let availableFiscalYears = [2026];
  export let selectedFiscalYear = 2026;
  export let onSelectFiscalYear = () => {};
  export let onRenewalUser = () => {};
  export let onRegisterUnregistered = () => {};

  // 列幅リサイズ
  let logsWidths = { timestamp: 155, displayId: 105, userName: 120, result: 75, roleName: 95, purpose: 120, eventType: 100, duration: 95, action: 110 };
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
  let logsPerPage = 20;

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

  $: totalLogPages = Math.ceil(filteredLogs.length / logsPerPage) || 1;
  $: logPageItems = getPageItems(totalLogPages, logCurrentPage);
  $: if (logCurrentPage > totalLogPages) setLogPage(totalLogPages);
  $: paginatedLogs = filteredLogs.slice((logCurrentPage - 1) * logsPerPage, logCurrentPage * logsPerPage);

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

  async function exportLogsCSV() {
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

    const defaultFilename = `access_history_${new Date().toISOString().slice(0, 10)}.csv`;
    const csvContent = "\uFEFF" + [header.join(","), ...rows.map(r => r.join(","))].join("\r\n");

    // Wails ネイティブ環境ではOS標準のファイル保存ダイアログを使用
    if (typeof window !== 'undefined' && window.go?.main?.App) {
      try {
        const savedPath = await SaveCSVContent(defaultFilename, csvContent);
        if (savedPath) {
          alert(`CSVファイルを保存しました: ${savedPath}`);
        }
      } catch (err) {
        alert("CSV出力に失敗しました: " + (err?.message || err));
      }
      return;
    }

    // リモートブラウザ閲覧時（Webモード）はBlobダウンロードへフォールバック
    const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.setAttribute("href", url);
    link.setAttribute("download", defaultFilename);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  }
</script>

<div class="space-y-6">
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
    <div>
      <h2 class="text-xl font-bold text-m3-on-surface tracking-tight">入退室履歴</h2>
      <p class="text-xs text-m3-on-surface-variant mt-0.5">クリエイトセンターへのすべての入退室ログを検索・一覧確認できます（列ヘッダーでソート・リサイズ可能）</p>
    </div>
    <div class="flex items-center gap-3">
      <!-- 表示年度プルダウン (M3 Pill Select) -->
      <div class="flex items-center gap-2 bg-m3-surface-container-lowest border border-m3-outline-variant rounded-full px-4 py-2 shadow-m3-1">
        <label for="logs-fiscal-year-select" class="text-xs font-bold text-m3-on-surface-variant whitespace-nowrap">表示年度:</label>
        <select 
          id="logs-fiscal-year-select" 
          value={selectedFiscalYear} 
          on:change={(e) => onSelectFiscalYear(Number(e.target.value))}
          class="text-xs font-bold text-m3-primary bg-transparent focus:outline-none cursor-pointer"
        >
          {#each availableFiscalYears as fy}
            <option value={fy}>{fy}年度</option>
          {/each}
        </select>
      </div>

      <button 
        on:click={exportLogsCSV}
        class="inline-flex items-center justify-center gap-2 px-5 py-2.5 bg-m3-tertiary text-m3-on-tertiary hover:brightness-105 rounded-full text-xs font-bold transition shadow-m3-1"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
        </svg>
        <span>CSV出力</span>
      </button>
    </div>
  </div>

  <!-- 検索・絞り込みエリア (M3 Card + M3 Search Bar + Outlined Inputs) -->
  <div class="m3-card-elevated p-5">
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-12 gap-4 items-end">
      <div class="lg:col-span-4">
        <label for="filter-date-start" class="block text-xs font-bold text-m3-on-surface-variant mb-1.5 ml-1">日付指定 (開始日 ～ 終了日)</label>
        <div class="flex items-center gap-2">
          <input id="filter-date-start" type="date" bind:value={filterDateStart} class="m3-input-outlined py-2" />
          <span class="text-m3-on-surface-variant text-xs">～</span>
          <input id="filter-date-end" type="date" bind:value={filterDateEnd} class="m3-input-outlined py-2" />
        </div>
      </div>

      <div class="lg:col-span-2">
        <label for="filter-type-select" class="block text-xs font-bold text-m3-on-surface-variant mb-1.5 ml-1">種類</label>
        <select id="filter-type-select" bind:value={filterType} class="m3-input-outlined py-2">
          <option value="all">すべて</option>
          <option value="entry">入室</option>
          <option value="exit">退室</option>
        </select>
      </div>

      <div class="lg:col-span-2">
        <label for="filter-role-select" class="block text-xs font-bold text-m3-on-surface-variant mb-1.5 ml-1">区分</label>
        <select id="filter-role-select" bind:value={filterRole} class="m3-input-outlined py-2">
          <option value="all">すべて</option>
          <option value="学生">学生</option>
          <option value="教職員">教職員</option>
          <option value="学生スタッフ">学生スタッフ</option>
        </select>
      </div>

      <!-- M3 Search Bar (Pill) -->
      <div class="lg:col-span-2">
        <label for="filter-keyword-input" class="block text-xs font-bold text-m3-on-surface-variant mb-1.5 ml-1">キーワード検索</label>
        <div class="relative">
          <span class="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none text-m3-on-surface-variant">
            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
            </svg>
          </span>
          <input 
            id="filter-keyword-input" 
            type="text" 
            bind:value={filterKeyword} 
            placeholder="氏名またはID" 
            class="w-full text-xs pl-9 pr-3.5 py-2 rounded-full bg-m3-surface-container-high text-m3-on-surface placeholder:text-m3-on-surface-variant/70 focus:outline-none focus:ring-2 focus:ring-m3-primary/30 transition" 
          />
        </div>
      </div>

      <div class="lg:col-span-2 flex items-center gap-2">
        <button 
          on:click={() => setLogPage(1)}
          class="flex-1 m3-btn-filled py-2 px-4"
        >
          検索
        </button>
        <button 
          on:click={clearLogFilters}
          class="m3-btn-outlined py-2 px-3.5"
        >
          クリア
        </button>
      </div>
    </div>
  </div>

  <!-- テーブル (M3 Data Table + ソート & 列リサイズ対応) -->
  <div class="m3-card-elevated overflow-hidden">
    <div class="px-6 py-4 border-b border-m3-outline-variant/40 flex flex-col sm:flex-row sm:items-center justify-between gap-3 bg-m3-surface-container-low/60">
      <div class="flex items-center gap-4">
        <span class="text-xs font-semibold text-m3-on-surface-variant">
          検索結果：<span class="font-bold text-m3-on-surface">{filteredLogs.length}</span> 件
          {#if filteredLogs.length > 0}
            <span class="text-m3-on-surface-variant/80 ml-1">
              ({(logCurrentPage - 1) * logsPerPage + 1}〜{Math.min(logCurrentPage * logsPerPage, filteredLogs.length)}件を表示)
            </span>
          {/if}
        </span>

        <!-- 表示件数選択 (M3 Pill Select) -->
        <div class="flex items-center gap-1.5 text-xs text-m3-on-surface-variant">
          <span>表示件数:</span>
          <select 
            bind:value={logsPerPage} 
            on:change={() => setLogPage(1)}
            class="bg-m3-surface-container-lowest border border-m3-outline-variant text-xs rounded-full px-3 py-1 focus:outline-none focus:border-m3-primary font-bold text-m3-on-surface cursor-pointer"
          >
            <option value={10}>10件</option>
            <option value={20}>20件</option>
            <option value={50}>50件</option>
            <option value={100}>100件</option>
          </select>
        </div>
      </div>
      <span class="text-xs text-m3-on-surface-variant/80">※列ヘッダーをクリックでソート、右端をドラッグで幅変更可能</span>
    </div>

    <div class="overflow-x-auto">
      <table class="w-full text-left text-xs border-collapse">
        <thead class="bg-m3-surface-container-low text-m3-on-surface-variant font-bold border-b border-m3-outline-variant/60">
          <tr>
            <!-- 1. 日時 -->
            <th style="width: {logsWidths.timestamp}px;" class="px-5 py-3.5 relative group select-none">
              <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('timestamp')}>
                <span>日時 {logsSortKey === 'timestamp' ? (logsSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-m3-primary/30" on:mousedown={(e) => startResize(e, 'timestamp')}></div>
            </th>

            <!-- 2. ID -->
            <th style="width: {logsWidths.displayId}px;" class="px-5 py-3.5 relative group select-none">
              <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('displayId')}>
                <span>ID {logsSortKey === 'displayId' ? (logsSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-m3-primary/30" on:mousedown={(e) => startResize(e, 'displayId')}></div>
            </th>

            <!-- 3. 氏名 -->
            <th style="width: {logsWidths.userName}px;" class="px-5 py-3.5 relative group select-none">
              <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('userName')}>
                <span>氏名 {logsSortKey === 'userName' ? (logsSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-m3-primary/30" on:mousedown={(e) => startResize(e, 'userName')}></div>
            </th>

            <!-- 4. 判定 -->
            <th style="width: {logsWidths.result}px;" class="px-4 py-3.5 text-center relative group select-none">
              <div class="flex items-center justify-center cursor-pointer" on:click={() => handleSort('result')}>
                <span>判定 {logsSortKey === 'result' ? (logsSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-m3-primary/30" on:mousedown={(e) => startResize(e, 'result')}></div>
            </th>

            <!-- 5. 区分 -->
            <th style="width: {logsWidths.roleName}px;" class="px-5 py-3.5 relative group select-none">
              <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('roleName')}>
                <span>区分 {logsSortKey === 'roleName' ? (logsSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-m3-primary/30" on:mousedown={(e) => startResize(e, 'roleName')}></div>
            </th>

            <!-- 6. 利用目的 -->
            <th style="width: {logsWidths.purpose}px;" class="px-5 py-3.5 relative group select-none">
              <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('purpose')}>
                <span>利用目的 {logsSortKey === 'purpose' ? (logsSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-m3-primary/30" on:mousedown={(e) => startResize(e, 'purpose')}></div>
            </th>

            <!-- 7. 入退室 -->
            <th style="width: {logsWidths.eventType}px;" class="px-4 py-3.5 text-center relative group select-none">
              <div class="flex items-center justify-center cursor-pointer" on:click={() => handleSort('eventType')}>
                <span>入退室 {logsSortKey === 'eventType' ? (logsSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-m3-primary/30" on:mousedown={(e) => startResize(e, 'eventType')}></div>
            </th>

            <!-- 8. 滞在時間 -->
            <th style="width: {logsWidths.duration}px;" class="px-5 py-3.5 relative group select-none">
              <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('durationSecond')}>
                <span>滞在時間 {logsSortKey === 'durationSecond' ? (logsSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-m3-primary/30" on:mousedown={(e) => startResize(e, 'duration')}></div>
            </th>

            <!-- 9. 操作 -->
            <th style="width: {logsWidths.action}px;" class="px-4 py-3.5 text-center relative group select-none">
              <div class="flex items-center justify-center">
                <span>操作</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-m3-primary/30" on:mousedown={(e) => startResize(e, 'action')}></div>
            </th>
          </tr>
        </thead>
        <tbody class="divide-y divide-m3-outline-variant/30">
          {#if paginatedLogs.length === 0}
            <tr>
              <td colspan="9" class="px-6 py-10 text-center text-m3-on-surface-variant font-medium">条件に一致する履歴はありません。</td>
            </tr>
          {:else}
            {#each paginatedLogs as log}
              <tr class="hover:bg-m3-surface-container-high transition-colors duration-150">
                <td class="px-5 py-3.5 font-mono text-m3-on-surface-variant">{log.formatted}</td>
                <td class="px-5 py-3.5 font-mono text-m3-on-surface-variant font-semibold">{log.displayId || '-'}</td>
                <td class="px-5 py-3.5 font-bold text-m3-on-surface">{log.userName || '未登録'}</td>
                <td class="px-4 py-3.5 text-center">
                  <span class="inline-block px-2.5 py-0.5 rounded-full text-[11px] font-bold {(log.result === '成功' || log.result === '登録済') ? 'bg-m3-tertiary-container text-m3-on-tertiary-container' : 'bg-m3-error-container text-m3-on-error-container'}">
                    {log.result}
                  </span>
                </td>
                <td class="px-5 py-3.5 text-m3-on-surface-variant">{log.roleName || '-'}</td>
                <td class="px-5 py-3.5 text-m3-on-surface-variant truncate max-w-[140px]">{log.purpose || '-'}</td>
                <td class="px-4 py-3.5 text-center">
                  {#if log.eventType === 'unregistered_renewal'}
                    <span class="inline-block px-3 py-0.5 rounded-full text-[11px] font-bold {log.isRegistered ? 'bg-m3-secondary-container text-m3-on-secondary-container' : 'bg-m3-warning-container text-m3-on-warning-container'}">
                      {log.typeLabel}
                    </span>
                  {:else if log.eventType === 'unregistered'}
                    <span class="inline-block px-3 py-0.5 rounded-full text-[11px] font-bold {log.isRegistered ? 'bg-m3-secondary-container text-m3-on-secondary-container' : 'bg-m3-error-container text-m3-on-error-container'}">
                      {log.typeLabel}
                    </span>
                  {:else}
                    <span class="inline-block px-3 py-0.5 rounded-full text-xs font-bold {log.eventType === 'entry' ? 'bg-m3-primary-container text-m3-on-primary-container' : (log.eventType === 'force_exit' ? 'bg-m3-warning-container text-m3-on-warning-container' : 'bg-m3-error-container text-m3-on-error-container')}">
                      {log.typeLabel}
                    </span>
                  {/if}
                </td>
                <td class="px-5 py-3.5 text-m3-on-surface-variant">{log.durationText || '-'}</td>
                <td class="px-4 py-3.5 text-center whitespace-nowrap">
                  {#if log.eventType === 'unregistered_renewal'}
                    {#if log.isRegistered}
                      <span class="inline-flex items-center gap-1 text-[11px] font-bold text-m3-on-tertiary-container bg-m3-tertiary-container px-2.5 py-0.5 rounded-full">
                        ✓ 登録済
                      </span>
                    {:else}
                      <button
                        on:click={() => onRenewalUser(log)}
                        class="inline-flex items-center gap-1 px-3 py-1 bg-m3-warning hover:brightness-105 text-m3-on-warning rounded-full text-[11px] font-bold shadow-m3-1 transition"
                        title="前年度の登録情報を引き継いで今年度登録を更新"
                      >
                        <span>🔄</span> 登録更新
                      </button>
                    {/if}
                  {:else if log.eventType === 'unregistered'}
                    {#if log.isRegistered}
                      <span class="inline-flex items-center gap-1 text-[11px] font-bold text-m3-on-tertiary-container bg-m3-tertiary-container px-2.5 py-0.5 rounded-full">
                        ✓ 登録済
                      </span>
                    {:else}
                      <button
                        on:click={() => onRegisterUnregistered(log)}
                        class="inline-flex items-center gap-1 px-3 py-1 bg-m3-primary hover:brightness-105 text-m3-on-primary rounded-full text-[11px] font-bold shadow-m3-1 transition"
                        title="読み取られたカードIDで新規登録"
                      >
                        <span>＋</span> 新規登録
                      </button>
                    {/if}
                  {:else}
                    <span class="text-m3-on-surface-variant/60">-</span>
                  {/if}
                </td>
              </tr>
            {/each}
          {/if}
        </tbody>
      </table>
    </div>

    <!-- M3 Navigation Controls (ページネーション) -->
    <div class="px-4 sm:px-6 py-4 border-t border-m3-outline-variant/40 bg-m3-surface-container-low/40">
      <div class="grid grid-cols-1 md:grid-cols-[1fr_auto_1fr] items-center gap-3">
        <div class="text-xs text-m3-on-surface-variant text-center md:text-left">
          全 {filteredLogs.length} 件中 {filteredLogs.length > 0 ? (logCurrentPage - 1) * logsPerPage + 1 : 0} ～ {Math.min(logCurrentPage * logsPerPage, filteredLogs.length)} 件を表示
        </div>
        {#if totalLogPages > 1}
          <div class="min-w-0 max-w-full justify-self-center flex flex-wrap items-center justify-center gap-3">
            <nav aria-label="入退室履歴のページ" class="flex max-w-full flex-wrap items-center justify-center gap-1 text-xs">
              <button 
                disabled={logCurrentPage === 1}
                aria-label="前のページ"
                on:click={() => setLogPage(logCurrentPage - 1)}
                class="h-8 w-8 rounded-full border border-m3-outline-variant hover:bg-m3-surface-container-high disabled:opacity-40 disabled:cursor-not-allowed font-bold"
              >‹</button>

              {#each logPageItems as item (item.key)}
                {#if item.type === 'ellipsis'}
                  <span aria-hidden="true" class="w-5 text-center text-m3-on-surface-variant">…</span>
                {:else}
                  <button
                    aria-current={logCurrentPage === item.page ? 'page' : undefined}
                    on:click={() => setLogPage(item.page)}
                    class="h-8 min-w-8 rounded-full px-2 font-bold transition-colors duration-150 {logCurrentPage === item.page ? 'bg-m3-primary text-m3-on-primary shadow-m3-1' : 'text-m3-on-surface-variant hover:bg-m3-surface-container-high'}"
                  >
                    {item.page}
                  </button>
                {/if}
              {/each}

              <button 
                disabled={logCurrentPage === totalLogPages}
                aria-label="次のページ"
                on:click={() => setLogPage(logCurrentPage + 1)}
                class="h-8 w-8 rounded-full border border-m3-outline-variant hover:bg-m3-surface-container-high disabled:opacity-40 disabled:cursor-not-allowed font-bold"
              >›</button>
            </nav>
            <form class="flex items-center justify-center gap-2 text-xs" on:submit|preventDefault={() => setLogPage(logPageInput)}>
              <label for="log-page-input" class="text-m3-on-surface-variant">ページへ移動</label>
              <input
                id="log-page-input"
                type="number"
                min="1"
                max={totalLogPages}
                bind:value={logPageInput}
                on:change={(event) => setLogPage(event.currentTarget.value)}
                class="h-8 w-16 rounded-full border border-m3-outline-variant bg-m3-surface-container-lowest text-center text-xs focus:outline-none focus:border-m3-primary"
              />
              <span class="text-m3-on-surface-variant">/ {totalLogPages}</span>
              <button type="submit" class="m3-btn-tonal py-1.5 px-3 text-xs">移動</button>
            </form>
          </div>
        {/if}
        <div class="hidden md:block"></div>
      </div>
    </div>
  </div>
</div>
