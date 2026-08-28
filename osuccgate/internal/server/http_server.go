package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"osuccgate/internal/db"
	"osuccgate/internal/service"
	"strconv"
	"strings"
	"time"
)

type HTTPServer struct {
	server      *http.Server
	dbManager   *db.DBManager
	gateService *service.GateService
	adminPin    string
	port        int
}

func NewHTTPServer(dbMgr *db.DBManager, gateSvc *service.GateService, port int, pin string) *HTTPServer {
	if pin == "" {
		pin = "1234" // デフォルト管理者PIN
	}
	if port <= 0 {
		port = 8080
	}
	return &HTTPServer{
		dbManager:   dbMgr,
		gateService: gateSvc,
		adminPin:    pin,
		port:        port,
	}
}

func (s *HTTPServer) Start() {
	mux := http.NewServeMux()

	handleWithCORS := func(pattern string, handler http.HandlerFunc) {
		mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Admin-PIN")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}
			handler(w, r)
		})
	}

	// APIルート
	handleWithCORS("/api/auth/verify", s.handleVerifyPIN)
	handleWithCORS("/api/stats", s.handleGetStats)
	handleWithCORS("/api/users", s.handleUsers)
	handleWithCORS("/api/users/", s.handleUserByID)
	handleWithCORS("/api/logs", s.handleLogs)
	handleWithCORS("/api/inside", s.handleInsideUsers)
	handleWithCORS("/api/force-exit", s.handleForceExit)
	handleWithCORS("/api/export/csv", s.handleExportCSV)

	// Web管理者画面 SPA
	mux.HandleFunc("/", s.handleWebAdminSPA)

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		log.Printf("[HTTP Server] Admin Web Server running on http://0.0.0.0:%d", s.port)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[HTTP Server] Error: %v", err)
		}
	}()
}

func (s *HTTPServer) Stop() {
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.server.Shutdown(ctx)
	}
}

func (s *HTTPServer) checkPIN(r *http.Request) bool {
	pin := r.Header.Get("X-Admin-PIN")
	if pin == "" {
		pin = r.URL.Query().Get("pin")
	}
	return pin == s.adminPin
}

func (s *HTTPServer) handleVerifyPIN(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PIN string `json:"pin"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	valid := (body.PIN == s.adminPin)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": valid,
	})
}

func (s *HTTPServer) handleGetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.dbManager.GetDashboardStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *HTTPServer) handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		users, err := s.dbManager.GetAllUsers()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)

	case http.MethodPost:
		if !s.checkPIN(r) {
			http.Error(w, "Unauthorized PIN", http.StatusUnauthorized)
			return
		}
		var u db.RegisteredUser
		if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if u.CardID == "" || u.Name == "" {
			http.Error(w, "CardID and Name are required", http.StatusBadRequest)
			return
		}
		if err := s.dbManager.UpsertUser(&u); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *HTTPServer) handleUserByID(w http.ResponseWriter, r *http.Request) {
	cardID := strings.TrimPrefix(r.URL.Path, "/api/users/")
	if cardID == "" {
		http.Error(w, "Invalid card ID", http.StatusBadRequest)
		return
	}

	if r.Method == http.MethodDelete {
		if !s.checkPIN(r) {
			http.Error(w, "Unauthorized PIN", http.StatusUnauthorized)
			return
		}
		if err := s.dbManager.DeleteUser(cardID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (s *HTTPServer) handleLogs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		limit := 100
		if lStr := r.URL.Query().Get("limit"); lStr != "" {
			if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
				limit = l
			}
		}
		logs, err := s.dbManager.GetRecentLogs(limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(logs)

	case http.MethodDelete:
		if !s.checkPIN(r) {
			http.Error(w, "Unauthorized PIN", http.StatusUnauthorized)
			return
		}
		if err := s.dbManager.ClearAllLogs(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *HTTPServer) handleInsideUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.dbManager.GetCurrentInsideUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func (s *HTTPServer) handleForceExit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.checkPIN(r) {
		http.Error(w, "Unauthorized PIN", http.StatusUnauthorized)
		return
	}
	count, err := s.gateService.ForceExitAllInside()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"count":   count,
	})
}

func (s *HTTPServer) handleExportCSV(w http.ResponseWriter, r *http.Request) {
	yearStr := r.URL.Query().Get("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 2000 {
		year = time.Now().Year()
	}

	csvBytes, err := s.gateService.GenerateFiscalYearCSV(year)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fileName := fmt.Sprintf("access_logs_%d_fiscal_year.csv", year)
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	w.Header().Set("Content-Length", strconv.Itoa(len(csvBytes)))
	w.Write(csvBytes)
}

func (s *HTTPServer) handleWebAdminSPA(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="ja">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>入退室管理システム - 管理者コンソール</title>
<script src="https://cdn.tailwindcss.com"></script>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;600;700;900&family=Noto+Sans+JP:wght@400;500;700&display=swap" rel="stylesheet">
<style>
body { font-family: 'Noto Sans JP', 'Inter', sans-serif; background-color: #0f172a; color: #f8fafc; }
</style>
</head>
<body class="min-h-screen">
  <div id="app"></div>
  <script>
    const API_BASE = window.location.origin;
    let currentPIN = sessionStorage.getItem("admin_pin") || "";
    let stats = { currentInsideCount: 0, totalUserCount: 0, todayLogCount: 0 };
    let logs = [];
    let users = [];
    let insideUsers = [];
    let activeTab = 'logs';

    async function checkAuth() {
      if (!currentPIN) {
        showPinModal();
        return;
      }
      const res = await fetch(API_BASE + '/api/auth/verify', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({pin: currentPIN})
      });
      const data = await res.json();
      if (!data.success) {
        sessionStorage.removeItem("admin_pin");
        currentPIN = "";
        showPinModal();
      } else {
        renderApp();
        loadAllData();
      }
    }

    function showPinModal() {
      const app = document.getElementById("app");
      app.innerHTML = ` + "`" + `
        <div class="fixed inset-0 bg-slate-950/80 backdrop-blur-md flex items-center justify-center p-4">
          <div class="bg-slate-900 border border-slate-700 rounded-2xl p-8 max-w-md w-full shadow-2xl text-center">
            <div class="w-16 h-16 bg-blue-600/20 text-blue-400 rounded-full flex items-center justify-center mx-auto mb-4 text-2xl font-bold">🔒</div>
            <h2 class="text-2xl font-bold text-white mb-2">管理者認証</h2>
            <p class="text-slate-400 text-sm mb-6">4桁の管理者PINコードを入力してください</p>
            <input type="password" id="pinInput" maxlength="8" class="w-full text-center text-3xl tracking-widest bg-slate-800 border border-slate-700 text-white rounded-xl py-3 px-4 focus:outline-none focus:border-blue-500 mb-6 font-mono" placeholder="••••" autofocus>
            <button onclick="submitPin()" class="w-full bg-blue-600 hover:bg-blue-500 text-white font-bold py-3 px-4 rounded-xl transition">ログイン</button>
          </div>
        </div>
      ` + "`" + `;
      document.getElementById("pinInput").addEventListener("keypress", (e) => { if(e.key === 'Enter') submitPin(); });
    }

    async function submitPin() {
      const pin = document.getElementById("pinInput").value;
      const res = await fetch(API_BASE + '/api/auth/verify', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({pin: pin})
      });
      const data = await res.json();
      if (data.success) {
        currentPIN = pin;
        sessionStorage.setItem("admin_pin", pin);
        renderApp();
        loadAllData();
      } else {
        alert("PINコードが正しくありません");
      }
    }

    async function loadAllData() {
      try {
        const [statsRes, logsRes, usersRes, insideRes] = await Promise.all([
          fetch(API_BASE + '/api/stats'),
          fetch(API_BASE + '/api/logs?limit=50'),
          fetch(API_BASE + '/api/users'),
          fetch(API_BASE + '/api/inside')
        ]);
        stats = await statsRes.json();
        logs = await logsRes.json() || [];
        users = await usersRes.json() || [];
        insideUsers = await insideRes.json() || [];
        updateUI();
      } catch(e) {
        console.error(e);
      }
    }

    function renderApp() {
      const app = document.getElementById("app");
      app.innerHTML = ` + "`" + `
        <header class="bg-slate-900 border-b border-slate-800 px-6 py-4 flex flex-wrap items-center justify-between gap-4">
          <div class="flex items-center gap-3">
            <div class="p-2 bg-blue-600/20 text-blue-400 rounded-lg font-bold">OSUG</div>
            <h1 class="text-xl font-bold text-white">入退室管理システム <span class="text-xs bg-slate-800 text-slate-400 px-2 py-1 rounded ml-2">管理者コンソール</span></h1>
          </div>
          <div class="flex items-center gap-3">
            <button onclick="showAdvancedOpsModal()" class="bg-rose-900/50 hover:bg-rose-800 text-rose-300 border border-rose-700/50 px-4 py-2 rounded-lg font-semibold text-sm transition">⚠️ 高度な操作</button>
            <button onclick="loadAllData()" class="bg-slate-800 hover:bg-slate-700 text-slate-200 px-3 py-2 rounded-lg text-sm transition">🔄 更新</button>
            <button onclick="logout()" class="bg-slate-800 hover:bg-slate-700 text-slate-400 px-3 py-2 rounded-lg text-sm transition">ログアウト</button>
          </div>
        </header>

        <main class="max-w-7xl mx-auto p-6 space-y-6">
          <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
            <div class="bg-slate-900/80 border border-slate-800 rounded-2xl p-6 shadow-lg">
              <div class="text-slate-400 text-sm font-medium mb-1">現在の在室者数</div>
              <div class="text-4xl font-extrabold text-blue-400 flex items-center gap-2">
                <span id="statInside">0</span> <span class="text-lg font-normal text-slate-500">名</span>
              </div>
            </div>
            <div class="bg-slate-900/80 border border-slate-800 rounded-2xl p-6 shadow-lg">
              <div class="text-slate-400 text-sm font-medium mb-1">登録ユーザー総数</div>
              <div class="text-4xl font-extrabold text-emerald-400 flex items-center gap-2">
                <span id="statTotal">0</span> <span class="text-lg font-normal text-slate-500">名</span>
              </div>
            </div>
            <div class="bg-slate-900/80 border border-slate-800 rounded-2xl p-6 shadow-lg">
              <div class="text-slate-400 text-sm font-medium mb-1">本日の総打刻数</div>
              <div class="text-4xl font-extrabold text-amber-400 flex items-center gap-2">
                <span id="statToday">0</span> <span class="text-lg font-normal text-slate-500">回</span>
              </div>
            </div>
          </div>

          <div class="flex items-center justify-between border-b border-slate-800 pb-3">
            <div class="flex gap-2">
              <button onclick="setTab('logs')" id="tab-logs" class="px-4 py-2 rounded-lg font-medium text-sm transition bg-blue-600 text-white">入退室ログ</button>
              <button onclick="setTab('inside')" id="tab-inside" class="px-4 py-2 rounded-lg font-medium text-sm transition bg-slate-800 text-slate-400 hover:text-white">在室中リスト</button>
              <button onclick="setTab('users')" id="tab-users" class="px-4 py-2 rounded-lg font-medium text-sm transition bg-slate-800 text-slate-400 hover:text-white">利用者マスター</button>
            </div>
            <div id="tabActions"></div>
          </div>

          <div id="tabContent" class="bg-slate-900/80 border border-slate-800 rounded-2xl p-6 shadow-lg overflow-x-auto">
          </div>
        </main>
        <div id="modalContainer"></div>
      ` + "`" + `;
    }

    function setTab(tab) {
      activeTab = tab;
      ['logs', 'inside', 'users'].forEach(t => {
        const btn = document.getElementById('tab-' + t);
        if (btn) {
          if (t === tab) {
            btn.className = 'px-4 py-2 rounded-lg font-medium text-sm transition bg-blue-600 text-white';
          } else {
            btn.className = 'px-4 py-2 rounded-lg font-medium text-sm transition bg-slate-800 text-slate-400 hover:text-white';
          }
        }
      });
      updateUI();
    }

    function updateUI() {
      if (!document.getElementById("statInside")) return;
      document.getElementById("statInside").innerText = stats.currentInsideCount || 0;
      document.getElementById("statTotal").innerText = stats.totalUserCount || 0;
      document.getElementById("statToday").innerText = stats.todayLogCount || 0;

      const actions = document.getElementById("tabActions");
      const content = document.getElementById("tabContent");

      if (activeTab === 'logs') {
        actions.innerHTML = ` + "`" + `<span class="text-xs text-slate-500">直近50件を表示</span>` + "`" + `;
        if (!logs.length) {
          content.innerHTML = '<div class="text-center py-12 text-slate-500">打刻ログはまだありません</div>';
          return;
        }
        content.innerHTML = ` + "`" + `
          <table class="w-full text-left text-sm text-slate-300">
            <thead class="text-xs text-slate-400 uppercase bg-slate-800/50">
              <tr>
                <th class="p-3">日時</th>
                <th class="p-3">氏名</th>
                <th class="p-3">区分</th>
                <th class="p-3">学籍/職員番号</th>
                <th class="p-3">種別</th>
                <th class="p-3">入力方法</th>
                <th class="p-3">滞在時間</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-800">
              ${logs.map(l => {
                let badge = '<span class="px-2 py-1 rounded-md text-xs font-bold bg-blue-600/30 text-blue-400 border border-blue-500/30">🔵 入室</span>';
                let method = '<span class="text-xs text-slate-400">カード</span>';
                if (l.eventType === 'exit') {
                  badge = '<span class="px-2 py-1 rounded-md text-xs font-bold bg-amber-600/30 text-amber-400 border border-amber-500/30">🟠 退室</span>';
                  method = '<span class="text-xs text-slate-400">カード</span>';
                } else if (l.eventType === 'force_exit') {
                  badge = '<span class="px-2 py-1 rounded-md text-xs font-bold bg-rose-600/30 text-rose-400 border border-rose-500/30">⚠️ 強制退室</span>';
                  method = '<span class="text-xs text-rose-300 font-semibold">システム自動 (23:00)</span>';
                }
                return ` + "`" + `
                  <tr class="hover:bg-slate-800/30">
                    <td class="p-3 font-mono text-slate-400">${new Date(l.timestamp).toLocaleString('ja-JP')}</td>
                    <td class="p-3 font-bold text-white">${l.userName || '未登録'}</td>
                    <td class="p-3"><span class="px-2 py-0.5 rounded text-xs ${l.roleCode === 0 ? 'bg-purple-900/50 text-purple-300 border border-purple-700/50' : (l.roleCode === 9 ? 'bg-emerald-900/50 text-emerald-300 border border-emerald-700/50' : 'bg-slate-800 text-slate-300')}">${l.roleName || '-'}</span></td>
                    <td class="p-3 font-mono">${l.studentNo || '-'}</td>
                    <td class="p-3">${badge}</td>
                    <td class="p-3">${method}</td>
                    <td class="p-3 text-slate-400">${l.durationText || '-'}</td>
                  </tr>
                ` + "`" + `;
              }).join('')}
            </tbody>
          </table>
        ` + "`" + `;
      } else if (activeTab === 'inside') {
        actions.innerHTML = '';
        if (!insideUsers.length) {
          content.innerHTML = '<div class="text-center py-12 text-slate-500">現在在室中のユーザーはいません</div>';
          return;
        }
        content.innerHTML = ` + "`" + `
          <table class="w-full text-left text-sm text-slate-300">
            <thead class="text-xs text-slate-400 uppercase bg-slate-800/50">
              <tr>
                <th class="p-3">氏名</th>
                <th class="p-3">区分</th>
                <th class="p-3">学籍/職員番号</th>
                <th class="p-3">入室時刻</th>
                <th class="p-3">現在の滞在時間</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-800">
              ${insideUsers.map(u => ` + "`" + `
                <tr class="hover:bg-slate-800/30">
                  <td class="p-3 font-bold text-white">${u.userName || '未登録'}</td>
                  <td class="p-3"><span class="px-2 py-0.5 rounded text-xs bg-slate-800 text-slate-300">${u.roleName || '-'}</span></td>
                  <td class="p-3 font-mono">${u.studentNo || '-'}</td>
                  <td class="p-3 font-mono text-slate-400">${new Date(u.lastEventTime).toLocaleTimeString('ja-JP')}</td>
                  <td class="p-3 font-semibold text-blue-400">${u.stayDuration || '数秒'}</td>
                </tr>
              ` + "`" + `).join('')}
            </tbody>
          </table>
        ` + "`" + `;
      } else if (activeTab === 'users') {
        actions.innerHTML = ` + "`" + `<button onclick="openUserModal()" class="bg-blue-600 hover:bg-blue-500 text-white font-medium px-4 py-2 rounded-lg text-sm transition">＋ 新規利用者登録</button>` + "`" + `;
        if (!users.length) {
          content.innerHTML = '<div class="text-center py-12 text-slate-500">登録ユーザーはいません</div>';
          return;
        }
        content.innerHTML = ` + "`" + `
          <table class="w-full text-left text-sm text-slate-300">
            <thead class="text-xs text-slate-400 uppercase bg-slate-800/50">
              <tr>
                <th class="p-3">識別ID (磁気/NFC)</th>
                <th class="p-3">氏名</th>
                <th class="p-3">区分 (コード)</th>
                <th class="p-3">学籍/職員番号</th>
                <th class="p-3 text-right">操作</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-800">
              ${users.map(u => ` + "`" + `
                <tr class="hover:bg-slate-800/30">
                  <td class="p-3 font-mono text-xs text-slate-400">${u.cardId}</td>
                  <td class="p-3 font-bold text-white">${u.name}</td>
                  <td class="p-3"><span class="px-2 py-0.5 rounded text-xs ${u.roleCode === 0 ? 'bg-purple-900/50 text-purple-300' : (u.roleCode === 9 ? 'bg-emerald-900/50 text-emerald-300' : 'bg-slate-800 text-slate-300')}">${u.roleName} (${u.roleCode})</span></td>
                  <td class="p-3 font-mono">${u.studentNo || '-'}</td>
                  <td class="p-3 text-right space-x-2">
                    <button onclick='editUser(${JSON.stringify(u)})' class="text-blue-400 hover:underline">編集</button>
                  </td>
                </tr>
              ` + "`" + `).join('')}
            </tbody>
          </table>
        ` + "`" + `;
      }
    }

    function openUserModal(user = null) {
      const modal = document.getElementById("modalContainer");
      modal.innerHTML = ` + "`" + `
        <div class="fixed inset-0 bg-slate-950/80 backdrop-blur-md flex items-center justify-center p-4 z-50">
          <div class="bg-slate-900 border border-slate-700 rounded-2xl p-6 max-w-lg w-full shadow-2xl">
            <h3 class="text-xl font-bold text-white mb-4">${user ? '利用者情報の編集' : '新規利用者の登録'}</h3>
            <form id="userForm" class="space-y-4" onsubmit="saveUser(event)">
              <div>
                <label class="block text-xs font-semibold text-slate-400 uppercase mb-1">識別ID (磁気カード番号 / NFC IDm) *</label>
                <input type="text" id="formCardId" value="${user ? user.cardId : ''}" ${user ? 'readonly class="w-full bg-slate-800/50 border border-slate-700 text-slate-400 rounded-xl px-3 py-2 font-mono"' : 'class="w-full bg-slate-800 border border-slate-700 text-white rounded-xl px-3 py-2 font-mono" placeholder="カードを通すか入力" required'}>
              </div>
              <div>
                <label class="block text-xs font-semibold text-slate-400 uppercase mb-1">氏名 *</label>
                <input type="text" id="formName" value="${user ? user.name : ''}" class="w-full bg-slate-800 border border-slate-700 text-white rounded-xl px-3 py-2" placeholder="山田 太郎" required>
              </div>
              <div class="grid grid-cols-2 gap-4">
                <div>
                  <label class="block text-xs font-semibold text-slate-400 uppercase mb-1">区分名</label>
                  <select id="formRoleName" class="w-full bg-slate-800 border border-slate-700 text-white rounded-xl px-3 py-2" onchange="onRoleChange()">
                    <option value="教職員" ${user && user.roleName === '教職員' ? 'selected' : ''}>教職員</option>
                    <option value="学生" ${(!user || user.roleName === '学生') ? 'selected' : ''}>学生</option>
                    <option value="学生スタッフ" ${user && user.roleName === '学生スタッフ' ? 'selected' : ''}>学生スタッフ</option>
                  </select>
                </div>
                <div>
                  <label class="block text-xs font-semibold text-slate-400 uppercase mb-1">ロールコード</label>
                  <input type="number" id="formRoleCode" value="${user ? user.roleCode : 1}" readonly class="w-full bg-slate-800/50 border border-slate-700 text-slate-400 rounded-xl px-3 py-2 font-mono">
                </div>
              </div>
              <div>
                <label class="block text-xs font-semibold text-slate-400 uppercase mb-1">学籍番号 / 職員番号</label>
                <input type="text" id="formStudentNo" value="${user ? user.studentNo : ''}" class="w-full bg-slate-800 border border-slate-700 text-white rounded-xl px-3 py-2 font-mono" placeholder="任意">
              </div>
              <div class="flex justify-end gap-3 pt-4 border-t border-slate-800">
                <button type="button" onclick="closeModal()" class="px-4 py-2 rounded-xl text-sm font-medium text-slate-400 hover:text-white">キャンセル</button>
                <button type="submit" class="bg-blue-600 hover:bg-blue-500 text-white font-medium px-5 py-2 rounded-xl text-sm transition">保存</button>
              </div>
            </form>
          </div>
        </div>
      ` + "`" + `;
    }

    function onRoleChange() {
      const role = document.getElementById("formRoleName").value;
      const codeInput = document.getElementById("formRoleCode");
      if (role === '教職員') codeInput.value = 0;
      else if (role === '学生') codeInput.value = 1;
      else if (role === '学生スタッフ') codeInput.value = 9;
      else codeInput.value = 1;
    }

    function editUser(user) {
      openUserModal(user);
    }

    async function saveUser(e) {
      e.preventDefault();
      const payload = {
        cardId: document.getElementById("formCardId").value.trim(),
        name: document.getElementById("formName").value.trim(),
        roleName: document.getElementById("formRoleName").value,
        roleCode: parseInt(document.getElementById("formRoleCode").value, 10),
        studentNo: document.getElementById("formStudentNo").value.trim()
      };
      const res = await fetch(API_BASE + '/api/users', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-Admin-PIN': currentPIN },
        body: JSON.stringify(payload)
      });
      if (res.ok) {
        closeModal();
        loadAllData();
      } else {
        alert("保存に失敗しました");
      }
    }

    function showAdvancedOpsModal() {
      const modal = document.getElementById("modalContainer");
      modal.innerHTML = ` + "`" + `
        <div class="fixed inset-0 bg-slate-950/80 backdrop-blur-md flex items-center justify-center p-4 z-50">
          <div class="bg-slate-900 border border-rose-700/60 rounded-2xl p-6 max-w-lg w-full shadow-2xl">
            <div class="flex items-center gap-3 text-rose-400 mb-4">
              <span class="text-3xl">⚠️</span>
              <h3 class="text-xl font-bold text-white">高度な操作（重要確認）</h3>
            </div>
            <div class="bg-rose-950/40 border border-rose-800/50 rounded-xl p-4 mb-6">
              <p class="text-rose-200 text-sm font-semibold leading-relaxed">
                システムの管理者及びリーダーに了承を得て操作を行ってください。
              </p>
            </div>
            <div class="space-y-4 mb-6" id="advancedPanel">
              <div class="p-4 bg-slate-800/50 rounded-xl border border-slate-700/50">
                <div class="font-bold text-white text-sm mb-1">📊 年度別入退室ログ出力</div>
                <p class="text-xs text-slate-400 mb-3">指定した年度（4/1〜翌3/31）のログをCSV形式でダウンロードします。</p>
                <div class="flex gap-2">
                  <input type="number" id="exportYear" value="${new Date().getFullYear()}" class="bg-slate-900 border border-slate-700 text-white rounded-lg px-3 py-1.5 text-sm w-32">
                  <button onclick="downloadCSV()" class="bg-emerald-600 hover:bg-emerald-500 text-white text-sm font-medium px-4 py-1.5 rounded-lg transition">CSVダウンロード</button>
                </div>
              </div>

              <div class="p-4 bg-slate-800/50 rounded-xl border border-slate-700/50">
                <div class="font-bold text-amber-400 text-sm mb-1">🚪 即時一斉強制退室 (手動実行)</div>
                <p class="text-xs text-slate-400 mb-3">現在在室中のすべてのユーザーを即座に「強制退室」として記録します。</p>
                <button onclick="forceExitAction()" class="bg-amber-700 hover:bg-amber-600 text-white text-sm font-bold px-4 py-2 rounded-lg transition w-full">在室者の一括強制退室を実行</button>
              </div>

              <div class="p-4 bg-slate-800/50 rounded-xl border border-slate-700/50">
                <div class="font-bold text-rose-400 text-sm mb-1">🗑️ 登録者の個別削除</div>
                <p class="text-xs text-slate-400 mb-2">削除する利用者のカードIDを入力してください。</p>
                <div class="flex gap-2">
                  <input type="text" id="deleteCardId" placeholder="カードID" class="bg-slate-900 border border-slate-700 text-white rounded-lg px-3 py-1.5 text-sm flex-1 font-mono">
                  <button onclick="deleteUserAction()" class="bg-rose-600 hover:bg-rose-500 text-white text-sm font-medium px-4 py-1.5 rounded-lg transition">削除実行</button>
                </div>
              </div>

              <div class="p-4 bg-slate-800/50 rounded-xl border border-slate-700/50">
                <div class="font-bold text-rose-400 text-sm mb-1">💣 全入退室ログの完全削除</div>
                <p class="text-xs text-slate-400 mb-3">蓄積されたすべての入退室履歴を削除します（取り消せません）。</p>
                <button onclick="clearLogsAction()" class="bg-rose-900 hover:bg-rose-800 border border-rose-700 text-rose-200 text-sm font-bold px-4 py-2 rounded-lg transition w-full">全ログ削除を実行</button>
              </div>
            </div>
            <div class="flex justify-end pt-2">
              <button onclick="closeModal()" class="px-5 py-2 bg-slate-800 hover:bg-slate-700 text-white font-medium rounded-xl text-sm transition">閉じる（中止）</button>
            </div>
          </div>
        </div>
      ` + "`" + `;
    }

    function downloadCSV() {
      const year = document.getElementById("exportYear").value || 2026;
      window.open(API_BASE + '/api/export/csv?year=' + year + '&pin=' + encodeURIComponent(currentPIN), '_blank');
    }

    async function forceExitAction() {
      if (!confirm("現在在室中のすべてのユーザーを強制退室させますか？")) return;
      const res = await fetch(API_BASE + '/api/force-exit', {
        method: 'POST',
        headers: { 'X-Admin-PIN': currentPIN }
      });
      if (res.ok) {
        const data = await res.json();
        alert(data.count + " 名のユーザーを強制退室処理しました");
        closeModal();
        loadAllData();
      } else {
        alert("強制退室に失敗しました");
      }
    }

    async function deleteUserAction() {
      const cardId = document.getElementById("deleteCardId").value.trim();
      if (!cardId) return alert("カードIDを入力してください");
      if (!confirm("本当にカードID: " + cardId + " を削除しますか？")) return;
      const res = await fetch(API_BASE + '/api/users/' + encodeURIComponent(cardId), {
        method: 'DELETE',
        headers: { 'X-Admin-PIN': currentPIN }
      });
      if (res.ok) {
        alert("削除しました");
        closeModal();
        loadAllData();
      } else {
        alert("削除に失敗しました");
      }
    }

    async function clearLogsAction() {
      if (!confirm("警告: すべての入退室履歴ログを削除します。よろしいですか？")) return;
      const res = await fetch(API_BASE + '/api/logs', {
        method: 'DELETE',
        headers: { 'X-Admin-PIN': currentPIN }
      });
      if (res.ok) {
        alert("全ログを削除しました");
        closeModal();
        loadAllData();
      } else {
        alert("削除に失敗しました");
      }
    }

    function closeModal() {
      document.getElementById("modalContainer").innerHTML = '';
    }

    function logout() {
      sessionStorage.removeItem("admin_pin");
      currentPIN = "";
      checkAuth();
    }

    setInterval(() => {
      if (currentPIN) loadAllData();
    }, 10000);

    checkAuth();
  </script>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}
