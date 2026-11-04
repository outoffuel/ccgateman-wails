import {
  ApplyUpdate as wailsApplyUpdate,
  CheckUpdate as wailsCheckUpdate,
  ClearAllLogs as wailsClearAllLogs,
  DeleteUser as wailsDeleteUser,
  ExportFiscalYearLogsCSV as wailsExportFiscalYearLogsCSV,
  FindUserInAnyFiscalYear as wailsFindUserInAnyFiscalYear,
  ForceExitAllInside as wailsForceExitAllInside,
  GetAllUsers as wailsGetAllUsers,
  GetAppVersion as wailsGetAppVersion,
  GetAvailableFiscalYears as wailsGetAvailableFiscalYears,
  GetCurrentFiscalYear as wailsGetCurrentFiscalYear,
  GetCurrentInsideUsers as wailsGetCurrentInsideUsers,
  GetDashboardStats as wailsGetDashboardStats,
  GetMonthlyStats as wailsGetMonthlyStats,
  GetRecentLogs as wailsGetRecentLogs,
  ImportLogsCSV as wailsImportLogsCSV,
  ImportUsersCSV as wailsImportUsersCSV,
  SaveCSVContent as wailsSaveCSVContent,
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

export function GetAvailableFiscalYears() {
  return isWailsAvailable()
    ? wailsGetAvailableFiscalYears()
    : requestJSON('/api/fiscal-years');
}

export function GetCurrentFiscalYear() {
  if (isWailsAvailable()) {
    return wailsGetCurrentFiscalYear();
  }
  const now = new Date();
  const y = now.getFullYear();
  return now.getMonth() < 3 ? y - 1 : y;
}

export function FindUserInAnyFiscalYear(identifier) {
  return isWailsAvailable()
    ? wailsFindUserInAnyFiscalYear(identifier)
    : requestJSON(`/api/users/find-past?identifier=${encodeURIComponent(identifier)}`);
}

export function GetDashboardStats(fiscalYear = 0) {
  return isWailsAvailable()
    ? wailsGetDashboardStats(fiscalYear)
    : requestJSON(`/api/stats${fiscalYear ? `?fiscalYear=${encodeURIComponent(fiscalYear)}` : ''}`);
}

export function GetMonthlyStats(fiscalYear = 0) {
  return isWailsAvailable()
    ? wailsGetMonthlyStats(fiscalYear)
    : requestJSON(`/api/monthly-stats${fiscalYear ? `?fiscalYear=${encodeURIComponent(fiscalYear)}` : ''}`);
}

export function GetAllUsers(fiscalYear = 0) {
  return isWailsAvailable()
    ? wailsGetAllUsers(fiscalYear)
    : requestJSON(`/api/users${fiscalYear ? `?fiscalYear=${encodeURIComponent(fiscalYear)}` : ''}`);
}

export function GetRecentLogs(limit, fiscalYear = 0) {
  const fyParam = fiscalYear ? `&fiscalYear=${encodeURIComponent(fiscalYear)}` : '';
  return isWailsAvailable()
    ? wailsGetRecentLogs(limit, fiscalYear)
    : requestJSON(`/api/logs?limit=${encodeURIComponent(limit)}${fyParam}`);
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

export function DeleteUser(cardId, fiscalYear = 0) {
  const fyParam = fiscalYear ? `?fiscalYear=${encodeURIComponent(fiscalYear)}` : '';
  return isWailsAvailable()
    ? wailsDeleteUser(cardId, fiscalYear)
    : request(`/api/users/${encodeURIComponent(cardId)}${fyParam}`, { method: 'DELETE' });
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

export async function SaveCSVContent(defaultFilename, content) {
  if (isWailsAvailable()) {
    return wailsSaveCSVContent(defaultFilename, content);
  }
  const utf8BOM = '\uFEFF';
  const csvWithBOM = content.startsWith(utf8BOM) ? content : utf8BOM + content;
  const blob = new Blob([csvWithBOM], { type: 'text/csv;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.setAttribute('href', url);
  link.setAttribute('download', defaultFilename);
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
  return defaultFilename;
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

export function GetAppVersion() {
  if (isWailsAvailable()) return wailsGetAppVersion();
  return requestJSON('/api/version').then(res => res.version).catch(() => 'dev');
}

export function CheckUpdate() {
  if (isWailsAvailable()) return wailsCheckUpdate();
  return requestJSON('/api/system/check-update', { method: 'POST' });
}

export function ApplyUpdate() {
  if (isWailsAvailable()) return wailsApplyUpdate();
  return requestJSON('/api/system/apply-update', { method: 'POST' });
}
