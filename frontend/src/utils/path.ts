/** Last two segments of a path, for compact display (`/a/b/c/d` becomes `c/d`). */
export function shortPath(path: string): string {
  return path.split('/').filter(Boolean).slice(-2).join('/');
}

/** Last segment of a path, for compact display (`/a/b/c` becomes `c`). */
export function baseName(path: string): string {
  const parts = path.split('/').filter(Boolean);
  return parts.length > 0 ? parts[parts.length - 1]! : path;
}
