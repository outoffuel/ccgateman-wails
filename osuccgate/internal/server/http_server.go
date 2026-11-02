package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"osuccgate/internal/db"
	"osuccgate/internal/service"
	"osuccgate/internal/updater"
	"strconv"
	"strings"
	"time"
)

type HTTPServer struct {
	server      *http.Server
	dbManager   *db.DBManager
	gateService *service.GateService
	updater     *updater.Updater
	adminPin    string
	appVersion  string
	port        int
	webAssets   fs.FS
}

func NewHTTPServer(dbMgr *db.DBManager, gateSvc *service.GateService, port int, pin string, webAssets fs.FS, upd *updater.Updater, version string) *HTTPServer {
	if pin == "" {
		pin = "8202"
	}
	if port <= 0 {
		port = 8080
	}
	if version == "" {
		version = "dev"
	}
	return &HTTPServer{
		dbManager:   dbMgr,
		gateService: gateSvc,
		updater:     upd,
		adminPin:    pin,
		appVersion:  version,
		port:        port,
		webAssets:   webAssets,
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
	handleWithCORS("/api/fiscal-years", s.handleGetFiscalYears)
	handleWithCORS("/api/stats", s.handleGetStats)
	handleWithCORS("/api/monthly-stats", s.handleGetMonthlyStats)
	handleWithCORS("/api/users", s.handleUsers)
	handleWithCORS("/api/users/find-past", s.handleFindPastUser)
	handleWithCORS("/api/users/", s.handleUserByID)
	handleWithCORS("/api/logs", s.handleLogs)
	handleWithCORS("/api/inside", s.handleInsideUsers)
	handleWithCORS("/api/force-exit", s.handleForceExit)
	handleWithCORS("/api/export/csv", s.handleExportCSV)
	handleWithCORS("/api/import/users", s.handleImportUsers)
	handleWithCORS("/api/import/logs", s.handleImportLogs)
	handleWithCORS("/api/version", s.handleGetVersion)
	handleWithCORS("/api/system/update-status", s.handleGetVersion)
	handleWithCORS("/api/system/check-update", s.handleCheckUpdate)
	handleWithCORS("/api/system/apply-update", s.handleApplyUpdate)

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

func (s *HTTPServer) handleGetFiscalYears(w http.ResponseWriter, r *http.Request) {
	if !s.checkPIN(r) {
		http.Error(w, "Unauthorized PIN", http.StatusUnauthorized)
		return
	}
	years, err := s.dbManager.GetAvailableFiscalYears()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(years)
}

func (s *HTTPServer) handleFindPastUser(w http.ResponseWriter, r *http.Request) {
	if !s.checkPIN(r) {
		http.Error(w, "Unauthorized PIN", http.StatusUnauthorized)
		return
	}
	identifier := r.URL.Query().Get("identifier")
	if identifier == "" {
		identifier = r.URL.Query().Get("cardId")
	}
	if identifier == "" {
		http.Error(w, "Identifier is required", http.StatusBadRequest)
		return
	}
	u, err := s.dbManager.FindUserInAnyFiscalYear(identifier)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(u)
}

func (s *HTTPServer) handleGetStats(w http.ResponseWriter, r *http.Request) {
	if !s.checkPIN(r) {
		http.Error(w, "Unauthorized PIN", http.StatusUnauthorized)
		return
	}
	var fy int
	if fyStr := r.URL.Query().Get("fiscalYear"); fyStr != "" {
		fy, _ = strconv.Atoi(fyStr)
	} else if yStr := r.URL.Query().Get("year"); yStr != "" {
		fy, _ = strconv.Atoi(yStr)
	}

	stats, err := s.dbManager.GetDashboardStats(fy)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *HTTPServer) handleGetMonthlyStats(w http.ResponseWriter, r *http.Request) {
	if !s.checkPIN(r) {
		http.Error(w, "Unauthorized PIN", http.StatusUnauthorized)
		return
	}
	var fy int
	if fyStr := r.URL.Query().Get("fiscalYear"); fyStr != "" {
		fy, _ = strconv.Atoi(fyStr)
	} else if yStr := r.URL.Query().Get("year"); yStr != "" {
		fy, _ = strconv.Atoi(yStr)
	}

	stats, err := s.dbManager.GetMonthlyStats(fy)
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
		if !s.checkPIN(r) {
			http.Error(w, "Unauthorized PIN", http.StatusUnauthorized)
			return
		}
		var fy int
		if fyStr := r.URL.Query().Get("fiscalYear"); fyStr != "" {
			fy, _ = strconv.Atoi(fyStr)
		} else if yStr := r.URL.Query().Get("year"); yStr != "" {
			fy, _ = strconv.Atoi(yStr)
		}
		users, err := s.dbManager.GetAllUsers(fy)
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
		var fy int
		if fyStr := r.URL.Query().Get("fiscalYear"); fyStr != "" {
			fy, _ = strconv.Atoi(fyStr)
		} else if yStr := r.URL.Query().Get("year"); yStr != "" {
			fy, _ = strconv.Atoi(yStr)
		}
		if err := s.dbManager.DeleteUser(cardID, fy); err != nil {
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
		if !s.checkPIN(r) {
			http.Error(w, "Unauthorized PIN", http.StatusUnauthorized)
			return
		}
		limit := 100
		if lStr := r.URL.Query().Get("limit"); lStr != "" {
			if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
				limit = l
			}
		}
		var fy int
		if fyStr := r.URL.Query().Get("fiscalYear"); fyStr != "" {
			fy, _ = strconv.Atoi(fyStr)
		} else if yStr := r.URL.Query().Get("year"); yStr != "" {
			fy, _ = strconv.Atoi(yStr)
		}
		logs, err := s.dbManager.GetRecentLogs(limit, fy)
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
	if !s.checkPIN(r) {
		http.Error(w, "Unauthorized PIN", http.StatusUnauthorized)
		return
	}
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
	if !s.checkPIN(r) {
		http.Error(w, "Unauthorized PIN", http.StatusUnauthorized)
		return
	}
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

func (s *HTTPServer) handleImportUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.checkPIN(r) {
		http.Error(w, "Unauthorized PIN", http.StatusUnauthorized)
		return
	}

	data, err := readUploadedCSV(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	imported, total, err := s.gateService.ImportUsersCSV(data)
	writeImportResponse(w, imported, total, err, "利用者")
}

func (s *HTTPServer) handleImportLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.checkPIN(r) {
		http.Error(w, "Unauthorized PIN", http.StatusUnauthorized)
		return
	}

	data, err := readUploadedCSV(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	imported, total, err := s.gateService.ImportLogsCSV(data)
	writeImportResponse(w, imported, total, err, "入退室ログ")
}

func readUploadedCSV(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		file, _, err := r.FormFile("file")
		if err != nil {
			return nil, fmt.Errorf("File not provided: %w", err)
		}
		defer file.Close()
		return io.ReadAll(file)
	}
	return io.ReadAll(r.Body)
}

func writeImportResponse(w http.ResponseWriter, imported, total int, err error, label string) {
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"imported": imported,
		"total":    total,
		"message":  fmt.Sprintf("%d 件の%sを登録・更新しました", imported, label),
	})
}

func (s *HTTPServer) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (s *HTTPServer) handleWebAdminSPA(w http.ResponseWriter, r *http.Request) {
	distFS, err := fs.Sub(s.webAssets, "frontend/dist")
	if err != nil {
		http.Error(w, "Admin frontend is unavailable", http.StatusInternalServerError)
		return
	}
	http.FileServer(http.FS(distFS)).ServeHTTP(w, r)
}

func (s *HTTPServer) handleGetVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	s.jsonResponse(w, http.StatusOK, map[string]string{
		"version": s.appVersion,
	})
}

func (s *HTTPServer) handleCheckUpdate(w http.ResponseWriter, r *http.Request) {
	if !s.checkPIN(r) {
		http.Error(w, "Unauthorized: Invalid PIN", http.StatusUnauthorized)
		return
	}
	if s.updater == nil {
		s.jsonResponse(w, http.StatusServiceUnavailable, map[string]string{
			"error": "Updater is not configured",
		})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	info, err := s.updater.CheckForUpdate(ctx)
	if err != nil {
		s.jsonResponse(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}
	s.jsonResponse(w, http.StatusOK, info)
}

func (s *HTTPServer) handleApplyUpdate(w http.ResponseWriter, r *http.Request) {
	if !s.checkPIN(r) {
		http.Error(w, "Unauthorized: Invalid PIN", http.StatusUnauthorized)
		return
	}
	if s.updater == nil {
		s.jsonResponse(w, http.StatusServiceUnavailable, map[string]string{
			"error": "Updater is not configured",
		})
		return
	}

	checkCtx, checkCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer checkCancel()

	info, err := s.updater.CheckForUpdate(checkCtx)
	if err != nil {
		s.jsonResponse(w, http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to check release: %v", err),
		})
		return
	}
	if !info.HasUpdate || info.AssetURL == "" {
		s.jsonResponse(w, http.StatusBadRequest, map[string]string{
			"error": "No update asset available for this system",
		})
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{
		"message": "Update started. The application will restart automatically.",
	})

	go func() {
		time.Sleep(500 * time.Millisecond)
		updateCtx, updateCancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer updateCancel()
		if err := s.updater.ApplyUpdate(updateCtx, info); err != nil {
			log.Printf("[HTTP Server] Error applying update: %v", err)
		}
	}()
}
