/**
 * Icon set: 24x24 stroke glyphs (path data only).
 * Every entry is a list of SVG path `d` strings drawn with a 2px round stroke.
 */
const circle = 'M12 2a10 10 0 1 0 0 20 10 10 0 0 0 0-20z';
const file = ['M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z', 'M14 2v6h6'];

/** Path data of every available icon, keyed by icon name. */
export const ICONS = {
  'arrow-right': ['M5 12h14', 'm12 5 7 7-7 7'],
  bookmark: ['m19 21-7-4-7 4V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z'],
  check: ['M20 6 9 17l-5-5'],
  'check-circle': [circle, 'm9 12 2 2 4-4'],
  'chevron-down': ['m6 9 6 6 6-6'],
  'chevron-right': ['m9 18 6-6-6-6'],
  'chevron-up': ['m18 15-6-6-6 6'],
  clock: [circle, 'M12 6v6l4 2'],
  cog: [
    'M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8z',
    'M12 2v3M12 19v3M2 12h3M19 12h3M4.93 4.93l2.12 2.12M16.95 16.95l2.12 2.12M4.93 19.07l2.12-2.12M16.95 7.05l2.12-2.12',
  ],
  database: [
    'M12 3c4.97 0 9 1.34 9 3s-4.03 3-9 3-9-1.34-9-3 4.03-3 9-3z',
    'M3 6v6c0 1.66 4.03 3 9 3s9-1.34 9-3V6',
    'M3 12v6c0 1.66 4.03 3 9 3s9-1.34 9-3v-6',
  ],
  'exclamation-circle': [circle, 'M12 8v4M12 16h.01'],
  'exclamation-triangle': [
    'm21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3',
    'M12 9v4M12 17h.01',
  ],
  file,
  'file-edit': [...file, 'm10 17 5-5 2 2-5 5-3 1z'],
  filter: ['M22 3H2l8 9.46V19l4 2v-8.54z'],
  folder: [
    'M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2z',
  ],
  globe: [
    circle,
    'M2 12h20',
    'M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z',
  ],
  home: ['M3 9.5 12 3l9 6.5V20a1 1 0 0 1-1 1h-5v-7H9v7H4a1 1 0 0 1-1-1z'],
  inbox: [
    'M22 12h-6l-2 3h-4l-2-3H2',
    'M5.45 5.11 2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z',
  ],
  list: ['M8 6h13M8 12h13M8 18h13M3 6h.01M3 12h.01M3 18h.01'],
  'plus-circle': [circle, 'M12 8v8M8 12h8'],
  'question-circle': [circle, 'M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3', 'M12 17h.01'],
  search: ['M11 3a8 8 0 1 0 0 16 8 8 0 0 0 0-16z', 'm21 21-4.3-4.3'],
  send: ['m22 2-7 20-4-9-9-4z', 'M22 2 11 13'],
  server: [
    'M4 3h16a1 1 0 0 1 1 1v6a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1z',
    'M4 13h16a1 1 0 0 1 1 1v6a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1v-6a1 1 0 0 1 1-1z',
    'M7 7.5h.01M7 17.5h.01',
  ],
  spinner: ['M21 12a9 9 0 1 1-6.219-8.56'],
  stop: ['M6 6h12v12H6z'],
  sync: [
    'M21 12a9 9 0 0 0-9-9 9.75 9.75 0 0 0-6.74 2.74L3 8',
    'M3 3v5h5',
    'M3 12a9 9 0 0 0 9 9 9.75 9.75 0 0 0 6.74-2.74L21 16',
    'M16 16h5v5',
  ],
  terminal: ['m4 17 6-6-6-6', 'M12 19h8'],
  times: ['M18 6 6 18M6 6l12 12'],
  users: [
    'M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2',
    'M9 3a4 4 0 1 0 0 8 4 4 0 0 0 0-8z',
    'M22 21v-2a4 4 0 0 0-3-3.87',
    'M16 3.13a4 4 0 0 1 0 7.75',
  ],
  wifi: [
    'M5 12.55a11 11 0 0 1 14.08 0',
    'M1.42 9a16 16 0 0 1 21.16 0',
    'M8.53 16.11a6 6 0 0 1 6.95 0',
    'M12 20h.01',
  ],
} as const satisfies Record<string, readonly string[]>;

/** Name of an icon in {@link ICONS}. */
export type IconName = keyof typeof ICONS;
