<script>
  import { onMount } from 'svelte';
  import LoginView from './components/LoginView.svelte';
  import MainDashboardView from './components/MainDashboardView.svelte';
  import RemoteControlDialog from './components/RemoteControlDialog.svelte';
  import { 
    isAuthenticated, 
    appState, 
    setAuthenticated,
    setLoading 
  } from './stores/app.js';
  import { EventsOn } from '../wailsjs/runtime/runtime.js';

  let currentView = 'login';
  let loading = true;

  // Estado del diálogo de control remoto
  let showRemoteControlDialog = false;
  let remoteControlRequest = {
    sessionId: '',
    adminUsername: '',
    adminUserId: '',
    clientPcId: ''
  };

  // Estado de sesión de control remoto
  let remoteControlActive = false;
  let activeSessionId = '';

  // Suscribirse a cambios de estado
  $: currentView = $appState.currentView;

  onMount(async () => {
    // Verificar si hay una sesión existente
    try {
      // Por ahora no hay método GetSessionInfo, así que simplificamos
      console.log('App mounted successfully');
    } catch (err) {
      console.log('No existing session found');
    } finally {
      loading = false;
      setLoading(false);
    }

    // Configurar event listeners para control remoto
    setupRemoteControlListeners();
  });

  function setupRemoteControlListeners() {
    // Escuchar solicitudes de control remoto entrantes
    EventsOn('incoming_control_request', (data) => {
      console.log('Incoming control request:', data);
      remoteControlRequest = {
        sessionId: data.sessionId || '',
        adminUsername: data.adminUsername || 'Administrador',
        adminUserId: data.adminUserId || '',
        clientPcId: data.clientPcId || ''
      };
      showRemoteControlDialog = true;
    });

    // Escuchar cuando se acepta una sesión
    EventsOn('control_session_accepted', (data) => {
      console.log('Control session accepted:', data);
      remoteControlActive = true;
      activeSessionId = data.sessionId || '';
      showRemoteControlDialog = false;
    });

    // Escuchar cuando se rechaza una sesión
    EventsOn('control_session_rejected', (data) => {
      console.log('Control session rejected:', data);
      showRemoteControlDialog = false;
      remoteControlActive = false;
      activeSessionId = '';
    });

    // Escuchar cuando termina una sesión
    EventsOn('control_session_ended', (data) => {
      console.log('Control session ended:', data);
      remoteControlActive = false;
      activeSessionId = '';
    });
  }

  function handleRemoteControlAccepted(event) {
    console.log('Remote control accepted:', event.detail);
    remoteControlActive = true;
    activeSessionId = event.detail.sessionId;
  }

  function handleRemoteControlRejected(event) {
    console.log('Remote control rejected:', event.detail);
    showRemoteControlDialog = false;
  }
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

  <!-- Indicador de sesión de control remoto activa -->
  {#if remoteControlActive}
    <div class="remote-control-indicator">
      <div class="indicator-content">
        <span class="indicator-icon">🖥️</span>
        <span class="indicator-text">Sesión de control remoto activa</span>
        <div class="indicator-pulse"></div>
      </div>
    </div>
  {/if}

  <!-- Diálogo de solicitud de control remoto -->
  <RemoteControlDialog
    bind:visible={showRemoteControlDialog}
    adminUsername={remoteControlRequest.adminUsername}
    sessionId={remoteControlRequest.sessionId}
    on:accepted={handleRemoteControlAccepted}
    on:rejected={handleRemoteControlRejected}
  />
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
    position: relative;
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

  /* Indicador de sesión de control remoto */
  .remote-control-indicator {
    position: fixed;
    top: 20px;
    right: 20px;
    z-index: 999;
    background: linear-gradient(135deg, #ff6b6b 0%, #ee5a24 100%);
    color: white;
    padding: 12px 20px;
    border-radius: 25px;
    box-shadow: 0 4px 15px rgba(255, 107, 107, 0.3);
    animation: slideInRight 0.3s ease-out;
  }

  .indicator-content {
    display: flex;
    align-items: center;
    gap: 8px;
    position: relative;
  }

  .indicator-icon {
    font-size: 16px;
  }

  .indicator-text {
    font-size: 14px;
    font-weight: 600;
  }

  .indicator-pulse {
    width: 8px;
    height: 8px;
    background: white;
    border-radius: 50%;
    animation: pulse 2s infinite;
  }

  @keyframes slideInRight {
    from {
      opacity: 0;
      transform: translateX(100%);
    }
    to {
      opacity: 1;
      transform: translateX(0);
    }
  }

  @keyframes pulse {
    0%, 100% {
      opacity: 1;
      transform: scale(1);
    }
    50% {
      opacity: 0.5;
      transform: scale(1.2);
    }
  }
</style>
