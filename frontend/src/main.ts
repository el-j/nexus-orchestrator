import { createApp } from 'vue';
import './assets/main.css';
import App from './App.vue';
import router from './router/index';

const app = createApp(App);

app.config.errorHandler = (err, _instance, info) => {
  console.error('[Vue] Unhandled error:', err, info);
};

window.addEventListener('unhandledrejection', (event) => {
  console.error('[Vue] Unhandled promise rejection:', event.reason);
  event.preventDefault();
});

app.use(router);
app.mount('#app');
