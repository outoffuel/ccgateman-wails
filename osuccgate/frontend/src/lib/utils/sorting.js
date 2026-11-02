const collator = new Intl.Collator('ja', { numeric: true });

/**
 * リストを指定キーと方向でソートする
 * @template T
 * @param {T[]} list 
 * @param {string} key 
 * @param {'asc'|'desc'} dir 
 * @returns {T[]}
 */
export function sortList(list, key, dir) {
  if (!list || list.length === 0) return [];
  return [...list].sort((a, b) => {
    let valA = a[key] ?? '';
    let valB = b[key] ?? '';
    let res = 0;
    if (typeof valA === 'number' && typeof valB === 'number') {
      res = valA - valB;
    } else {
      res = collator.compare(String(valA), String(valB));
    }
    return dir === 'asc' ? res : -res;
  });
}
