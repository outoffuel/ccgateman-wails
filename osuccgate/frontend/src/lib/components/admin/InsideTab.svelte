<script>
  import { onDestroy } from 'svelte';
  import { sortList } from '../../utils/sorting.js';
  import { formatLogTime } from '../../utils/date.js';

  export let insideUsers = [];
  export let onOpenAdvanced = () => {};

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
</script>

<div class="space-y-6">
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
    <div>
      <h2 class="text-xl font-bold text-slate-800">在室中メンバー</h2>
      <p class="text-xs text-slate-500 mt-0.5">現在、施設内にいる利用者のリアルタイム状況です</p>
    </div>
    <button 
      on:click={onOpenAdvanced}
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
              <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('userName')}>
                <span>氏名 {insideSortKey === 'userName' ? (insideSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'userName')}></div>
            </th>

            <th style="width: {insideWidths.roleName}px;" class="px-6 py-3.5 relative group select-none">
              <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('roleName')}>
                <span>区分 {insideSortKey === 'roleName' ? (insideSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'roleName')}></div>
            </th>

            <th style="width: {insideWidths.studentNo}px;" class="px-6 py-3.5 relative group select-none">
              <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('studentNo')}>
                <span>学籍番号/職員番号 {insideSortKey === 'studentNo' ? (insideSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'studentNo')}></div>
            </th>

            <th style="width: {insideWidths.lastEventTime}px;" class="px-6 py-3.5 relative group select-none">
              <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('lastEventTime')}>
                <span>入室時刻 {insideSortKey === 'lastEventTime' ? (insideSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'lastEventTime')}></div>
            </th>

            <th style="width: {insideWidths.stayDuration}px;" class="px-6 py-3.5 relative group select-none">
              <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('stayDuration')}>
                <span>滞在時間 {insideSortKey === 'stayDuration' ? (insideSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'stayDuration')}></div>
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

