<script>
  import { onMount } from 'svelte';
  import LoginView from './components/LoginView.svelte';
  import MainDashboardView from './components/MainDashboardView.svelte';
  import { 
    isAuthenticated, 
    appState, 
    setAuthenticated,
    setLoading 
  } from './stores/app.js';
  import { GetSessionInfo } from '../wailsjs/go/main/App.js';

  let currentView = 'login';
  let loading = true;

  // Suscribirse a cambios de estado
  $: currentView = $appState.currentView;

  onMount(async () => {
    // Verificar si hay una sesión existente
    try {
      const sessionData = await GetSessionInfo();
      if (sessionData && sessionData.token && sessionData.userId) {
        setAuthenticated(true, {
          username: sessionData.username,
          userId: sessionData.userId,
          token: sessionData.token
        });
      }
    } catch (err) {
      console.log('No existing session found');
    } finally {
      loading = false;
      setLoading(false);
    }
  });
</script>

<main>
  {#if loading}
    <div class="loading-container">
      <div class="loading-spinner"></div>
      <p>Cargando...</p>
    </div>
  {:else if currentView === 'login'}
    <LoginView />
  {:else if currentView === 'dashboard'}
    <MainDashboardView />
  {/if}
</main>

<style>
  :global(body) {
    margin: 0;
    padding: 0;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', 'Oxygen',
      'Ubuntu', 'Cantarell', 'Fira Sans', 'Droid Sans', 'Helvetica Neue',
      sans-serif;
    -webkit-font-smoothing: antialiased;
    -moz-osx-font-smoothing: grayscale;
  }

  :global(*) {
    box-sizing: border-box;
  }

  main {
    width: 100%;
    height: 100vh;
    overflow: hidden;
  }

  .loading-container {
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    height: 100vh;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: white;
  }

  .loading-spinner {
    width: 40px;
    height: 40px;
    border: 4px solid rgba(255, 255, 255, 0.3);
    border-top: 4px solid white;
    border-radius: 50%;
    animation: spin 1s linear infinite;
    margin-bottom: 20px;
  }

  .loading-container p {
    font-size: 18px;
    margin: 0;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }
</style>
