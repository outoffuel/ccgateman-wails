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
  let usersWidths = { adminNo: 70, studentNo: 120, name: 120, furigana: 120, gender: 65, roleName: 100, contact: 160, purpose: 140, actions: 100 };
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
      <h2 class="text-xl font-bold text-m3-on-surface tracking-tight">ユーザー管理</h2>
      <p class="text-xs text-m3-on-surface-variant mt-0.5">クリエイトセンターへの入室を許可する利用者を登録・管理します（列幅調整・ソート対応）</p>
    </div>
    <div class="flex items-center gap-3">
      <!-- 表示年度プルダウン (M3 Pill Select) -->
      <div class="flex items-center gap-2 bg-m3-surface-container-lowest border border-m3-outline-variant rounded-full px-4 py-2 shadow-m3-1">
        <label for="users-fiscal-year-select" class="text-xs font-bold text-m3-on-surface-variant whitespace-nowrap">表示年度:</label>
        <select 
          id="users-fiscal-year-select" 
          value={selectedFiscalYear} 
          on:change={(e) => onSelectFiscalYear(Number(e.target.value))}
          class="text-xs font-bold text-m3-primary bg-transparent focus:outline-none cursor-pointer"
        >
          {#each availableFiscalYears as fy}
            <option value={fy}>{fy}年度</option>
          {/each}
        </select>
      </div>

      <!-- M3 Filled Pill Button: ＋ 新規登録 -->
      <button 
        on:click={onCreateUser}
        class="m3-btn-filled py-2.5 px-5"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"/>
        </svg>
        <span>新規登録</span>
      </button>
    </div>
  </div>

  {#if selectedFiscalYear !== currentFiscalYear}
    <div class="px-5 py-3 rounded-2xl bg-m3-warning-container text-m3-on-warning-container text-xs flex items-center justify-between">
      <div class="flex items-center gap-2">
        <span class="font-bold">⚠️ アーカイブ表示中：</span>
        <span>現在は過去の「{selectedFiscalYear}年度」の登録者マスターを表示しています。</span>
      </div>
      <button 
        on:click={() => onSelectFiscalYear(currentFiscalYear)}
        class="text-xs font-bold underline hover:opacity-80"
      >
        今年度（{currentFiscalYear}年度）に戻る →
      </button>
    </div>
  {/if}

  <!-- M3 Search Bar & M3 Segmented Filter Buttons -->
  <div class="m3-card-elevated p-5 flex flex-col md:flex-row md:items-center justify-between gap-4">
    <!-- M3 Search Bar -->
    <div class="relative w-full md:w-88">
      <span class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none text-m3-on-surface-variant">
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
        </svg>
      </span>
      <input 
        type="text" 
        bind:value={userSearch}
        placeholder="氏名、フリガナ、学籍番号で検索..." 
        class="w-full text-xs pl-10 pr-4 py-2.5 rounded-full bg-m3-surface-container-high text-m3-on-surface placeholder:text-m3-on-surface-variant/70 focus:outline-none focus:ring-2 focus:ring-m3-primary/30 transition"
      />
    </div>

    <!-- M3 Segmented Filter Buttons -->
    <div class="flex flex-wrap items-center gap-2.5">
      <span class="text-xs font-bold text-m3-on-surface-variant">区分絞り込み:</span>
      <div class="inline-flex rounded-full border border-m3-outline/60 overflow-hidden text-xs font-bold divide-x divide-m3-outline/50 bg-m3-surface-container-lowest">
        <button 
          on:click={() => userRoleFilter = 'all'} 
          class="inline-flex items-center gap-1 px-4 py-1.5 transition-colors duration-150 {userRoleFilter === 'all' ? 'bg-m3-secondary-container text-m3-on-secondary-container' : 'text-m3-on-surface-variant hover:bg-m3-surface-container-high'}"
        >
          {#if userRoleFilter === 'all'}
            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7"/></svg>
          {/if}
          <span>すべて</span>
        </button>
        <button 
          on:click={() => userRoleFilter = '学生'} 
          class="inline-flex items-center gap-1 px-4 py-1.5 transition-colors duration-150 {userRoleFilter === '学生' ? 'bg-m3-secondary-container text-m3-on-secondary-container' : 'text-m3-on-surface-variant hover:bg-m3-surface-container-high'}"
        >
          {#if userRoleFilter === '学生'}
            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7"/></svg>
          {/if}
          <span>学生</span>
        </button>
        <button 
          on:click={() => userRoleFilter = '教職員'} 
          class="inline-flex items-center gap-1 px-4 py-1.5 transition-colors duration-150 {userRoleFilter === '教職員' ? 'bg-m3-secondary-container text-m3-on-secondary-container' : 'text-m3-on-surface-variant hover:bg-m3-surface-container-high'}"
        >
          {#if userRoleFilter === '教職員'}
            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7"/></svg>
          {/if}
          <span>教職員</span>
        </button>
        <button 
          on:click={() => userRoleFilter = '学生スタッフ'} 
          class="inline-flex items-center gap-1 px-4 py-1.5 transition-colors duration-150 {userRoleFilter === '学生スタッフ' ? 'bg-m3-secondary-container text-m3-on-secondary-container' : 'text-m3-on-surface-variant hover:bg-m3-surface-container-high'}"
        >
          {#if userRoleFilter === '学生スタッフ'}
            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7"/></svg>
          {/if}
          <span>スタッフ</span>
        </button>
      </div>
    </div>
  </div>

  <!-- M3 Data Table (ソート & 列リサイズ対応 & ページネーション & 個人情報保護) -->
  <div class="m3-card-elevated overflow-hidden">
    <!-- テーブル上部ステータスバー -->
    <div class="px-6 py-3.5 border-b border-m3-outline-variant/40 flex flex-wrap items-center justify-between gap-3 bg-m3-surface-container-low/60">
      <div class="flex items-center gap-4">
        <span class="text-xs font-semibold text-m3-on-surface-variant">
          登録者：<span class="font-bold text-m3-on-surface">{filteredUsers.length}</span> 人
          {#if filteredUsers.length > 0}
            <span class="text-m3-on-surface-variant/80 ml-1">
              ({(userCurrentPage - 1) * usersPerPage + 1}〜{Math.min(userCurrentPage * usersPerPage, filteredUsers.length)}件を表示)
            </span>
          {/if}
        </span>

        <!-- 表示件数選択 -->
        <div class="flex items-center gap-1.5 text-xs text-m3-on-surface-variant">
          <span>表示件数:</span>
          <select 
            bind:value={usersPerPage} 
            class="bg-m3-surface-container-lowest border border-m3-outline-variant text-xs rounded-full px-3 py-1 focus:outline-none focus:border-m3-primary font-bold text-m3-on-surface"
          >
            <option value={25}>25件</option>
            <option value={50}>50件</option>
            <option value={100}>100件</option>
          </select>
        </div>
      </div>

      <div class="flex items-center gap-3">
        <!-- 個人情報一括表示 / 非表示トグルボタン (M3 Assist Chip) -->
        <button 
          type="button"
          on:click={toggleShowAllContacts}
          class="inline-flex items-center gap-1.5 px-3.5 py-1.5 text-xs font-bold rounded-full border transition {showAllContacts ? 'bg-m3-warning-container text-m3-on-warning-container border-m3-warning/30' : 'bg-m3-surface-container-highest text-m3-on-surface-variant border-m3-outline-variant/60 hover:bg-m3-secondary-container/50'}"
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
        <thead class="bg-m3-surface-container-low text-m3-on-surface-variant font-bold border-b border-m3-outline-variant/60">
          <tr>
            <!-- 管理番号 -->
            <th style="width: {usersWidths.adminNo}px;" class="px-5 py-3.5 relative group select-none">
              <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('adminNo')}>
                <span>No {usersSortKey === 'adminNo' ? (usersSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-m3-primary/30" on:mousedown={(e) => startResize(e, 'adminNo')}></div>
            </th>

            <!-- 学籍番号/職員番号 -->
            <th style="width: {usersWidths.studentNo}px;" class="px-5 py-3.5 relative group select-none">
              <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('studentNo')}>
                <span>学籍/職員番号 {usersSortKey === 'studentNo' ? (usersSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-m3-primary/30" on:mousedown={(e) => startResize(e, 'studentNo')}></div>
            </th>

            <!-- 氏名 -->
            <th style="width: {usersWidths.name}px;" class="px-5 py-3.5 relative group select-none">
              <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('name')}>
                <span>氏名 {usersSortKey === 'name' ? (usersSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-m3-primary/30" on:mousedown={(e) => startResize(e, 'name')}></div>
            </th>

            <!-- フリガナ -->
            <th style="width: {usersWidths.furigana}px;" class="px-5 py-3.5 relative group select-none">
              <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('furigana')}>
                <span>フリガナ {usersSortKey === 'furigana' ? (usersSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-m3-primary/30" on:mousedown={(e) => startResize(e, 'furigana')}></div>
            </th>

            <!-- 性別 -->
            <th style="width: {usersWidths.gender}px;" class="px-4 py-3.5 text-center relative group select-none">
              <div class="flex items-center justify-center cursor-pointer" on:click={() => handleSort('gender')}>
                <span>性別 {usersSortKey === 'gender' ? (usersSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-m3-primary/30" on:mousedown={(e) => startResize(e, 'gender')}></div>
            </th>

            <!-- 区分 -->
            <th style="width: {usersWidths.roleName}px;" class="px-5 py-3.5 relative group select-none">
              <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('roleCode')}>
                <span>区分 {usersSortKey === 'roleCode' ? (usersSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-m3-primary/30" on:mousedown={(e) => startResize(e, 'roleName')}></div>
            </th>

            <!-- 連絡先 (個人情報保護対応) -->
            <th style="width: {usersWidths.contact}px;" class="px-5 py-3.5 relative group select-none">
              <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('contact')}>
                <span>連絡先 {usersSortKey === 'contact' ? (usersSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-m3-primary/30" on:mousedown={(e) => startResize(e, 'contact')}></div>
            </th>

            <!-- 利用目的 -->
            <th style="width: {usersWidths.purpose}px;" class="px-5 py-3.5 relative group select-none">
              <div class="flex items-center justify-between cursor-pointer" on:click={() => handleSort('purpose')}>
                <span>利用目的 {usersSortKey === 'purpose' ? (usersSortDir === 'asc' ? '▲' : '▼') : ''}</span>
              </div>
              <div class="absolute right-0 top-0 bottom-0 w-2 cursor-col-resize hover:bg-m3-primary/30" on:mousedown={(e) => startResize(e, 'purpose')}></div>
            </th>

            <!-- 操作 -->
            <th style="width: {usersWidths.actions}px;" class="px-5 py-3.5 text-right select-none">
              操作
            </th>
          </tr>
        </thead>
        <tbody class="divide-y divide-m3-outline-variant/30">
          {#if filteredUsers.length === 0}
            <tr>
              <td colspan="9" class="px-6 py-10 text-center text-m3-on-surface-variant font-medium">該当するユーザーが見つかりませんでした。</td>
            </tr>
          {:else}
            {#each paginatedUsers as user}
              <tr class="hover:bg-m3-surface-container-high transition-colors duration-150">
                <td class="px-5 py-3.5 font-mono text-m3-on-surface-variant">{user.adminNo || '-'}</td>
                <td class="px-5 py-3.5 font-mono text-m3-on-surface font-semibold">{user.studentNo}</td>
                <td class="px-5 py-3.5 font-bold text-m3-on-surface">{user.name}</td>
                <td class="px-5 py-3.5 text-m3-on-surface-variant text-[11px]">{user.furigana || '-'}</td>
                <td class="px-4 py-3.5 text-center text-m3-on-surface-variant">{user.gender || '-'}</td>
                <td class="px-5 py-3.5">
                  <span class="inline-block px-3 py-0.5 rounded-full text-xs font-bold {user.roleCode === 0 ? 'bg-m3-secondary-container text-m3-on-secondary-container' : (user.roleCode === 9 ? 'bg-m3-warning-container text-m3-on-warning-container' : 'bg-m3-primary-container/70 text-m3-on-primary-container')}">
                    {user.roleName || '学生'}
                  </span>
                </td>
                <!-- 連絡先セル：未クリック時はマスク表示、クリックで開示 (M3 Assist Chip) -->
                <td class="px-5 py-3.5 text-m3-on-surface-variant">
                  {#if !user.contact}
                    <span class="text-m3-on-surface-variant/60">-</span>
                  {:else if showAllContacts || revealedContacts.has(user.cardId || user.studentNo)}
                    <button 
                      type="button" 
                      on:click={() => toggleContactReveal(user.cardId || user.studentNo)}
                      class="inline-flex items-center gap-1.5 text-m3-on-primary-container font-mono text-xs bg-m3-primary-container/70 hover:bg-m3-primary-container px-2.5 py-0.5 rounded-full transition text-left"
                      title="クリックして再度マスクする"
                    >
                      <span>{user.contact}</span>
                      <svg class="w-3 h-3 text-m3-primary flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                      </svg>
                    </button>
                  {:else}
                    <button 
                      type="button" 
                      on:click={() => toggleContactReveal(user.cardId || user.studentNo)}
                      class="inline-flex items-center gap-1.5 text-m3-on-surface-variant hover:text-m3-primary font-mono text-xs bg-m3-surface-container-high hover:bg-m3-primary-container/50 px-2.5 py-0.5 rounded-full transition cursor-pointer"
                      title="クリックして電話番号・連絡先を表示"
                    >
                      <span>••••••••</span>
                      <svg class="w-3 h-3 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l18 18" />
                      </svg>
                    </button>
                  {/if}
                </td>
                <td class="px-5 py-3.5 text-m3-on-surface-variant truncate max-w-[140px]">{user.purpose || '-'}</td>
                <td class="px-5 py-3.5 text-right space-x-1 whitespace-nowrap">
                  <button 
                    on:click={() => onEditUser(user)}
                    class="px-3 py-1 text-xs font-bold text-m3-primary hover:bg-m3-primary-container/60 rounded-full transition"
                  >編集</button>
                  <button 
                    on:click={() => onDeleteUser(user)}
                    class="px-3 py-1 text-xs font-bold text-m3-error hover:bg-m3-error-container/60 rounded-full transition"
                  >削除</button>
                </td>
              </tr>
            {/each}
          {/if}
        </tbody>
      </table>
    </div>

    <!-- M3 Navigation Controls (ページネーションバー) -->
    {#if totalUserPages > 1}
      <div class="px-4 sm:px-6 py-4 border-t border-m3-outline-variant/40 bg-m3-surface-container-low/40">
        <div class="grid grid-cols-1 md:grid-cols-[1fr_auto_1fr] items-center gap-3">
          <span class="text-xs text-m3-on-surface-variant font-medium text-center md:text-left">
            全 {filteredUsers.length} 人 ({userCurrentPage} / {totalUserPages} ページ)
          </span>
          <div class="min-w-0 max-w-full justify-self-center flex flex-wrap items-center justify-center gap-3">
            <nav aria-label="ユーザー一覧のページ" class="flex max-w-full flex-wrap items-center justify-center gap-1 text-xs">
              <button
                disabled={userCurrentPage === 1}
                aria-label="前のページ"
                on:click={() => setUserPage(userCurrentPage - 1)}
                class="h-8 w-8 rounded-full border border-m3-outline-variant hover:bg-m3-surface-container-high disabled:opacity-40 disabled:cursor-not-allowed font-bold"
              >‹</button>

              {#each userPageItems as item (item.key)}
                {#if item.type === 'ellipsis'}
                  <span aria-hidden="true" class="w-5 text-center text-m3-on-surface-variant">…</span>
                {:else}
                  <button
                    aria-current={userCurrentPage === item.page ? 'page' : undefined}
                    on:click={() => setUserPage(item.page)}
                    class="h-8 min-w-8 rounded-full px-2 font-bold transition-colors duration-150 {userCurrentPage === item.page ? 'bg-m3-primary text-m3-on-primary shadow-m3-1' : 'text-m3-on-surface-variant hover:bg-m3-surface-container-high'}"
                  >
                    {item.page}
                  </button>
                {/if}
              {/each}

              <button
                disabled={userCurrentPage === totalUserPages}
                aria-label="次のページ"
                on:click={() => setUserPage(userCurrentPage + 1)}
                class="h-8 w-8 rounded-full border border-m3-outline-variant hover:bg-m3-surface-container-high disabled:opacity-40 disabled:cursor-not-allowed font-bold"
              >›</button>
            </nav>
            <form class="flex items-center justify-center gap-2 text-xs" on:submit|preventDefault={() => setUserPage(userPageInput)}>
              <label for="user-page-input" class="text-m3-on-surface-variant">ページへ移動</label>
              <input
                id="user-page-input"
                type="number"
                min="1"
                max={totalUserPages}
                bind:value={userPageInput}
                on:change={(event) => setUserPage(event.currentTarget.value)}
                class="h-8 w-16 rounded-full border border-m3-outline-variant bg-m3-surface-container-lowest text-center text-xs focus:outline-none focus:border-m3-primary"
              />
              <span class="text-m3-on-surface-variant">/ {totalUserPages}</span>
              <button type="submit" class="m3-btn-tonal py-1.5 px-3 text-xs">移動</button>
            </form>
          </div>
          <div class="hidden md:block"></div>
        </div>
      </div>
    {/if}
  </div>
</div>
