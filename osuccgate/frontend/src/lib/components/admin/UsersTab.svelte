<script>
  import { onDestroy } from 'svelte';
  import { sortList } from '../../utils/sorting.js';
  import { clampPage, getPageItems } from '../../utils/pagination.js';

  export let users = [];
  export let availableFiscalYears = [2026];
  export let selectedFiscalYear = 2026;
  export let currentFiscalYear = 2026;
  export let onSelectFiscalYear = () => {};
  export let onCreateUser = () => {};
  export let onEditUser = () => {};
  export let onDeleteUser = () => {};

  // 列幅リサイズ
  let usersWidths = { adminNo: 70, studentNo: 120, name: 120, furigana: 120, gender: 65, roleName: 100, contact: 160, purpose: 140, actions: 90 };
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
    startWidth = usersWidths[colKey];

    window.addEventListener('mousemove', handleResizeMove);
    window.addEventListener('mouseup', handleResizeEnd);
  }

  function handleResizeMove(e) {
    if (!isResizing) return;
    const delta = e.clientX - startX;
    const newWidth = Math.max(50, startWidth + delta);
    usersWidths = { ...usersWidths, [activeResizeCol]: newWidth };
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
  let usersSortKey = 'roleCode';
  let usersSortDir = 'asc';

  function handleSort(key) {
    usersSortDir = (usersSortKey === key && usersSortDir === 'asc') ? 'desc' : 'asc';
    usersSortKey = key;
  }

  // 検索・絞り込み・ページネーション・個人情報保護
  let userSearch = "";
  let userRoleFilter = "all"; // 'all' | '学生' | '教職員' | '学生スタッフ'
  let userCurrentPage = 1;
  let userPageInput = 1;
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
  $: userPageItems = getPageItems(totalUserPages, userCurrentPage);
  $: if (userCurrentPage > totalUserPages) setUserPage(totalUserPages);
  $: paginatedUsers = filteredUsers.slice((userCurrentPage - 1) * usersPerPage, userCurrentPage * usersPerPage);

  function setUserPage(page) {
    userCurrentPage = clampPage(page, totalUserPages);
    userPageInput = userCurrentPage;
  }

  // 検索・絞り込み時にページネーションを1に戻す
  $: if (userSearch || userRoleFilter) {
    setUserPage(1);
  }
</script>

<div class="space-y-6">
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
    <div>
      <h2 class="text-xl font-bold text-slate-800">ユーザー管理</h2>
      <p class="text-xs text-slate-500 mt-0.5">クリエイトセンターへの入室を許可する利用者を登録・管理します（列幅調整・ソート対応）</p>
    </div>
    <div class="flex items-center gap-3">
      <!-- 表示年度プルダウン (要件4) -->
      <div class="flex items-center gap-1.5 bg-white border border-slate-300 rounded-xl px-3 py-1.5 shadow-xs">
        <label for="users-fiscal-year-select" class="text-xs font-bold text-slate-600 whitespace-nowrap">表示年度:</label>
        <select 
          id="users-fiscal-year-select" 
          value={selectedFiscalYear} 
          on:change={(e) => onSelectFiscalYear(Number(e.target.value))}
          class="text-xs font-bold text-blue-700 bg-transparent focus:outline-none cursor-pointer"
        >
          {#each availableFiscalYears as fy}
            <option value={fy}>{fy}年度</option>
          {/each}
        </select>
      </div>

      <!-- 目立つ青色の ＋ 新規登録 ボタン -->
      <button 
        on:click={onCreateUser}
        class="px-5 py-2.5 bg-blue-600 hover:bg-blue-700 text-white rounded-xl text-xs font-bold transition shadow-sm flex items-center justify-center gap-1.5"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"/>
        </svg>
        新規登録
      </button>
    </div>
  </div>

  {#if selectedFiscalYear !== currentFiscalYear}
    <div class="px-4 py-2.5 rounded-xl bg-amber-50 border border-amber-200 text-xs text-amber-800 flex items-center justify-between">
      <div class="flex items-center gap-2">
        <span class="font-bold">⚠️ アーカイブ表示中：</span>
        <span>現在は過去の「{selectedFiscalYear}年度」の登録者マスターを表示しています。</span>
      </div>
      <button 
        on:click={() => onSelectFiscalYear(currentFiscalYear)}
        class="text-xs font-bold text-amber-900 hover:underline"
      >
        今年度（{currentFiscalYear}年度）に戻る →
      </button>
    </div>
  {/if}

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
              <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('adminNo')}>
                <span>No {usersSortKey === 'adminNo' ? (usersSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'adminNo')}></div>
            </th>

            <!-- 学籍番号/職員番号 -->
            <th style="width: {usersWidths.studentNo}px;" class="px-5 py-3.5 relative group select-none">
              <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('studentNo')}>
                <span>学籍/職員番号 {usersSortKey === 'studentNo' ? (usersSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'studentNo')}></div>
            </th>

            <!-- 氏名 -->
            <th style="width: {usersWidths.name}px;" class="px-5 py-3.5 relative group select-none">
              <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('name')}>
                <span>氏名 {usersSortKey === 'name' ? (usersSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'name')}></div>
            </th>

            <!-- フリガナ -->
            <th style="width: {usersWidths.furigana}px;" class="px-5 py-3.5 relative group select-none">
              <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('furigana')}>
                <span>フリガナ {usersSortKey === 'furigana' ? (usersSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'furigana')}></div>
            </th>

            <!-- 性別 -->
            <th style="width: {usersWidths.gender}px;" class="px-4 py-3.5 text-center relative group select-none">
              <div class="flex items-center justify-center cursor-pointer" on:click={() => handleSort('gender')}>
                <span>性別 {usersSortKey === 'gender' ? (usersSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'gender')}></div>
            </th>

            <!-- 区分 -->
            <th style="width: {usersWidths.roleName}px;" class="px-5 py-3.5 relative group select-none">
              <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('roleCode')}>
                <span>区分 {usersSortKey === 'roleCode' ? (usersSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'roleName')}></div>
            </th>

            <!-- 連絡先 (個人情報保護対応) -->
            <th style="width: {usersWidths.contact}px;" class="px-5 py-3.5 relative group select-none">
              <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('contact')}>
                <span>連絡先 {usersSortKey === 'contact' ? (usersSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'contact')}></div>
            </th>

            <!-- 利用目的 -->
            <th style="width: {usersWidths.purpose}px;" class="px-5 py-3.5 relative group select-none">
              <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('purpose')}>
                <span>利用目的 {usersSortKey === 'purpose' ? (usersSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-blue-400/40" on:mousedown={(e) => startResize(e, 'purpose')}></div>
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
                    on:click={() => onEditUser(user)}
                    class="px-2 py-1 text-xs font-bold text-blue-600 hover:text-blue-800 hover:bg-blue-50 rounded transition"
                  >編集</button>
                  <button 
                    on:click={() => onDeleteUser(user)}
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
      <div class="px-4 sm:px-6 py-3.5 border-t border-slate-100 bg-slate-50/30">
        <div class="grid grid-cols-1 md:grid-cols-[1fr_auto_1fr] items-center gap-3">
          <span class="text-xs text-slate-500 font-medium text-center md:text-left">
            全 {filteredUsers.length} 人 ({userCurrentPage} / {totalUserPages} ページ)
          </span>
          <div class="min-w-0 max-w-full justify-self-center flex flex-wrap items-center justify-center gap-3">
            <nav aria-label="ユーザー一覧のページ" class="flex max-w-full flex-wrap items-center justify-center gap-1 text-xs">
              <button
                disabled={userCurrentPage === 1}
                aria-label="前のページ"
                on:click={() => setUserPage(userCurrentPage - 1)}
                class="h-8 w-8 rounded border border-slate-300 hover:bg-slate-50 disabled:text-slate-300 disabled:cursor-not-allowed"
              >‹</button>

              {#each userPageItems as item (item.key)}
                {#if item.type === 'ellipsis'}
                  <span aria-hidden="true" class="w-5 text-center text-slate-500">…</span>
                {:else}
                  <button
                    aria-current={userCurrentPage === item.page ? 'page' : undefined}
                    on:click={() => setUserPage(item.page)}
                    class="h-8 min-w-8 rounded px-1.5 font-bold {userCurrentPage === item.page ? 'bg-blue-600 text-white' : 'border border-slate-200 hover:bg-slate-50 text-slate-700'}"
                  >
                    {item.page}
                  </button>
                {/if}
              {/each}

              <button
                disabled={userCurrentPage === totalUserPages}
                aria-label="次のページ"
                on:click={() => setUserPage(userCurrentPage + 1)}
                class="h-8 w-8 rounded border border-slate-300 hover:bg-slate-50 disabled:text-slate-300 disabled:cursor-not-allowed"
              >›</button>
            </nav>
            <form class="flex items-center justify-center gap-2 text-xs" on:submit|preventDefault={() => setUserPage(userPageInput)}>
              <label for="user-page-input" class="text-slate-600">ページへ移動</label>
              <input
                id="user-page-input"
                type="number"
                min="1"
                max={totalUserPages}
                bind:value={userPageInput}
                on:change={(event) => setUserPage(event.currentTarget.value)}
                class="h-8 w-16 rounded border border-slate-300 text-center text-xs focus:outline-none focus:ring-2 focus:ring-blue-500/20"
              />
              <span class="text-slate-500">/ {totalUserPages}</span>
              <button type="submit" class="h-8 rounded border border-slate-300 px-2 font-semibold text-slate-700 hover:bg-slate-50">移動</button>
            </form>
          </div>
          <div class="hidden md:block"></div>
        </div>
      </div>
    {/if}
  </div>
</div>

