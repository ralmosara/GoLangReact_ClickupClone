import type { Config } from 'tailwindcss'

export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  // class-based dark mode — toggled by <html data-theme="dark"> + the
  // `dark` class on <html>. Wired by ThemeProvider in src/lib/theme.tsx.
  // The `dark:` variants below only apply when that class is present, so
  // pages that haven't been retrofitted continue to render in light mode.
  darkMode: ['class', '[data-theme="dark"]'],
  theme: {
    extend: {
      fontFamily: {
        sans: [
          'Inter',
          'ui-sans-serif',
          'system-ui',
          '-apple-system',
          'BlinkMacSystemFont',
          '"Segoe UI"',
          'sans-serif',
        ],
      },
      colors: {
        canvas: '#F4F5F7',
        surface: '#FFFFFF',
        sidebar: {
          DEFAULT: '#0D0D12',
          hover:   '#181824',
          active:  '#1F1F2E',
          border:  'rgba(255,255,255,0.06)',
          text:    '#A0A0B8',
          heading: '#5C5C7A',
          muted:   '#5C5C7A',
        },
        brand: {
          50:  '#EEEEFF',
          100: '#DCDCFF',
          200: '#BEBEFF',
          300: '#9D9DFF',
          400: '#7C7CFA',
          500: '#5B5CF8',
          600: '#4A4BE8',
          700: '#3838C8',
          800: '#2828A8',
          900: '#1A1A80',
        },
        ink: {
          1: '#111827',
          2: '#374151',
          3: '#6B7280',
          4: '#9CA3AF',
          5: '#D1D5DB',
        },
        status: {
          open:        { bg: '#F0F4FF', text: '#4F63C8', dot: '#6375E8' },
          in_progress: { bg: '#FFF7ED', text: '#C05621', dot: '#F97316' },
          review:      { bg: '#FEFCE8', text: '#A16207', dot: '#EAB308' },
          completed:   { bg: '#F0FDF4', text: '#166534', dot: '#22C55E' },
          cancelled:   { bg: '#FEF2F2', text: '#991B1B', dot: '#EF4444' },
        },
      },
      boxShadow: {
        card:    '0 1px 2px rgba(17,24,39,0.04), 0 4px 16px rgba(17,24,39,0.04)',
        'card-hover': '0 2px 8px rgba(17,24,39,0.06), 0 12px 32px rgba(17,24,39,0.08)',
        modal:   '0 8px 48px rgba(17,24,39,0.16)',
        input:   '0 0 0 3px rgba(91,92,248,0.15)',
        button:  '0 1px 2px rgba(91,92,248,0.25), 0 4px 12px rgba(91,92,248,0.20)',
      },
      borderRadius: {
        DEFAULT: '10px',
        sm:  '6px',
        md:  '10px',
        lg:  '14px',
        xl:  '18px',
        '2xl': '24px',
      },
      backgroundImage: {
        'brand-gradient': 'linear-gradient(135deg, #5B5CF8 0%, #8B5CF6 100%)',
        'sidebar-gradient': 'linear-gradient(180deg, #0D0D12 0%, #0A0A10 100%)',
        'canvas-gradient': 'radial-gradient(ellipse at top, #EEF0FF 0%, #F4F5F7 60%)',
      },
      animation: {
        'fade-in':    'fadeIn 0.15s ease-out',
        'slide-up':   'slideUp 0.2s ease-out',
        'scale-in':   'scaleIn 0.15s ease-out',
      },
      keyframes: {
        fadeIn:  { from: { opacity: '0' }, to: { opacity: '1' } },
        slideUp: { from: { opacity: '0', transform: 'translateY(6px)' }, to: { opacity: '1', transform: 'translateY(0)' } },
        scaleIn: { from: { opacity: '0', transform: 'scale(0.97)' }, to: { opacity: '1', transform: 'scale(1)' } },
      },
    },
  },
  plugins: [],
} satisfies Config
