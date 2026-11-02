/**
 * ページ番号を許容範囲内 (1 〜 totalPages) に収める
 * @param {number|string} page 
 * @param {number} totalPages 
 * @returns {number}
 */
export function clampPage(page, totalPages) {
  const parsedPage = Number.parseInt(String(page), 10);
  if (!Number.isFinite(parsedPage)) return 1;
  return Math.min(Math.max(parsedPage, 1), Math.max(totalPages, 1));
}

/**
 * ページネーションのページ番号配列（省略記号を含む）を生成する
 * @param {number} totalPages 
 * @param {number} currentPage 
 * @returns {Array<{type: 'page'|'ellipsis', page?: number, key: string}>}
 */
export function getPageItems(totalPages, currentPage) {
  const pages = [...new Set([
    1,
    2,
    currentPage - 1,
    currentPage,
    currentPage + 1,
    totalPages - 1,
    totalPages
  ].filter(page => page >= 1 && page <= totalPages))].sort((a, b) => a - b);

  return pages.flatMap((page, index) => {
    const previousPage = pages[index - 1];
    const items = [];
    if (previousPage && page - previousPage > 1) {
      items.push({ type: 'ellipsis', key: `ellipsis-${previousPage}-${page}` });
    }
    items.push({ type: 'page', page, key: `page-${page}` });
    return items;
  });
}

