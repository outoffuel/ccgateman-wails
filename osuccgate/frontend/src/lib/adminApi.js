import {
  ClearAllLogs as wailsClearAllLogs,
  DeleteUser as wailsDeleteUser,
  ExportFiscalYearLogsCSV as wailsExportFiscalYearLogsCSV,
  ForceExitAllInside as wailsForceExitAllInside,
  GetAllUsers as wailsGetAllUsers,
  GetCurrentInsideUsers as wailsGetCurrentInsideUsers,
  GetDashboardStats as wailsGetDashboardStats,
  GetMonthlyStats as wailsGetMonthlyStats,
  GetRecentLogs as wailsGetRecentLogs,
  ImportLogsCSV as wailsImportLogsCSV,
  ImportUsersCSV as wailsImportUsersCSV,
  SaveUser as wailsSaveUser,
  ToggleFullscreen as wailsToggleFullscreen,
  VerifyPIN as wailsVerifyPIN
} from './wailsjs/go/main/App.js';

function isWailsAvailable() {
  return typeof window !== 'undefined' && Boolean(window.go?.main?.App);
}

async function request(path, options = {}) {
  const headers = new Headers(options.headers || {});
  const pin = sessionStorage.getItem('admin_pin');
  if (pin) headers.set('X-Admin-PIN', pin);

  const response = await fetch(`${window.location.origin}${path}`, { ...options, headers });
  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `HTTP ${response.status}`);
  }
  return response;
}

async function requestJSON(path, options) {
  return (await request(path, options)).json();
}

function selectCSVAndImport(endpoint, label) {
  return new Promise((resolve, reject) => {
    const input = document.createElement('input');
    input.type = 'file';
    input.accept = '.csv,text/csv';
    input.addEventListener('change', async () => {
      const file = input.files?.[0];
      input.remove();
      if (!file) {
        resolve('');
        return;
      }

      try {
        const result = await requestJSON(endpoint, {
          method: 'POST',
          headers: { 'Content-Type': 'text/csv' },
          body: file
        });
        resolve(result.message || `${label}をインポートしました`);
      } catch (error) {
        reject(error);
      }
    }, { once: true });
    input.click();
  });
}

export function VerifyPIN(pin) {
  if (isWailsAvailable()) return wailsVerifyPIN(pin);
  return requestJSON('/api/auth/verify', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ pin })
  }).then(result => result.success);
}

export function GetDashboardStats() {
  return isWailsAvailable() ? wailsGetDashboardStats() : requestJSON('/api/stats');
}

export function GetMonthlyStats() {
  return isWailsAvailable() ? wailsGetMonthlyStats() : requestJSON('/api/monthly-stats');
}

export function GetAllUsers() {
  return isWailsAvailable() ? wailsGetAllUsers() : requestJSON('/api/users');
}

export function GetRecentLogs(limit) {
  return isWailsAvailable()
    ? wailsGetRecentLogs(limit)
    : requestJSON(`/api/logs?limit=${encodeURIComponent(limit)}`);
}

export function GetCurrentInsideUsers() {
  return isWailsAvailable() ? wailsGetCurrentInsideUsers() : requestJSON('/api/inside');
}

export function SaveUser(user) {
  if (isWailsAvailable()) return wailsSaveUser(user);
  return request('/api/users', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(user)
  });
}

export function DeleteUser(cardId) {
  return isWailsAvailable()
    ? wailsDeleteUser(cardId)
    : request(`/api/users/${encodeURIComponent(cardId)}`, { method: 'DELETE' });
}

export function ClearAllLogs() {
  return isWailsAvailable()
    ? wailsClearAllLogs()
    : request('/api/logs', { method: 'DELETE' });
}

export async function ForceExitAllInside() {
  if (isWailsAvailable()) return wailsForceExitAllInside();
  const result = await requestJSON('/api/force-exit', { method: 'POST' });
  return result.count;
}

export async function ExportFiscalYearLogsCSV(year) {
  if (isWailsAvailable()) return wailsExportFiscalYearLogsCSV(year);

  const response = await request(`/api/export/csv?year=${encodeURIComponent(year)}`);
  const blob = await response.blob();
  const filename = `access_logs_${year}_fiscal_year.csv`;
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  link.click();
  URL.revokeObjectURL(url);
  return filename;
}

export function ImportUsersCSV() {
  return isWailsAvailable()
    ? wailsImportUsersCSV()
    : selectCSVAndImport('/api/import/users', '利用者CSV');
}

export function ImportLogsCSV() {
  return isWailsAvailable()
    ? wailsImportLogsCSV()
    : selectCSVAndImport('/api/import/logs', '入退室ログCSV');
}

export function ToggleFullscreen() {
  if (isWailsAvailable()) return wailsToggleFullscreen();
  return document.fullscreenElement
    ? document.exitFullscreen()
    : document.documentElement.requestFullscreen();
}