import './app.css';
import App from './App.svelte';
import { configureApiClient } from '$gen/api/client';

// Configure API client — Vite proxy handles /v1 → localhost:8080
configureApiClient({
  baseUrl: '',
  getAuthHeaders: () => ({
    'X-Actor': 'poc-user',
    'X-Source': 'user',
  }),
});

const app = new App({
  target: document.getElementById('app')!,
});

export default app;
