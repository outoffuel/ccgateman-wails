<script>
  export let stats = {
    fiscalYear: 2026,
    currentInsideCount: 0,
    totalUserCount: 0,
    todayLogCount: 0,
    todayEntryCount: 0,
    todayExitCount: 0,
    weekLogCount: 0,
    weekEntryCount: 0,
    weekExitCount: 0,
    monthLogCount: 0,
    monthEntryCount: 0,
    monthExitCount: 0,
    yearLogCount: 0,
    yearEntryCount: 0,
    yearExitCount: 0,
    weekChart: [],
    monthChart: [],
    yearChart: []
  };
  export let insideUsers = [];
  export let processedLogs = [];
  export let monthlyStatsRows = [];
  export let availableFiscalYears = [2026];
  export let selectedFiscalYear = 2026;
  export let currentFiscalYear = 2026;
  export let onSelectFiscalYear = () => {};
  export let onNavigateToLogs = () => {};
  export let onRenewalUser = () => {};
  export let onRegisterUnregistered = () => {};

  // 表示対象の年度 (選択年度に完全連動)
  $: activeFiscalYear = selectedFiscalYear || stats?.fiscalYear || currentFiscalYear;

  // 1. 在室人数・本日内訳 (SQL集計値を使用)
  $: currentInsideNum = insideUsers ? insideUsers.length : (stats?.currentInsideCount ?? 0);
  $: todayEntriesCount = stats?.todayEntryCount ?? 0;
  $: todayExitsCount = stats?.todayExitCount ?? 0;

  // 2. 週間の入退室総数 (直近7日間: SQL集計値を使用し、合計 = 入室 + 退室 を厳密に一致させる)
  $: weekEntries = stats?.weekEntryCount ?? 0;
  $: weekExits = stats?.weekExitCount ?? 0;
  $: weekTotalCount = weekEntries + weekExits;

  // 3. 月間の入退室総数 (当月1日以降: SQL集計値を使用し、合計 = 入室 + 退室 を厳密に一致させる)
  $: monthEntries = stats?.monthEntryCount ?? 0;
  $: monthExits = stats?.monthExitCount ?? 0;
  $: monthTotalCount = monthEntries + monthExits;

  // 4. 年間の入退室総数 (選択年度4月〜翌3月: SQL集計値を使用し、合計 = 入室 + 退室 を厳密に一致させる)
  $: yearEntries = stats?.yearEntryCount ?? 0;
  $: yearExits = stats?.yearExitCount ?? 0;
  $: yearTotalCount = yearEntries + yearExits;

  // 選択年度の月別統計行
  $: fiscalYearMonthlyRows = (monthlyStatsRows || []).filter(r => r.fiscalYear === activeFiscalYear);

  // グラフ表示期間 ('week' | 'month' | 'year')
  let chartPeriod = 'week';

  // バックエンドSQL集計によるグラフデータ (500件制限および架空推計データを完全排除)
  $: currentChartList = (() => {
    if (chartPeriod === 'week') {
      if (stats?.weekChart && stats.weekChart.length > 0) {
        return stats.weekChart;
      }
      return Array.from({ length: 7 }, (_, i) => {
        const d = new Date();
        d.setDate(d.getDate() - (6 - i));
        const dayOfWeek = ['日', '月', '火', '水', '木', '金', '土'][d.getDay()];
        return {
          label: `${d.getMonth() + 1}/${d.getDate()}(${dayOfWeek})`,
          entries: 0,
          exits: 0,
        };
      });
    }

    if (chartPeriod === 'month') {
      if (stats?.monthChart && stats.monthChart.length > 0) {
        return stats.monthChart;
      }
      return Array.from({ length: 4 }, (_, i) => ({
        label: `${4 - i}週前`,
        entries: 0,
        exits: 0,
      }));
    }

    // year (選択年度の4月〜翌3月実数)
    if (stats?.yearChart && stats.yearChart.length > 0) {
      return stats.yearChart;
    }
    return [4, 5, 6, 7, 8, 9, 10, 11, 12, 1, 2, 3].map(m => ({
      label: `${m}月`,
      entries: 0,
      exits: 0,
    }));
  })();

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
</script>

<div class="space-y-6">
  <!-- 年度選択バー & アーカイブ通知 (M3 Tonal Surface Banner) -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 bg-m3-surface-container-low rounded-2xl px-6 py-3.5 border border-m3-outline-variant/60">
    <div class="flex flex-wrap items-center gap-2.5 text-xs text-m3-on-surface-variant">
      <span class="font-bold text-m3-on-surface">集計対象年度：</span>
      <span class="px-3.5 py-1 rounded-full font-bold bg-m3-primary-container text-m3-on-primary-container">
        {activeFiscalYear}年度（{activeFiscalYear}/04/01 ～ {activeFiscalYear + 1}/03/31）
      </span>
      {#if currentFiscalYear && activeFiscalYear !== currentFiscalYear}
        <span class="px-3 py-1 rounded-full bg-m3-warning-container text-m3-on-warning-container font-bold">
          アーカイブ表示中
        </span>
      {/if}
    </div>

    <div class="flex items-center gap-3">
      {#if currentFiscalYear && activeFiscalYear !== currentFiscalYear}
        <button
          on:click={() => onSelectFiscalYear(currentFiscalYear)}
          class="m3-btn-tonal py-1.5 px-3.5 text-xs"
        >
          今年度（{currentFiscalYear}年度）に戻る
        </button>
      {/if}
      <div class="flex items-center gap-2 bg-m3-surface-container-lowest border border-m3-outline-variant rounded-full px-4 py-1.5 text-xs">
        <label for="dashboard-top-fy-select" class="font-bold text-m3-on-surface-variant">表示年度:</label>
        <select
          id="dashboard-top-fy-select"
          value={selectedFiscalYear}
          on:change={(e) => onSelectFiscalYear(Number(e.target.value))}
          class="font-bold text-m3-primary bg-transparent focus:outline-none cursor-pointer"
        >
          {#each availableFiscalYears as fy}
            <option value={fy}>{fy}年度</option>
          {/each}
        </select>
      </div>
    </div>
  </div>

  <!-- 4つの統計カード横並び (M3 Elevated / Filled Cards) -->
  <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5">
    <!-- カード1：現在の在室人数 (Primary Container Icon) -->
    <div class="m3-card-elevated p-6 flex items-center justify-between">
      <div>
        <p class="text-xs font-bold text-m3-on-surface-variant tracking-wide">現在の在室人数</p>
        <div class="mt-2.5 flex items-baseline gap-1.5">
          <span class="text-4xl font-extrabold text-m3-on-surface tracking-tight">{currentInsideNum}</span>
          <span class="text-sm text-m3-on-surface-variant font-semibold">人</span>
        </div>
        <p class="text-[11px] text-m3-on-surface-variant/80 mt-1.5 font-medium">
          本日 入室 {todayEntriesCount} / 退室 {todayExitsCount}
        </p>
      </div>
      <div class="w-14 h-14 rounded-2xl bg-m3-primary-container text-m3-on-primary-container flex items-center justify-center flex-shrink-0">
        <svg class="w-7 h-7" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"/>
        </svg>
      </div>
    </div>

    <!-- カード2：週間の入退室数 (Tertiary Container Icon) -->
    <div class="m3-card-elevated p-6 flex items-center justify-between">
      <div>
        <p class="text-xs font-bold text-m3-on-surface-variant tracking-wide">週間の入退室数</p>
        <div class="mt-2.5 flex items-baseline gap-1.5">
          <span class="text-4xl font-extrabold text-m3-on-surface tracking-tight">{weekTotalCount}</span>
          <span class="text-sm text-m3-on-surface-variant font-semibold">回</span>
        </div>
        <p class="text-[11px] text-m3-on-surface-variant/80 mt-1.5 font-medium">
          直近7日間 (入室 {weekEntries} / 退室 {weekExits})
        </p>
      </div>
      <div class="w-14 h-14 rounded-2xl bg-m3-tertiary-container text-m3-on-tertiary-container flex items-center justify-center flex-shrink-0">
        <svg class="w-7 h-7" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"/>
        </svg>
      </div>
    </div>

    <!-- カード3：月間の入退室数 (Secondary Container Icon) -->
    <div class="m3-card-elevated p-6 flex items-center justify-between">
      <div>
        <p class="text-xs font-bold text-m3-on-surface-variant tracking-wide">月間の入退室数</p>
        <div class="mt-2.5 flex items-baseline gap-1.5">
          <span class="text-4xl font-extrabold text-m3-on-surface tracking-tight">{monthTotalCount}</span>
          <span class="text-sm text-m3-on-surface-variant font-semibold">回</span>
        </div>
        <p class="text-[11px] text-m3-on-surface-variant/80 mt-1.5 font-medium">
          当月累計 (入室 {monthEntries} / 退室 {monthExits})
        </p>
      </div>
      <div class="w-14 h-14 rounded-2xl bg-m3-secondary-container text-m3-on-secondary-container flex items-center justify-center flex-shrink-0">
        <svg class="w-7 h-7" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"/>
        </svg>
      </div>
    </div>

    <!-- カード4：年間の入退室数 (Warning Container Icon) -->
    <div class="m3-card-elevated p-6 flex items-center justify-between">
      <div>
        <p class="text-xs font-bold text-m3-on-surface-variant tracking-wide">年間の入退室数 ({activeFiscalYear}年度)</p>
        <div class="mt-2.5 flex items-baseline gap-1.5">
          <span class="text-4xl font-extrabold text-m3-on-surface tracking-tight">{yearTotalCount}</span>
          <span class="text-sm text-m3-on-surface-variant font-semibold">回</span>
        </div>
        <p class="text-[11px] text-m3-on-surface-variant/80 mt-1.5 font-medium">
          {activeFiscalYear}年度累計 (入室 {yearEntries} / 退室 {yearExits})
        </p>
      </div>
      <div class="w-14 h-14 rounded-2xl bg-m3-warning-container text-m3-on-warning-container flex items-center justify-center flex-shrink-0">
        <svg class="w-7 h-7" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 8v8m-4-5v5m-4-2v2m-2 4h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"/>
        </svg>
      </div>
    </div>
  </div>

  <!-- グラフ & 最近の入退室履歴 -->
  <div class="grid grid-cols-1 lg:grid-cols-12 gap-6">
    <!-- 入退室数グラフ (M3 Elevated Card + M3 Segmented Buttons) -->
    <div class="lg:col-span-7 m3-card-elevated p-6 flex flex-col justify-between">
      <div>
        <div class="flex flex-wrap items-center justify-between pb-4 border-b border-m3-outline-variant/40 gap-3">
          <div class="flex items-center gap-3.5">
            <h3 class="text-base font-bold text-m3-on-surface">入退室数</h3>
            <!-- M3 Segmented Buttons (週 / 月 / 年間) -->
            <div class="inline-flex rounded-full border border-m3-outline/60 overflow-hidden text-xs font-bold divide-x divide-m3-outline/50 bg-m3-surface-container-lowest">
              <button 
                on:click={() => chartPeriod = 'week'}
                class="inline-flex items-center gap-1 px-4 py-1.5 transition-colors duration-150 {chartPeriod === 'week' ? 'bg-m3-secondary-container text-m3-on-secondary-container' : 'text-m3-on-surface-variant hover:bg-m3-surface-container-high'}"
              >
                {#if chartPeriod === 'week'}
                  <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7"/>
                  </svg>
                {/if}
                <span>週</span>
              </button>
              <button 
                on:click={() => chartPeriod = 'month'}
                class="inline-flex items-center gap-1 px-4 py-1.5 transition-colors duration-150 {chartPeriod === 'month' ? 'bg-m3-secondary-container text-m3-on-secondary-container' : 'text-m3-on-surface-variant hover:bg-m3-surface-container-high'}"
              >
                {#if chartPeriod === 'month'}
                  <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7"/>
                  </svg>
                {/if}
                <span>月</span>
              </button>
              <button 
                on:click={() => chartPeriod = 'year'}
                class="inline-flex items-center gap-1 px-4 py-1.5 transition-colors duration-150 {chartPeriod === 'year' ? 'bg-m3-secondary-container text-m3-on-secondary-container' : 'text-m3-on-surface-variant hover:bg-m3-surface-container-high'}"
              >
                {#if chartPeriod === 'year'}
                  <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7"/>
                  </svg>
                {/if}
                <span>年間</span>
              </button>
            </div>
          </div>

          <div class="flex items-center gap-4 text-xs font-medium">
            <div class="flex items-center gap-1.5">
              <span class="w-3 h-3 rounded-full bg-m3-primary"></span>
              <span class="text-m3-on-surface-variant">入室</span>
            </div>
            <div class="flex items-center gap-1.5">
              <span class="w-3 h-3 rounded-full bg-m3-error"></span>
              <span class="text-m3-on-surface-variant">退室（強制退室含む）</span>
            </div>
          </div>
        </div>

        <div class="relative mt-6">
          <svg class="w-full h-64 overflow-visible" viewBox="0 0 520 220">
            <!-- 背景グリッド & 目盛り数値 (動的スケール) -->
            <g class="text-m3-on-surface-variant text-[10px] font-mono">
              <!-- 最大値ライン -->
              <line x1="40" y1="45" x2="500" y2="45" stroke="#E6E8EE" stroke-width="1" />
              <text x="32" y="49" text-anchor="end" fill="#74777F">{chartMax}回</text>
              
              <!-- 3/4ライン -->
              <line x1="40" y1="81" x2="500" y2="81" stroke="#E6E8EE" stroke-width="1" />
              <text x="32" y="85" text-anchor="end" fill="#74777F">{Math.round(chartMax * 0.75)}回</text>
              
              <!-- 1/2ライン -->
              <line x1="40" y1="117" x2="500" y2="117" stroke="#E6E8EE" stroke-width="1" />
              <text x="32" y="121" text-anchor="end" fill="#74777F">{Math.round(chartMax * 0.5)}回</text>
              
              <!-- 1/4ライン -->
              <line x1="40" y1="154" x2="500" y2="154" stroke="#E6E8EE" stroke-width="1" />
              <text x="32" y="158" text-anchor="end" fill="#74777F">{Math.round(chartMax * 0.25)}回</text>
              
              <!-- ゼロ基準線 -->
              <line x1="40" y1="190" x2="500" y2="190" stroke="#C4C6D0" stroke-width="1.5" />
              <text x="32" y="194" text-anchor="end" fill="#74777F">0</text>
            </g>

            <!-- 各期間の棒グラフ (SQL集計による実数を描画) -->
            {#each activeHourlyChartData as slot}
              <g class="group">
                <!-- 入室バー (M3 Primary) -->
                {#if slot.entries > 0}
                  <rect 
                    class="chart-bar" 
                    x={slot.entryX} 
                    y={190 - slot.entries * chartScale} 
                    width={slot.barWidth} 
                    height={slot.entries * chartScale} 
                    rx="4" 
                    fill="#0B57D0" 
                  />
                  <text 
                    x={slot.entryX + slot.barWidth / 2} 
                    y={190 - slot.entries * chartScale - 4} 
                    text-anchor="middle" 
                    font-size="9" 
                    font-weight="bold" 
                    fill="#0B57D0"
                  >{slot.entries}</text>
                {/if}

                <!-- 退室バー (M3 Error/Rose) -->
                {#if slot.exits > 0}
                  <rect 
                    class="chart-bar" 
                    x={slot.exitX} 
                    y={190 - slot.exits * chartScale} 
                    width={slot.barWidth} 
                    height={slot.exits * chartScale} 
                    rx="4" 
                    fill="#B3261E" 
                  />
                  <text 
                    x={slot.exitX + slot.barWidth / 2} 
                    y={190 - slot.exits * chartScale - 4} 
                    text-anchor="middle" 
                    font-size="9" 
                    font-weight="bold" 
                    fill="#B3261E"
                  >{slot.exits}</text>
                {/if}

                <!-- X軸ラベル -->
                <text x={slot.centerX} y="208" text-anchor="middle" font-size="10" fill="#44474E">
                  {slot.label}
                </text>
              </g>
            {/each}
          </svg>
        </div>
      </div>

      <!-- グラフ下部フッター注釈 -->
      <div class="mt-4 pt-3.5 border-t border-m3-outline-variant/40 flex items-center justify-between text-xs text-m3-on-surface-variant">
        <span>
          ※表示期間: {chartPeriod === 'week' ? '週（直近7日間・日別実数）' : (chartPeriod === 'month' ? '月（直近4週間・週別実数）' : `${activeFiscalYear}年度（4月〜翌3月・月別実数）`)}
        </span>
        <button 
          on:click={onNavigateToLogs}
          class="text-m3-primary hover:underline font-bold"
        >
          詳細ログを見る →
        </button>
      </div>
    </div>

    <!-- 最近の入退室履歴 (ダッシュボード右側 M3 Elevated Card) -->
    <div class="lg:col-span-5 m3-card-elevated p-6 flex flex-col justify-between">
      <div>
        <div class="flex items-center justify-between pb-3.5 border-b border-m3-outline-variant/40">
          <h3 class="text-base font-bold text-m3-on-surface">最近の入退室履歴</h3>
          <button on:click={onNavigateToLogs} class="m3-btn-tonal py-1 px-3 text-xs">
            <span>すべて見る</span>
            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
            </svg>
          </button>
        </div>

        <div class="overflow-x-auto mt-2">
          <table class="w-full text-left text-xs">
            <thead>
              <tr class="text-m3-on-surface-variant border-b border-m3-outline-variant/40">
                <th class="py-2.5 font-bold">時刻</th>
                <th class="py-2.5 font-bold">氏名</th>
                <th class="py-2.5 font-bold text-center">区分</th>
                <th class="py-2.5 font-bold text-right">種類 / 操作</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-m3-outline-variant/30 font-medium">
              {#each processedLogs.slice(0, 5) as log}
                <tr class="hover:bg-m3-surface-container-high transition-colors duration-150">
                  <td class="py-3 text-m3-on-surface-variant font-mono">
                    {log.formatted.split(' ')[1] || log.formatted}
                  </td>
                  <td class="py-3 text-m3-on-surface font-bold">{log.userName || '-'}</td>
                  <td class="py-3 text-center text-m3-on-surface-variant">{log.roleName || '-'}</td>
                  <td class="py-3 text-right whitespace-nowrap">
                    {#if log.eventType === 'unregistered_renewal'}
                      {#if log.isRegistered}
                        <span class="inline-block px-3 py-0.5 rounded-full text-[11px] font-bold bg-m3-secondary-container text-m3-on-secondary-container">
                          {log.typeLabel}
                        </span>
                      {:else}
                        <div class="inline-flex items-center gap-1.5 justify-end">
                          <span class="inline-block px-3 py-0.5 rounded-full text-[11px] font-bold bg-m3-warning-container text-m3-on-warning-container">
                            {log.typeLabel}
                          </span>
                          <button
                            on:click={() => onRenewalUser(log)}
                            class="px-2.5 py-0.5 bg-m3-warning hover:brightness-105 text-m3-on-warning rounded-full text-[10px] font-bold transition shadow-m3-1"
                            title="前年度データを引き継いで今年度更新"
                          >
                            🔄 更新
                          </button>
                        </div>
                      {/if}
                    {:else if log.eventType === 'unregistered'}
                      {#if log.isRegistered}
                        <span class="inline-block px-3 py-0.5 rounded-full text-[11px] font-bold bg-m3-secondary-container text-m3-on-secondary-container">
                          {log.typeLabel}
                        </span>
                      {:else}
                        <div class="inline-flex items-center gap-1.5 justify-end">
                          <span class="inline-block px-3 py-0.5 rounded-full text-[11px] font-bold bg-m3-error-container text-m3-on-error-container">
                            {log.typeLabel}
                          </span>
                          <button
                            on:click={() => onRegisterUnregistered(log)}
                            class="px-2.5 py-0.5 bg-m3-primary hover:brightness-105 text-m3-on-primary rounded-full text-[10px] font-bold transition shadow-m3-1"
                            title="カードIDで新規登録"
                          >
                            ＋ 登録
                          </button>
                        </div>
                      {/if}
                    {:else}
                      <span class="inline-block px-3 py-0.5 rounded-full text-[11px] font-bold {log.eventType === 'entry' ? 'bg-m3-primary-container text-m3-on-primary-container' : (log.eventType === 'force_exit' ? 'bg-m3-warning-container text-m3-on-warning-container' : 'bg-m3-error-container text-m3-on-error-container')}">
                        {log.typeLabel}
                      </span>
                    {/if}
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>

      <div class="pt-3.5 border-t border-m3-outline-variant/40 text-center">
        <button on:click={onNavigateToLogs} class="text-xs text-m3-primary hover:underline font-bold">
          すべての入退室履歴を確認する →
        </button>
      </div>
    </div>
  </div>

  <!-- 月別・区分別入退室集計テーブル (M3 Data Table Style) -->
  <div class="m3-card-elevated overflow-hidden">
    <div class="px-6 py-5 flex flex-col sm:flex-row sm:items-center justify-between border-b border-m3-outline-variant/40 gap-3">
      <div class="flex items-center gap-3">
        <h3 class="text-base font-bold text-m3-on-surface">月別・区分別入退室記録</h3>
        <span class="px-3 py-0.5 rounded-full text-xs bg-m3-secondary-container text-m3-on-secondary-container font-bold">
          {activeFiscalYear}年度
        </span>
      </div>
      <div class="flex flex-wrap items-center gap-3">
        <!-- 表示年度プルダウン (M3 Pill Select) -->
        <div class="flex items-center gap-2 bg-m3-surface-container-low border border-m3-outline-variant rounded-full px-3.5 py-1 text-xs">
          <label for="dashboard-fy-select" class="font-semibold text-m3-on-surface-variant">表示年度:</label>
          <select 
            id="dashboard-fy-select"
            value={selectedFiscalYear} 
            on:change={(e) => onSelectFiscalYear(Number(e.target.value))}
            class="font-bold text-m3-primary bg-transparent focus:outline-none cursor-pointer"
          >
            {#each availableFiscalYears as fy}
              <option value={fy}>{fy}年度</option>
            {/each}
          </select>
        </div>
        <span class="text-xs text-m3-on-surface-variant">
          ※区分内訳および当月までの合計は「入室回数（のべ利用人数）」、入退室総数は「入室＋退室」
        </span>
      </div>
    </div>

    <div class="overflow-x-auto">
      <table class="w-full text-xs text-left border-collapse">
        <thead class="bg-m3-surface-container-low text-m3-on-surface-variant font-bold border-b border-m3-outline-variant/60">
          <tr>
            <th class="px-6 py-3.5">年月</th>
            <th class="px-4 py-3.5 text-center">
              <span class="px-2.5 py-0.5 rounded-full bg-m3-surface-container-high text-m3-on-surface-variant text-[11px]">- (その他)</span>
            </th>
            <th class="px-4 py-3.5 text-center">
              <span class="px-2.5 py-0.5 rounded-full bg-m3-secondary-container text-m3-on-secondary-container text-[11px]">0（教職員）</span>
            </th>
            <th class="px-4 py-3.5 text-center">
              <span class="px-2.5 py-0.5 rounded-full bg-m3-primary-container text-m3-on-primary-container text-[11px]">1（学生）</span>
            </th>
            <th class="px-4 py-3.5 text-center">
              <span class="px-2.5 py-0.5 rounded-full bg-m3-warning-container text-m3-on-warning-container text-[11px]">9（スタッフ）</span>
            </th>
            <th class="px-4 py-3.5 text-right">当月入室計</th>
            <th class="px-5 py-3.5 text-right">当月までの合計（入室累計）</th>
            <th class="px-6 py-3.5 text-right text-m3-on-surface-variant">当月入退室総数（入室/退室）</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-m3-outline-variant/30">
          {#if fiscalYearMonthlyRows.length === 0}
            <tr>
              <td colspan="8" class="px-6 py-10 text-center text-m3-on-surface-variant font-medium">
                {activeFiscalYear}年度の入退室集計データがありません。
              </td>
            </tr>
          {:else}
            {#each fiscalYearMonthlyRows as row}
              <tr class="hover:bg-m3-surface-container-high transition-colors duration-150">
                <td class="px-6 py-3.5 font-mono font-bold text-m3-on-surface">{row.yearMonth}</td>
                <td class="px-4 py-3.5 text-center text-m3-on-surface-variant">{row.roleOtherCount}</td>
                <td class="px-4 py-3.5 text-center font-bold text-m3-secondary">{row.role0Count}</td>
                <td class="px-4 py-3.5 text-center font-bold text-m3-primary">{row.role1Count}</td>
                <td class="px-4 py-3.5 text-center font-bold text-m3-warning">{row.role9Count}</td>
                <td class="px-4 py-3.5 text-right font-bold text-m3-primary">{row.monthlyTotal.toLocaleString()} 回</td>
                <td class="px-5 py-3.5 text-right font-extrabold text-m3-on-surface">{row.fiscalYearCumulativeTotal.toLocaleString()} 回</td>
                <td class="px-6 py-3.5 text-right font-mono text-m3-on-surface-variant">
                  {(row.monthlyLogTotal ?? (row.monthlyTotal + (row.monthlyExitCount ?? 0))).toLocaleString()} 回
                  <span class="text-[11px] text-m3-on-surface-variant/70 ml-1">
                    (入{row.monthlyTotal} / 退{row.monthlyExitCount ?? 0})
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
