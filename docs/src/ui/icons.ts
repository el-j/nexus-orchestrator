/**
 * Icon set: 24x24 stroke glyphs (path data only).
 * Every entry is a list of SVG path `d` strings drawn with a 2px round stroke.
 */
const circle = 'M12 2a10 10 0 1 0 0 20 10 10 0 0 0 0-20z';

/** Path data of every available icon, keyed by icon name. */
export const ICONS = {
  'arrow-right': ['M5 12h14', 'm12 5 7 7-7 7'],
  book: [
    'M4 19.5A2.5 2.5 0 0 1 6.5 17H20',
    'M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z',
  ],
  check: ['M20 6 9 17l-5-5'],
  'check-circle': [circle, 'm9 12 2 2 4-4'],
  code: ['m16 18 6-6-6-6', 'm8 6-6 6 6 6'],
  download: ['M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4', 'm7 10 5 5 5-5', 'M12 15V3'],
  'exclamation-circle': [circle, 'M12 8v4M12 16h.01'],
  'exclamation-triangle': [
    'm21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3',
    'M12 9v4M12 17h.01',
  ],
  'info-circle': [circle, 'M12 16v-4M12 8h.01'],
  link: [
    'M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71',
    'M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71',
  ],
  server: [
    'M4 3h16a1 1 0 0 1 1 1v6a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1z',
    'M4 13h16a1 1 0 0 1 1 1v6a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1v-6a1 1 0 0 1 1-1z',
    'M7 7.5h.01M7 17.5h.01',
  ],
  spinner: ['M21 12a9 9 0 1 1-6.219-8.56'],
  tag: [
    'M12.59 2.59A2 2 0 0 0 11.17 2H4a2 2 0 0 0-2 2v7.17a2 2 0 0 0 .59 1.42l8.7 8.7a2 2 0 0 0 2.82 0l7.18-7.18a2 2 0 0 0 0-2.82z',
    'M7 7h.01',
  ],
  times: ['M18 6 6 18M6 6l12 12'],
  bars: ['M4 6h16M4 12h16M4 18h16'],
  github: [
    'M15 22v-4a4.8 4.8 0 0 0-1-3.5c3 0 6-2 6-5.5.08-1.25-.27-2.48-1-3.5.28-1.15.28-2.35 0-3.5 0 0-1 0-3 1.5-2.64-.5-5.36-.5-8 0C6 2 5 2 5 2c-.3 1.15-.3 2.35 0 3.5A5.403 5.403 0 0 0 4 9c0 3.5 3 5.5 6 5.5-.39.49-.68 1.05-.85 1.65-.17.6-.22 1.23-.15 1.85v4',
    'M9 18c-4.51 2-5-2-7-2',
  ],
  sitemap: [
    'M10 3h4v5h-4z',
    'M3 16h4v5H3z',
    'M10 16h4v5h-4z',
    'M17 16h4v5h-4z',
    'M12 8v4M5 16v-4h14v4M12 12v4',
  ],
} as const satisfies Record<string, readonly string[]>;

/** Name of an icon in {@link ICONS}. */
export type IconName = keyof typeof ICONS;
