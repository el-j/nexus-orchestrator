/** Last two segments of a path, for compact display (`/a/b/c/d` becomes `c/d`). */
export function shortPath(path: string): string {
  return path.split('/').filter(Boolean).slice(-2).join('/');
}
