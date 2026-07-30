export const SHANIU_VERSION = '0.2.8';

declare global {
  interface Window {
    __SHANIU_VERSION__?: string;
    SHANIU_VERSION?: string;
  }
}

export function mountShaniuVersion() {
  if (typeof window === 'undefined') return;
  window.__SHANIU_VERSION__ = SHANIU_VERSION;
  window.SHANIU_VERSION = SHANIU_VERSION;
}
