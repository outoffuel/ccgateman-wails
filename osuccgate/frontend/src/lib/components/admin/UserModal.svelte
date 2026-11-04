<script>
  import { tick } from 'svelte';

  export let isOpen = false;
  export let mode = 'create'; // 'create' | 'edit' | 'renewal'
  export let initialUser = null;
  export let currentFiscalYear = 2026;
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

  let studentNoInput;
  let nameInput;
  let wasOpen = false;

  function initForm() {
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
    } else if (mode === 'renewal' && initialUser) {
      // 過去年度からの更新: 前年度の登録情報を引き継ぎ、管理番号は今年度の連番で自動採番するため空欄にする
      formStudentNo = initialUser.studentNo || '';
      formName = initialUser.name || '';
      formFurigana = initialUser.furigana || '';
      formGender = initialUser.gender || '男';
      formRoleName = initialUser.roleName || '学生';
      formRoleCode = initialUser.roleCode ?? (initialUser.roleName === '教職員' ? 0 : initialUser.roleName === '学生スタッフ' ? 9 : 1);
      formCardId = initialUser.cardId || '';
      formContact = initialUser.contact || '';
      formPurpose = initialUser.purpose || '';
      formAdminNo = ''; // 今年度の新規採番
    } else if (mode === 'create' && initialUser) {
      // 未登録カードからの新規登録: 渡されたカードID/学籍番号を自動セット
      formStudentNo = initialUser.studentNo || initialUser.cardId || '';
      formName = initialUser.name || '';
      formFurigana = initialUser.furigana || '';
      formGender = initialUser.gender || '男';
      formRoleName = initialUser.roleName || '学生';
      formRoleCode = initialUser.roleCode ?? 1;
      formCardId = initialUser.cardId || '';
      formContact = initialUser.contact || '';
      formPurpose = initialUser.purpose || '';
      formAdminNo = '';
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

    // 開いた直後に適切な入力欄にフォーカスを当てる (WebKitGTK/Linux対応)
    tick().then(() => {
      if (formStudentNo && !formName) {
        nameInput?.focus();
      } else {
        studentNoInput?.focus();
      }
    });
    setTimeout(() => {
      if (formStudentNo && !formName) {
        nameInput?.focus();
      } else {
        studentNoInput?.focus();
      }
    }, 60);
  }

  // モーダルが「閉じた状態から開いた瞬間」のみ初期化を実行 (入力中の上書きリセットを防止)
  $: if (isOpen && !wasOpen) {
    wasOpen = true;
    initForm();
  } else if (!isOpen && wasOpen) {
    wasOpen = false;
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
      fiscalYear: currentFiscalYear,
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
  <!-- M3 Dialog Scrim (No backdrop-blur for smooth rendering) -->
  <div 
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4 select-auto animate-fade"
    on:click|self={onClose}
  >
    <!-- M3 Dialog Container (rounded-[28px]) -->
    <div 
      class="bg-m3-surface-container-lowest rounded-[28px] shadow-m3-3 border border-m3-outline-variant w-full max-w-lg overflow-hidden max-h-[90vh] flex flex-col select-text"
      on:click|stopPropagation
    >
      <!-- モーダルヘッダー -->
      <div class="px-7 pt-6 pb-4 border-b border-m3-outline-variant flex items-center justify-between bg-m3-surface-container-low">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-2xl bg-m3-primary-container text-m3-on-primary-container flex items-center justify-center flex-shrink-0">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"/>
            </svg>
          </div>
          <div>
            <h3 class="text-base font-bold text-m3-on-surface">
              {mode === 'create' ? '利用者新規登録' : (mode === 'renewal' ? `利用者登録の更新（${currentFiscalYear}年度）` : '利用者情報の編集')}
            </h3>
            <p class="text-[11px] text-m3-on-surface-variant font-medium">
              {currentFiscalYear}年度の利用者マスターとして保存されます
            </p>
          </div>
        </div>
        <button on:click={onClose} class="text-m3-on-surface-variant hover:text-m3-on-surface hover:bg-m3-surface-container-high p-2 rounded-full transition-colors duration-150">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
          </svg>
        </button>
      </div>

      <!-- フォームスクロール領域 (M3 Outlined Text Fields) -->
      <form on:submit|preventDefault={handleSubmit} class="px-7 py-6 space-y-4 overflow-y-auto flex-1">
        {#if formError}
          <div class="p-3.5 rounded-2xl bg-m3-error-container text-m3-on-error-container text-xs font-bold">
            {formError}
          </div>
        {/if}

        <!-- 1. 学籍番号 / 職員番号 (必須) -->
        <div>
          <label for="form-student-no" class="block text-xs font-bold text-m3-on-surface-variant mb-1.5 ml-1">
            学籍番号 / 職員番号 <span class="text-m3-error">*</span>
          </label>
          <input 
            id="form-student-no"
            type="text" 
            bind:this={studentNoInput}
            bind:value={formStudentNo} 
            required 
            placeholder="例: A0012345" 
            class="m3-input-outlined font-mono uppercase" 
          />
          <p class="text-[11px] text-m3-on-surface-variant/80 mt-1 ml-1">※カードIDが空の場合はこの番号が照合に使われます</p>
        </div>

        <!-- 2. 氏名 (必須) -->
        <div>
          <label for="form-name" class="block text-xs font-bold text-m3-on-surface-variant mb-1.5 ml-1">
            氏名 <span class="text-m3-error">*</span>
          </label>
          <input 
            id="form-name"
            type="text" 
            bind:this={nameInput}
            bind:value={formName} 
            required 
            placeholder="例: 山田 太郎" 
            class="m3-input-outlined" 
          />
        </div>

        <!-- 3. フリガナ -->
        <div>
          <label for="form-furigana" class="block text-xs font-bold text-m3-on-surface-variant mb-1.5 ml-1">フリガナ</label>
          <input 
            id="form-furigana"
            type="text" 
            bind:value={formFurigana} 
            placeholder="例: ヤマダ タロウ" 
            class="m3-input-outlined" 
          />
        </div>

        <!-- 4. 性別 & 5. 区分 -->
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label for="form-gender" class="block text-xs font-bold text-m3-on-surface-variant mb-1.5 ml-1">性別</label>
            <select 
              id="form-gender"
              bind:value={formGender} 
              class="m3-input-outlined"
            >
              <option value="男">男</option>
              <option value="女">女</option>
            </select>
          </div>

          <div>
            <label for="form-role" class="block text-xs font-bold text-m3-on-surface-variant mb-1.5 ml-1">区分</label>
            <select 
              id="form-role"
              bind:value={formRoleName} 
              on:change={handleRoleChange}
              class="m3-input-outlined font-semibold"
            >
              <option value="学生">学生</option>
              <option value="教職員">教職員</option>
              <option value="学生スタッフ">学生スタッフ</option>
            </select>
          </div>
        </div>

        <!-- 6. カードID (任意・空なら学籍番号と同じ) -->
        <div>
          <label for="form-card-id" class="block text-xs font-bold text-m3-on-surface-variant mb-1.5 ml-1">
            カードID (任意 / NFC・磁気併用時)
          </label>
          <input 
            id="form-card-id"
            type="text" 
            bind:value={formCardId} 
            placeholder="未入力の場合は学籍番号と同じ値が自動適用されます" 
            class="m3-input-outlined font-mono" 
          />
          <p class="text-[11px] text-m3-on-surface-variant/80 mt-1 ml-1">※PaSoRiのNFC IDm番号や個別カード番号を別途割り当てる場合に入力</p>
        </div>

        <!-- 7. 連絡先 -->
        <div>
          <label for="form-contact" class="block text-xs font-bold text-m3-on-surface-variant mb-1.5 ml-1">連絡先 (任意)</label>
          <input 
            id="form-contact"
            type="text" 
            bind:value={formContact} 
            placeholder="例: 090-xxxx-xxxx / user@example.com" 
            class="m3-input-outlined" 
          />
        </div>

        <!-- 8. 利用目的 -->
        <div>
          <label for="form-purpose" class="block text-xs font-bold text-m3-on-surface-variant mb-1.5 ml-1">利用目的 (任意)</label>
          <input 
            id="form-purpose"
            type="text" 
            bind:value={formPurpose} 
            placeholder="例: 卒業研究 / 自習 / TA業務" 
            class="m3-input-outlined" 
          />
        </div>

        <!-- M3 Dialog Actions (Pill Buttons) -->
        <div class="pt-4 border-t border-m3-outline-variant/40 flex items-center justify-end gap-3 sticky bottom-0 bg-m3-surface-container-lowest">
          <button 
            type="button" 
            on:click={onClose} 
            class="m3-btn-outlined py-2.5 px-5"
          >
            キャンセル
          </button>
          <button 
            type="submit" 
            disabled={isSaving}
            class="m3-btn-filled py-2.5 px-6"
          >
            {isSaving ? '保存中...' : (mode === 'create' ? '登録する' : (mode === 'renewal' ? '今年度として登録' : '保存する'))}
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}

<style>
  .animate-fade {
    animation: fadeIn 0.2s cubic-bezier(0.2, 0, 0, 1);
  }
  @keyframes fadeIn {
    from { opacity: 0; transform: scale(0.97); }
    to { opacity: 1; transform: scale(1); }
  }
</style>
