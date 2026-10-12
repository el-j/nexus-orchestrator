import { readonly, ref } from 'vue';

/** Visual tone of a toast. */
export type ToastSeverity = 'success' | 'info' | 'warn' | 'error';

/** What a caller passes to {@link ToastApi.add}. `life` is the display time in ms (default 3000). */
export interface ToastMessage {
  severity: ToastSeverity;
  summary: string;
  detail?: string;
  life?: number;
}

/** A toast that is currently shown. */
export interface ActiveToast extends ToastMessage {
  id: number;
}

/** The toast service handed to components and test doubles. */
export interface ToastApi {
  add: (message: ToastMessage) => void;
  remove: (id: number) => void;
}

const DEFAULT_LIFE_MS = 3000;
const toasts = ref<ActiveToast[]>([]);
let nextId = 1;

/** Removes a toast (also called by its own timer and its close button). */
function remove(id: number) {
  toasts.value = toasts.value.filter((t) => t.id !== id);
}

/** Shows a toast and removes it again after `life` ms. */
function add(message: ToastMessage) {
  const id = nextId++;
  toasts.value = [...toasts.value, { ...message, id }];
  setTimeout(() => remove(id), message.life ?? DEFAULT_LIFE_MS);
}

/** Read-only list of the toasts to render; shared by every caller of {@link useToast}. */
export const activeToasts = readonly(toasts);

/** Module-wide toast service. */
export function useToast(): ToastApi {
  return { add, remove };
}
