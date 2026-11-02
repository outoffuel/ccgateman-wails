<script>
  export let isOpen = false;
  export let mode = 'create'; // 'create' | 'edit'
  export let initialUser = null;
  export let onClose = () => {};
  export let onSave = async () => {};

  let formStudentNo = '';
  let formName = '';
  let formFurigana = '';
  let formGender = '男';
  let formRoleName = '学生';
  let formRoleCode = 1;
  let formCardId = '';
  let formContact = '';
  let formPurpose = '';
  let formAdminNo = '';
  let formError = '';
  let isSaving = false;

  $: if (isOpen) {
    if (mode === 'edit' && initialUser) {
      formStudentNo = initialUser.studentNo || '';
      formName = initialUser.name || '';
      formFurigana = initialUser.furigana || '';
      formGender = initialUser.gender || '男';
      formRoleName = initialUser.roleName || '学生';
      formRoleCode = initialUser.roleCode ?? (initialUser.roleName === '教職員' ? 0 : initialUser.roleName === '学生スタッフ' ? 9 : 1);
      formCardId = initialUser.cardId || '';
      formContact = initialUser.contact || '';
      formPurpose = initialUser.purpose || '';
      formAdminNo = initialUser.adminNo || '';
    } else {
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
    }
    formError = '';
    isSaving = false;
  }

  function handleRoleChange() {
    if (formRoleName === '教職員') formRoleCode = 0;
    else if (formRoleName === '学生スタッフ') formRoleCode = 9;
    else formRoleCode = 1;
  }

  async function handleSubmit() {
    if (!formStudentNo.trim()) {
      formError = '学籍番号 / 職員番号は必須です';
      return;
    }
    if (!formName.trim()) {
      formError = '氏名は必須です';
      return;
    }

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

    isSaving = true;
    formError = '';
    try {
      await onSave(userData);
      onClose();
    } catch (err) {
      formError = '保存に失敗しました: ' + (err?.message || err);
    } finally {
      isSaving = false;
    }
  }
</script>

{#if isOpen}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/50 backdrop-blur-xs p-4">
    <div class="bg-white rounded-2xl shadow-2xl border border-slate-200 w-full max-w-lg overflow-hidden max-h-[90vh] flex flex-col">
      <!-- モーダルヘッダー -->
      <div class="px-6 py-4 border-b border-slate-100 flex items-center justify-between bg-slate-50">
        <h3 class="text-base font-bold text-slate-800">
          {mode === 'create' ? '利用者新規登録' : '利用者情報の編集'}
        </h3>
        <button on:click={onClose} class="text-slate-400 hover:text-slate-600 p-1 rounded-lg">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
          </svg>
        </button>
      </div>

      <!-- フォームスクロール領域 -->
      <form on:submit|preventDefault={handleSubmit} class="p-6 space-y-4 overflow-y-auto flex-1">
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
            on:click={onClose} 
            class="px-4 py-2 border border-slate-300 text-slate-700 rounded-lg text-xs font-semibold hover:bg-slate-50"
          >
            キャンセル
          </button>
          <button 
            type="submit" 
            disabled={isSaving}
            class="px-5 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-xs font-bold shadow-sm disabled:opacity-50"
          >
            {isSaving ? '保存中...' : (mode === 'create' ? '登録' : '保存')}
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}

