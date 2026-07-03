// Simple toast replacement for react-toastify
// Uses native browser notification or console

type ToastType = 'success' | 'error' | 'info' | 'warning';

interface ToastOptions {
  type?: ToastType;
}

const toast = {
  success: (msg: string) => console.info(`✅ ${msg}`),
  error: (msg: string) => console.error(`❌ ${msg}`),
  info: (msg: string) => console.info(`ℹ️ ${msg}`),
  warning: (msg: string) => console.warn(`⚠️ ${msg}`),
};

export { toast };
export default toast;
