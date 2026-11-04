<script>
  import { onDestroy } from 'svelte';
  import { sortList } from '../../utils/sorting.js';
  import { formatLogTime } from '../../utils/date.js';

  export let insideUsers = [];
  export let onOpenAdvanced = () => {};

  // 検索フィルター
  let searchQuery = '';

  // 列幅リサイズ
  let insideWidths = { userName: 140, roleName: 110, studentNo: 130, lastEventTime: 160, stayDuration: 130 };
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
    startWidth = insideWidths[colKey];

    window.addEventListener('mousemove', handleResizeMove);
    window.addEventListener('mouseup', handleResizeEnd);
  }

  function handleResizeMove(e) {
    if (!isResizing) return;
    const delta = e.clientX - startX;
    const newWidth = Math.max(50, startWidth + delta);
    insideWidths = { ...insideWidths, [activeResizeCol]: newWidth };
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
  let insideSortKey = 'lastEventTime';
  let insideSortDir = 'desc';

  function handleSort(key) {
    insideSortDir = (insideSortKey === key && insideSortDir === 'asc') ? 'desc' : 'asc';
    insideSortKey = key;
  }

  $: filteredInsideUsers = sortList(insideUsers, insideSortKey, insideSortDir).filter(u => {
    if (!searchQuery.trim()) return true;
    const q = searchQuery.trim().toLowerCase();
    const mName = (u.userName || '').toLowerCase().includes(q);
    const mId = (u.studentNo || u.cardId || '').toLowerCase().includes(q);
    const mRole = (u.roleName || '').toLowerCase().includes(q);
    return mName || mId || mRole;
  });
</script>

<div class="space-y-6">
  <!-- ヘッダー & アクション -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
    <div>
      <h2 class="text-xl font-bold text-m3-on-surface tracking-tight">在室中メンバー</h2>
      <p class="text-xs text-m3-on-surface-variant mt-0.5">現在、施設内にいる利用者のリアルタイム状況です</p>
    </div>
    <button 
      on:click={onOpenAdvanced}
      class="inline-flex items-center justify-center gap-2 px-5 py-2.5 rounded-full bg-m3-warning-container text-m3-on-warning-container hover:brightness-95 text-xs font-bold transition shadow-m3-1"
    >
      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"/>
      </svg>
      <span>一括強制退室 / 管理操作</span>
    </button>
  </div>

  <!-- M3 Search Bar & 在室者ステータスピル -->
  <div class="m3-card-elevated p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
    <div class="relative w-full sm:w-96">
      <span class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none text-m3-on-surface-variant">
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
        </svg>
      </span>
      <input 
        type="text" 
        bind:value={searchQuery}
        placeholder="在室メンバーを氏名・学籍番号・区分で検索..." 
        class="w-full text-xs pl-10 pr-4 py-2.5 rounded-full bg-m3-surface-container-high text-m3-on-surface placeholder:text-m3-on-surface-variant/70 focus:outline-none focus:ring-2 focus:ring-m3-primary/30 transition"
      />
    </div>

    <div class="flex items-center gap-2 px-4 py-1.5 rounded-full bg-m3-tertiary-container/70 text-m3-on-tertiary-container text-xs font-bold">
      <span class="w-2 h-2 rounded-full bg-m3-tertiary animate-pulse"></span>
      <span>現在の在室者数：{insideUsers.length} 名</span>
    </div>
  </div>

  <!-- M3 Data Table -->
  <div class="m3-card-elevated overflow-hidden">
    <div class="overflow-x-auto">
      <table class="w-full text-left text-xs border-collapse">
        <thead class="bg-m3-surface-container-low text-m3-on-surface-variant font-bold border-b border-m3-outline-variant/60">
          <tr>
            <th style="width: {insideWidths.userName}px;" class="px-6 py-4 relative group select-none">
              <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('userName')}>
                <span>氏名 {insideSortKey === 'userName' ? (insideSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-m3-primary/30" on:mousedown={(e) => startResize(e, 'userName')}></div>
            </th>

            <th style="width: {insideWidths.roleName}px;" class="px-6 py-4 relative group select-none">
              <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('roleName')}>
                <span>区分 {insideSortKey === 'roleName' ? (insideSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-m3-primary/30" on:mousedown={(e) => startResize(e, 'roleName')}></div>
            </th>

            <th style="width: {insideWidths.studentNo}px;" class="px-6 py-4 relative group select-none">
              <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('studentNo')}>
                <span>学籍番号/職員番号 {insideSortKey === 'studentNo' ? (insideSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-m3-primary/30" on:mousedown={(e) => startResize(e, 'studentNo')}></div>
            </th>

            <th style="width: {insideWidths.lastEventTime}px;" class="px-6 py-4 relative group select-none">
              <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('lastEventTime')}>
                <span>入室時刻 {insideSortKey === 'lastEventTime' ? (insideSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-m3-primary/30" on:mousedown={(e) => startResize(e, 'lastEventTime')}></div>
            </th>

            <th style="width: {insideWidths.stayDuration}px;" class="px-6 py-4 relative group select-none">
              <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('stayDuration')}>
                <span>滞在時間 {insideSortKey === 'stayDuration' ? (insideSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-m3-primary/30" on:mousedown={(e) => startResize(e, 'stayDuration')}></div>
            </th>
          </tr>
        </thead>
        <tbody class="divide-y divide-m3-outline-variant/30">
          {#if filteredInsideUsers.length === 0}
            <tr>
              <td colspan="5" class="px-6 py-12 text-center text-m3-on-surface-variant font-medium">
                {insideUsers.length === 0 ? '現在、在室中のメンバーはいません。' : '検索条件に一致する在室メンバーはいません。'}
              </td>
            </tr>
          {:else}
            {#each filteredInsideUsers as user}
              <tr class="hover:bg-m3-surface-container-high transition-colors duration-150">
                <td class="px-6 py-4 font-bold text-m3-on-surface">{user.userName}</td>
                <td class="px-6 py-4">
                  <span class="inline-block px-3 py-0.5 rounded-full text-[11px] font-bold bg-m3-secondary-container text-m3-on-secondary-container">
                    {user.roleName || '-'}
                  </span>
                </td>
                <td class="px-6 py-4 font-mono text-m3-on-surface-variant">{user.studentNo || user.cardId}</td>
                <td class="px-6 py-4 font-mono text-m3-on-surface-variant">{formatLogTime(user.lastEventTime)}</td>
                <td class="px-6 py-4">
                  <span class="inline-flex items-center gap-1.5 px-3 py-1 rounded-full bg-m3-primary-container/70 text-m3-on-primary-container font-bold text-xs">
                    <span class="w-1.5 h-1.5 rounded-full bg-m3-primary"></span>
                    {user.stayDuration || '入室中'}
                  </span>
                </td>
              </tr>
            {/each}
          {/if}
        </tbody>
      </table>
    </div>
  </div>
</div>
