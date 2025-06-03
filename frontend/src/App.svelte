<script>
  import { onMount } from 'svelte';
  import LoginView from './components/LoginView.svelte';
  import MainDashboardView from './components/MainDashboardView.svelte';
  import RemoteControlDialog from './components/RemoteControlDialog.svelte';
  import VideoRecordingIndicator from './components/VideoRecordingIndicator.svelte';
  import FileTransferNotification from './components/FileTransferNotification.svelte';
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
  let activeSessionAdmin = '';

  // Suscribirse a cambios de estado
  $: currentView = $appState.currentView;

  // Configurar event listeners inmediatamente (ANTES del onMount)
  setupRemoteControlListeners();

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
  });

  function setupRemoteControlListeners() {
    console.log('🎧 Setting up remote control event listeners...');
    
    // Escuchar solicitudes de control remoto entrantes
    EventsOn('incoming_control_request', (data) => {
      console.log('🎮 Incoming control request:', data);
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
      console.log('✅ Control session accepted:', data);
      remoteControlActive = true;
      activeSessionId = data.sessionId || '';
      activeSessionAdmin = data.adminUsername || remoteControlRequest.adminUsername || 'Administrador';
      showRemoteControlDialog = false;
    });

    // Escuchar cuando se rechaza una sesión
    EventsOn('control_session_rejected', (data) => {
      console.log('❌ Control session rejected:', data);
      showRemoteControlDialog = false;
      remoteControlActive = false;
      activeSessionId = '';
      activeSessionAdmin = '';
    });

    // Escuchar cuando una sesión inicia efectivamente (backend confirmation)
    EventsOn('control_session_started', (data) => {
      console.log('🚀 Control session started by backend:', data);
      remoteControlActive = true;
      if (data && data.sessionId) {
        activeSessionId = data.sessionId;
      }
    });

    // Escuchar cuando termina una sesión (backend notification)
    EventsOn('control_session_ended', (data) => {
      console.log('🔚 Control session ended by backend:', data);
      remoteControlActive = false;
      activeSessionId = '';
      activeSessionAdmin = '';
    });

    // Escuchar cuando falla una sesión
    EventsOn('control_session_failed', (data) => {
      console.log('❌ Control session failed:', data);
      remoteControlActive = false;
      activeSessionId = '';
      activeSessionAdmin = '';
      showRemoteControlDialog = false;
    });
    
    console.log('✅ Remote control event listeners configured');
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

  function handleAuthenticated() {
    console.log('User authenticated, switching to dashboard');
    // Esta función será llamada cuando el login sea exitoso
  }
</script>

<main>
  {#if loading}
    <div class="loading-screen">
      <div class="loading-content">
        <div class="loading-logo">🖥️</div>
        <h2>RemoteDesk Cliente</h2>
        <div class="loading-spinner"></div>
        <p>Iniciando aplicación...</p>
      </div>
    </div>
  {:else if currentView === 'login'}
    <LoginView on:authenticated={handleAuthenticated} />
  {:else if currentView === 'dashboard'}
    <MainDashboardView />
  {/if}

  <!-- Notificación de Sesión Activa -->
  {#if remoteControlActive}
    <div class="session-notification" class:active={remoteControlActive}>
      <div class="notification-content">
        <div class="notification-icon">🎮</div>
        <div class="notification-text">
          <h4>Sesión Remota Activa</h4>
          <p>Administrador: <strong>{activeSessionAdmin}</strong></p>
          <small>Sesión: {activeSessionId.substring(0, 8)}...</small>
        </div>
        <div class="notification-status">
          <div class="status-pulse"></div>
          <span>En curso</span>
        </div>
      </div>
    </div>
  {/if}

  <!-- Diálogo de Control Remoto -->
  {#if showRemoteControlDialog}
    <RemoteControlDialog
      visible={showRemoteControlDialog}
      adminUsername={remoteControlRequest.adminUsername}
      sessionId={remoteControlRequest.sessionId}
      on:accepted={handleRemoteControlAccepted}
      on:rejected={handleRemoteControlRejected}
    />
  {/if}

  <!-- Indicador de Grabación de Video -->
  <VideoRecordingIndicator />

  <!-- Notificaciones de Transferencia de Archivos -->
  <FileTransferNotification />
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

  .loading-screen {
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    height: 100vh;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: white;
  }

  .loading-content {
    text-align: center;
  }

  .loading-logo {
    font-size: 48px;
    margin-bottom: 20px;
  }

  .loading-spinner {
    width: 40px;
    height: 40px;
    border: 4px solid rgba(255, 255, 255, 0.3);
    border-top: 4px solid white;
    border-radius: 50%;
    animation: spin 1s linear infinite;
    margin: 0 auto 20px;
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

  /* Notificación de Sesión Activa */
  .session-notification {
    position: fixed;
    top: 20px;
    right: 20px;
    background: linear-gradient(135deg, #10b981 0%, #059669 100%);
    color: white;
    padding: 16px 20px;
    border-radius: 12px;
    border: 1px solid rgba(255, 255, 255, 0.2);
    box-shadow: 0 8px 25px rgba(16, 185, 129, 0.3);
    backdrop-filter: blur(10px);
    z-index: 1000;
    min-width: 280px;
    max-width: 350px;
    animation: slideInRight 0.4s ease-out;
    transition: all 0.3s ease;
  }

  .session-notification.active {
    transform: translateX(0);
    opacity: 1;
  }

  .notification-content {
    display: flex;
    align-items: flex-start;
    gap: 12px;
  }

  .notification-icon {
    font-size: 20px;
    margin-top: 2px;
  }

  .notification-text {
    flex: 1;
  }

  .notification-text h4 {
    margin: 0 0 4px 0;
    font-size: 14px;
    font-weight: 600;
    color: white;
  }

  .notification-text p {
    margin: 0 0 4px 0;
    font-size: 12px;
    color: rgba(255, 255, 255, 0.9);
  }

  .notification-text small {
    font-size: 10px;
    color: rgba(255, 255, 255, 0.7);
    font-family: monospace;
  }

  .notification-status {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-top: 2px;
  }

  .notification-status span {
    font-size: 11px;
    font-weight: 600;
    color: rgba(255, 255, 255, 0.9);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .status-pulse {
    width: 8px;
    height: 8px;
    background: rgba(255, 255, 255, 0.9);
    border-radius: 50%;
    animation: pulse 2s infinite;
  }

  @keyframes slideInRight {
    from {
      transform: translateX(100%);
      opacity: 0;
    }
    to {
      transform: translateX(0);
      opacity: 1;
    }
  }

  @keyframes pulse {
    0%, 100% {
      opacity: 1;
      transform: scale(1);
    }
    50% {
      opacity: 0.6;
      transform: scale(1.2);
    }
  }
</style>
