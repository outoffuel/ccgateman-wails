module.exports = {
  content: [
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx,svelte}",
  ],
  theme: {
    extend: {
      colors: {
        m3: {
          primary: '#0B57D0',
          'on-primary': '#FFFFFF',
          'primary-container': '#D3E3FD',
          'on-primary-container': '#041E49',
          secondary: '#00639B',
          'on-secondary': '#FFFFFF',
          'secondary-container': '#C2E7FF',
          'on-secondary-container': '#001D35',
          tertiary: '#146C2E',
          'on-tertiary': '#FFFFFF',
          'tertiary-container': '#C4EED0',
          'on-tertiary-container': '#072711',
          warning: '#B06000',
          'on-warning': '#FFFFFF',
          'warning-container': '#FFDEA8',
          'on-warning-container': '#381E00',
          error: '#B3261E',
          'on-error': '#FFFFFF',
          'error-container': '#F9DEDC',
          'on-error-container': '#410E0B',
          surface: '#F8FAFD',
          'surface-dim': '#D8DADF',
          'surface-bright': '#F8FAFD',
          'surface-container-lowest': '#FFFFFF',
          'surface-container-low': '#F2F5FA',
          'surface-container': '#ECEEF4',
          'surface-container-high': '#E6E8EE',
          'surface-container-highest': '#E0E2E8',
          'on-surface': '#191C20',
          'on-surface-variant': '#44474E',
          outline: '#74777F',
          'outline-variant': '#C4C6D0',
          'inverse-surface': '#2E3036',
          'inverse-on-surface': '#F0F0F7',
        }
      },
      boxShadow: {
        'm3-1': '0 1px 2px 0 rgba(0, 0, 0, 0.06), 0 1px 3px 1px rgba(0, 0, 0, 0.04)',
        'm3-2': '0 1px 2px 0 rgba(0, 0, 0, 0.08), 0 2px 6px 2px rgba(0, 0, 0, 0.05)',
        'm3-3': '0 4px 8px 3px rgba(0, 0, 0, 0.06), 0 1px 3px 0 rgba(0, 0, 0, 0.08)',
      },
      borderRadius: {
        'm3-dialog': '28px',
      }
    },
  },
  plugins: [],
}
