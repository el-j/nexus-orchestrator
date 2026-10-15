import { createApp, defineComponent, h, type App } from 'vue';

/**
 * Runs a composable inside a real component's setup() so lifecycle hooks
 * (onMounted / onUnmounted / watch) behave exactly as in the application.
 * Returns the composable's result plus the app, so tests can unmount it.
 */
export function withSetup<T>(composable: () => T): { result: T; app: App; unmount: () => void } {
  let result!: T;
  const app = createApp(
    defineComponent({
      setup() {
        result = composable();
        return () => h('div');
      },
    }),
  );
  app.mount(document.createElement('div'));
  return { result, app, unmount: () => app.unmount() };
}
