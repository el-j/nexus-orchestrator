/**
 * nexusOrchestrator — main.js
 * Shared frontend JS for all documentation pages
 */

/* ── Navigation ─────────────────────────────────────────────── */
(function initNav() {
  const navbar = document.querySelector('.navbar');
  const hamburger = document.querySelector('.nav-hamburger');
  const mobileMenu = document.querySelector('.mobile-menu');

  // Scroll shadow on navbar
  if (navbar) {
    window.addEventListener('scroll', () => {
      navbar.classList.toggle('scrolled', window.scrollY > 20);
    }, { passive: true });
  }

  // Mobile menu toggle
  if (hamburger && mobileMenu) {
    hamburger.addEventListener('click', () => {
      const isOpen = mobileMenu.classList.toggle('open');
      hamburger.classList.toggle('open', isOpen);
      hamburger.setAttribute('aria-expanded', String(isOpen));
    });

    // Close menu on link click
    mobileMenu.querySelectorAll('a').forEach(a => {
      a.addEventListener('click', () => {
        mobileMenu.classList.remove('open');
        hamburger.classList.remove('open');
        hamburger.setAttribute('aria-expanded', 'false');
      });
    });
  }

  // Close mobile menu on outside click
  document.addEventListener('click', (e) => {
    if (mobileMenu && mobileMenu.classList.contains('open')) {
      if (!mobileMenu.contains(e.target) && !hamburger.contains(e.target)) {
        mobileMenu.classList.remove('open');
        hamburger.classList.remove('open');
        hamburger.setAttribute('aria-expanded', 'false');
      }
    }
  });
})();

/* ── Active Nav Link Detection ──────────────────────────────── */
(function setActiveNav() {
  const path = window.location.pathname;
  const filename = path.split('/').pop() || 'index.html';

  const allLinks = document.querySelectorAll('.navbar-nav a, .mobile-menu a');
  allLinks.forEach(link => {
    const href = link.getAttribute('href') || '';
    const linkFile = href.replace('./', '').split('/').pop() || 'index.html';

    // Exact match or index default
    if (
      linkFile === filename ||
      (filename === '' && linkFile === 'index.html') ||
      (filename === 'index.html' && linkFile === 'index.html')
    ) {
      link.classList.add('active');
      link.setAttribute('aria-current', 'page');
    }
  });
})();

/* ── Scroll Reveal (IntersectionObserver) ───────────────────── */
(function initScrollReveal() {
  const elements = document.querySelectorAll('.animate-on-scroll');
  if (!elements.length) return;

  if (!('IntersectionObserver' in window)) {
    // Fallback: show all
    elements.forEach(el => el.classList.add('visible'));
    return;
  }

  const observer = new IntersectionObserver((entries) => {
    entries.forEach(entry => {
      if (entry.isIntersecting) {
        entry.target.classList.add('visible');
        observer.unobserve(entry.target);
      }
    });
  }, {
    threshold: 0.1,
    rootMargin: '0px 0px -40px 0px'
  });

  elements.forEach(el => observer.observe(el));
})();

/* ── Smooth Scroll for Anchor Links ─────────────────────────── */
(function initSmoothScroll() {
  document.querySelectorAll('a[href^="#"]').forEach(anchor => {
    anchor.addEventListener('click', function(e) {
      const targetId = this.getAttribute('href').slice(1);
      if (!targetId) return;
      const target = document.getElementById(targetId);
      if (!target) return;
      e.preventDefault();
      const navHeight = parseInt(getComputedStyle(document.documentElement)
        .getPropertyValue('--nav-height')) || 64;
      const top = target.getBoundingClientRect().top + window.scrollY - navHeight - 16;
      window.scrollTo({ top, behavior: 'smooth' });
    });
  });
})();

/* ── Copy-to-Clipboard for Code Blocks ──────────────────────── */
(function initCopyButtons() {
  document.querySelectorAll('pre').forEach(pre => {
    // Skip if already has a copy button
    if (pre.querySelector('.copy-btn')) return;

    const btn = document.createElement('button');
    btn.className = 'copy-btn';
    btn.textContent = 'Copy';
    btn.setAttribute('aria-label', 'Copy code to clipboard');

    // Make pre position:relative for absolute copy button
    pre.style.position = 'relative';
    pre.appendChild(btn);

    btn.addEventListener('click', async () => {
      const code = pre.querySelector('code');
      const text = code ? code.innerText : pre.innerText;
      // Remove the "Copy" button text from what we copy
      const clean = text.replace(/\nCopy$/, '').trim();

      try {
        await navigator.clipboard.writeText(clean);
        btn.textContent = '✓ Copied';
        btn.classList.add('copied');
        setTimeout(() => {
          btn.textContent = 'Copy';
          btn.classList.remove('copied');
        }, 2000);
      } catch {
        // Fallback for older browsers
        const ta = document.createElement('textarea');
        ta.value = clean;
        ta.style.cssText = 'position:absolute;left:-9999px;top:-9999px';
        document.body.appendChild(ta);
        ta.select();
        document.execCommand('copy');
        document.body.removeChild(ta);
        btn.textContent = '✓ Copied';
        btn.classList.add('copied');
        setTimeout(() => {
          btn.textContent = 'Copy';
          btn.classList.remove('copied');
        }, 2000);
      }
    });
  });
})();

/* ── OS Detection ───────────────────────────────────────────── */
window.detectOS = function detectOS() {
  const ua = navigator.userAgent.toLowerCase();
  const platform = (navigator.platform || '').toLowerCase();

  if (/iphone|ipad/.test(ua)) return 'ios';
  if (/android/.test(ua)) return 'android';
  if (/mac/.test(platform) || /macintosh/.test(ua)) {
    // Detect Apple Silicon vs Intel via canvas heuristic
    const canvas = document.createElement('canvas');
    const gl = canvas.getContext('webgl');
    if (gl) {
      const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
      if (debugInfo) {
        const renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
        if (/apple m\d/i.test(renderer)) return 'mac-arm';
      }
    }
    return 'mac-intel';
  }
  if (/win/.test(platform) || /windows/.test(ua)) return 'windows';
  if (/linux/.test(platform) || /linux/.test(ua)) return 'linux';
  return 'unknown';
};

/* ── Downloads Page — OS Banner ─────────────────────────────── */
(function initOSDetection() {
  const banner = document.getElementById('detected-os-banner');
  if (!banner) return;

  const os = window.detectOS();
  const osNames = {
    'mac-arm':    'macOS (Apple Silicon)',
    'mac-intel':  'macOS (Intel)',
    'windows':    'Windows',
    'linux':      'Linux',
    'unknown':    null
  };

  const name = osNames[os];
  if (!name) return;

  banner.innerHTML = `
    <span style="font-size:1.5rem">${os.startsWith('mac') ? '🍎' : os === 'windows' ? '🪟' : '🐧'}</span>
    <div>
      <strong>We detected ${name}</strong> — the recommended download for your system is highlighted below.
    </div>
  `;
  banner.style.display = 'flex';

  // Highlight matching cards
  document.querySelectorAll('[data-os]').forEach(card => {
    if (card.dataset.os === os || (os.startsWith('mac') && card.dataset.os === 'mac')) {
      card.classList.add('recommended');
      const rec = card.querySelector('.rec-badge');
      if (rec) rec.style.display = 'inline-flex';
    }
  });
})();

/* ── Tab Switcher ───────────────────────────────────────────── */
(function initTabs() {
  document.querySelectorAll('.tabs').forEach(tabGroup => {
    const buttons = tabGroup.querySelectorAll('.tab-btn');
    const contentContainer = tabGroup.nextElementSibling;
    if (!contentContainer) return;

    buttons.forEach(btn => {
      btn.addEventListener('click', () => {
        buttons.forEach(b => {
          b.classList.remove('active');
          b.setAttribute('aria-selected', 'false');
        });
        btn.classList.add('active');
        btn.setAttribute('aria-selected', 'true');

        const target = btn.dataset.tab;
        contentContainer.querySelectorAll('.tab-content').forEach(c => {
          c.classList.toggle('active', c.dataset.tab === target);
        });
      });
    });
  });
})();

/* ── Hero Terminal Typing Animation ─────────────────────────── */
(function initTerminalTyping() {
  const terminal = document.getElementById('hero-terminal-output');
  if (!terminal) return;

  const lines = [
    { text: '$ nexus-daemon', type: 'prompt', delay: 300 },
    { text: ' ✓ HTTP API   → :9999', type: 'out-success', delay: 120 },
    { text: ' ✓ MCP Server → :9998', type: 'out-success', delay: 80 },
    { text: ' ✓ Dashboard  → :9999/ui', type: 'out-success', delay: 80 },
    { text: '', type: 'blank', delay: 400 },
    { text: '$ curl -X POST http://localhost:9999/api/tasks \\', type: 'prompt', delay: 600 },
    { text: '  -d \'{"instruction": "Add auth middleware"}\'', type: 'cmd-cont', delay: 60 },
    { text: '', type: 'blank', delay: 300 },
    { text: '{"id":"a1b2c3","status":"QUEUED"}', type: 'out-val', delay: 200 },
    { text: '', type: 'blank', delay: 400 },
    { text: '$ # Task routed → LM Studio (codellama)', type: 'comment', delay: 500 },
    { text: '  # Completed in 4.2s ✓', type: 'comment', delay: 100 },
  ];

  let lineIndex = 0;
  let charIndex = 0;
  let currentEl = null;
  let running = true;

  function getLineClass(type) {
    const map = {
      'prompt':   'prompt',
      'cmd-cont': 'cmd',
      'out-success': 'out-success',
      'out-val':  'out-val',
      'comment':  'comment',
      'blank':    '',
    };
    return map[type] || '';
  }

  function typeChar() {
    if (!running) return;
    if (lineIndex >= lines.length) {
      // Add blinking cursor at end
      const cursor = document.createElement('span');
      cursor.className = 'cursor';
      terminal.appendChild(cursor);
      return;
    }

    const line = lines[lineIndex];

    if (charIndex === 0) {
      // Start a new line
      if (line.type === 'blank') {
        terminal.appendChild(document.createElement('br'));
        lineIndex++;
        charIndex = 0;
        setTimeout(typeChar, line.delay || 200);
        return;
      }
      currentEl = document.createElement('span');
      const cls = getLineClass(line.type);
      if (cls) currentEl.className = cls;
      terminal.appendChild(currentEl);

      if (line.type !== 'blank') {
        terminal.appendChild(document.createElement('br'));
      }
    }

    if (charIndex < line.text.length) {
      currentEl.textContent += line.text[charIndex];
      charIndex++;
      setTimeout(typeChar, 28);
    } else {
      lineIndex++;
      charIndex = 0;
      setTimeout(typeChar, line.delay || 150);
    }
  }

  // Use IntersectionObserver to start when visible
  if ('IntersectionObserver' in window) {
    const obs = new IntersectionObserver((entries) => {
      entries.forEach(e => {
        if (e.isIntersecting) {
          obs.unobserve(e.target);
          setTimeout(typeChar, 800);
        }
      });
    }, { threshold: 0.3 });
    obs.observe(terminal.closest('.terminal') || terminal);
  } else {
    setTimeout(typeChar, 1000);
  }
})();

/* ── Page-specific: Downloads quick-install copy ────────────── */
(function initQuickInstall() {
  const qiBtn = document.getElementById('qi-copy-btn');
  if (!qiBtn) return;
  qiBtn.addEventListener('click', () => {
    const cmd = document.getElementById('qi-command');
    if (!cmd) return;
    navigator.clipboard.writeText(cmd.textContent.trim()).then(() => {
      qiBtn.textContent = '✓ Copied!';
      qiBtn.classList.add('copied');
      setTimeout(() => {
        qiBtn.textContent = 'Copy';
        qiBtn.classList.remove('copied');
      }, 2000);
    }).catch(() => {});
  });
})();
