/**
 * Smooth-scrolls to the element with the given id.
 * Used by sidebar TOC links on sub-page views so that clicking an in-page anchor
 * does not overwrite the Vue Router hash (which would cause the RouterView to
 * render nothing when no route matches the fragment id).
 */
export function scrollToId(id: string): void {
  const el = document.getElementById(id)
  if (!el) {
    console.warn(`[scrollToId] element #${id} not found`)
    return
  }
  el.scrollIntoView({ behavior: 'smooth' })
}
