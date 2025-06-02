<script>
    import { onMount, onDestroy } from 'svelte';
    import { 
        RegisterPC, 
        GetSystemInfo, 
        GetConnectionStatus, 
        Logout 
    } from '../../wailsjs/go/main/App.js';
    import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime.js';
    import { 
        setRegistered, 
        setConnected, 
        setAuthenticated, 
        setLoading, 
        setError,
        userInfo,
        pcInfo,
        isRegistered,
        isConnected
    } from '../stores/app.js';
    
    let systemInfo = {};
    let loading = false;
    let error = null;
    let registrationStatus = 'pending'; // 'pending', 'registering', 'registered', 'error'
    let connectionStatus = {
        isConnected: true, // Inicialmente true porque ya autenticamos
        status: 'connected',
        lastHeartbeat: Date.now() / 1000,
        serverUrl: 'localhost:8080',
        connectionTime: 0,
        errorMessage: ''
    };
    
    // Auto refresh de estado
    let statusRefreshInterval;
    let eventCleanup = null;
    
    onMount(async () => {
        await loadInitialData();
        await updateConnectionStatus();
        
        // Configurar auto-refresh del estado cada 10 segundos
        statusRefreshInterval = setInterval(updateConnectionStatus, 10000);
        
        // Suscribirse a eventos después de un delay
        setTimeout(() => {
            try {
                eventCleanup = EventsOn("connection_status_update", handleConnectionStatusUpdate);
                console.log("✅ Event listener registrado");
            } catch (err) {
                console.error("❌ Error registrando event listener:", err);
            }
        }, 1000);
    });
    
    onDestroy(() => {
        if (statusRefreshInterval) {
            clearInterval(statusRefreshInterval);
        }
        
        if (eventCleanup) {
            try {
                EventsOff("connection_status_update");
                console.log("✅ Event listener removido");
            } catch (err) {
                console.error("❌ Error removiendo event listener:", err);
            }
        }
    });
    
    async function loadInitialData() {
        try {
            console.log("🔄 Cargando información del sistema...");
            systemInfo = await GetSystemInfo();
            console.log("✅ Información del sistema cargada:", systemInfo);
        } catch (err) {
            console.error('❌ Error loading initial data:', err);
            setError('Error cargando datos del sistema');
        }
    }
    
    async function updateConnectionStatus() {
        try {
            const statusResponse = await GetConnectionStatus();
            console.log("🔄 Estado de conexión:", statusResponse);
            
            if (statusResponse.connection_info) {
                const connected = statusResponse.connection_info.is_connected;
                connectionStatus = {
                    isConnected: connected,
                    status: connected ? 'connected' : 'disconnected',
                    lastHeartbeat: Date.now() / 1000,
                    serverUrl: statusResponse.connection_info.server_url || 'localhost:8080',
                    connectionTime: parseInt(statusResponse.connection_info.connection_time) || 0,
                    errorMessage: ''
                };
                setConnected(connected);
                
                console.log(`🔗 Estado actualizado: ${connected ? 'CONECTADO' : 'DESCONECTADO'}`);
            } else {
                console.warn("⚠️ Respuesta de estado sin connection_info");
                // Mantener como conectado si no hay info específica de desconexión
                connectionStatus.lastHeartbeat = Date.now() / 1000;
            }
        } catch (err) {
            console.error('❌ Error obteniendo estado de conexión:', err);
            // No marcar como desconectado inmediatamente en caso de error temporal
        }
    }
    
    function handleConnectionStatusUpdate(newStatus) {
        console.log("🔄 Event de estado recibido:", newStatus);
        
        if (newStatus && typeof newStatus === 'object') {
            connectionStatus = { ...connectionStatus, ...newStatus };
            setConnected(newStatus.isConnected);
        }
    }
    
    async function handleRegisterPC() {
        if (!connectionStatus.isConnected) {
            error = 'No hay conexión con el servidor. Por favor reconecta.';
            return;
        }

        registrationStatus = 'registering';
        loading = true;
        error = null;
        setLoading(true);
        
        try {
            console.log("🔄 Registrando PC en el servidor...");
            const result = await RegisterPC();
            console.log("📝 Resultado del registro:", result);
            
            if (result.success) {
                registrationStatus = 'registered';
                setRegistered(true, {
                    pcId: result.pc_info?.identifier || 'unknown',
                    identifier: result.pc_info?.hostname || systemInfo.hostname || 'unknown'
                });
                console.log("✅ PC registrado exitosamente");
            } else {
                registrationStatus = 'error';
                error = result.error || 'Error registrando PC en el servidor';
                setError(error);
                console.error("❌ Error en registro:", error);
            }
        } catch (err) {
            registrationStatus = 'error';
            error = 'Error de comunicación: ' + err.message;
            setError(error);
            console.error("❌ Excepción en registro:", err);
        } finally {
            loading = false;
            setLoading(false);
        }
    }
    
    async function handleLogout() {
        try {
            console.log("🔄 Cerrando sesión...");
            await Logout();
            setAuthenticated(false);
            console.log("✅ Logout exitoso");
        } catch (err) {
            console.error('❌ Error durante logout:', err);
            // Forzar logout local aunque falle el servidor
            setAuthenticated(false);
        }
    }
    
    function getStatusColor(status) {
        switch (status) {
            case 'registered': return '#10b981';
            case 'registering': return '#f59e0b';
            case 'error': return '#ef4444';
            default: return '#6b7280';
        }
    }
    
    function getStatusText(status) {
        switch (status) {
            case 'registered': return 'Registrado';
            case 'registering': return 'Registrando...';
            case 'error': return 'Error';
            default: return 'Pendiente';
        }
    }
    
    function getConnectionStatusColor(status) {
        switch (status) {
            case 'connected': return '#10b981';
            case 'disconnected': return '#ef4444';
            case 'error': return '#dc2626';
            default: return '#6b7280';
        }
    }
    
    function formatTimestamp(timestamp) {
        if (!timestamp) return 'N/A';
        return new Date(timestamp * 1000).toLocaleTimeString('es-ES');
    }

    function formatUptime(seconds) {
        if (!seconds) return 'N/A';
        const hours = Math.floor(seconds / 3600);
        const minutes = Math.floor((seconds % 3600) / 60);
        return `${hours}h ${minutes}m`;
    }

    // Datos del usuario desde la store
    let userData = {};
    $: userData = $userInfo || {};
</script>

<div class="dashboard-container">
    <!-- Header Moderno -->
    <header class="dashboard-header">
        <div class="header-content">
            <div class="header-left">
                <div class="logo">
                    <div class="logo-icon">🖥️</div>
                    <div class="logo-text">
                        <h1>Escritorio Remoto</h1>
                        <span class="version">Cliente v2.0</span>
                    </div>
                </div>
            </div>
            
            <div class="header-right">
                <div class="user-profile">
                    <div class="user-avatar">
                        <span>{(userData.username || 'U').charAt(0).toUpperCase()}</span>
                    </div>
                    <div class="user-info">
                        <span class="user-name">{userData.username || 'Usuario'}</span>
                        <span class="user-status">Conectado</span>
                    </div>
                </div>
                
                <button class="logout-button" on:click={handleLogout} title="Cerrar Sesión">
                    <span class="logout-icon">🚪</span>
                    <span class="logout-text">Cerrar Sesión</span>
                </button>
            </div>
        </div>
    </header>
    
    <main class="dashboard-main">
        <!-- Estado Cards Grid -->
        <div class="status-grid">
            <!-- Estado de Conexión -->
            <div class="status-card connection-card">
                <div class="card-header">
                    <h3>🔗 Estado de Conexión</h3>
                    <div class="status-badge" class:connected={connectionStatus.isConnected} class:disconnected={!connectionStatus.isConnected}>
                        <div class="status-dot"></div>
                        <span>{connectionStatus.isConnected ? 'Conectado' : 'Desconectado'}</span>
                    </div>
                </div>
                
                <div class="card-content">
                    <div class="connection-details">
                        <div class="detail-row">
                            <span class="detail-label">Servidor:</span>
                            <span class="detail-value server">{connectionStatus.serverUrl}</span>
                        </div>
                        <div class="detail-row">
                            <span class="detail-label">Estado:</span>
                            <span class="detail-value" style="color: {getConnectionStatusColor(connectionStatus.status)}">
                                {connectionStatus.status === 'connected' ? 'Activo' : 'Inactivo'}
                            </span>
                        </div>
                        <div class="detail-row">
                            <span class="detail-label">Último ping:</span>
                            <span class="detail-value">{formatTimestamp(connectionStatus.lastHeartbeat)}</span>
                        </div>
                        <div class="detail-row">
                            <span class="detail-label">Tiempo conexión:</span>
                            <span class="detail-value">{formatUptime(connectionStatus.connectionTime)}</span>
                        </div>
                    </div>
                    
                    {#if connectionStatus.errorMessage}
                        <div class="error-alert">
                            <span class="error-icon">⚠️</span>
                            <span class="error-text">{connectionStatus.errorMessage}</span>
                        </div>
                    {/if}
                </div>
            </div>
            
            <!-- Registro del PC -->
            <div class="status-card registration-card">
                <div class="card-header">
                    <h3>📝 Registro del PC</h3>
                    <div class="status-badge" style="background-color: {getStatusColor(registrationStatus)}">
                        <span>{getStatusText(registrationStatus)}</span>
                    </div>
                </div>
                
                <div class="card-content">
                    {#if registrationStatus === 'registered'}
                        <div class="registration-success">
                            <div class="success-icon">✅</div>
                            <div class="success-details">
                                <h4>¡PC Registrado!</h4>
                                <p>Tu equipo está disponible para control remoto</p>
                                <div class="pc-details">
                                    <div class="detail-row">
                                        <span class="detail-label">Hostname:</span>
                                        <span class="detail-value">{systemInfo.hostname || 'N/A'}</span>
                                    </div>
                                    <div class="detail-row">
                                        <span class="detail-label">Sistema:</span>
                                        <span class="detail-value">{systemInfo.os || 'N/A'}</span>
                                    </div>
                                </div>
                            </div>
                        </div>
                    {:else if registrationStatus === 'pending'}
                        <div class="registration-pending">
                            <div class="pending-icon">📋</div>
                            <div class="pending-details">
                                <h4>PC no registrado</h4>
                                <p>Registra tu PC para habilitarlo en el sistema de control remoto</p>
                                
                                <button 
                                    class="register-button" 
                                    on:click={handleRegisterPC}
                                    disabled={loading || !connectionStatus.isConnected}
                                >
                                    {#if loading}
                                        <span class="spinner"></span>
                                        <span>Registrando...</span>
                                    {:else}
                                        <span class="register-icon">🚀</span>
                                        <span>Registrar PC</span>
                                    {/if}
                                </button>
                                
                                {#if !connectionStatus.isConnected}
                                    <div class="warning-message">
                                        <span class="warning-icon">⚠️</span>
                                        <span>Necesitas estar conectado al servidor para registrar el PC</span>
                                    </div>
                                {/if}
                            </div>
                        </div>
                    {:else if registrationStatus === 'error'}
                        <div class="registration-error">
                            <div class="error-icon">❌</div>
                            <div class="error-details">
                                <h4>Error de registro</h4>
                                <p class="error-text">{error}</p>
                                
                                <button 
                                    class="retry-button" 
                                    on:click={handleRegisterPC}
                                    disabled={loading || !connectionStatus.isConnected}
                                >
                                    <span class="retry-icon">🔄</span>
                                    <span>Reintentar</span>
                                </button>
                            </div>
                        </div>
                    {/if}
                </div>
            </div>
            
            <!-- Información del Sistema -->
            <div class="status-card system-card">
                <div class="card-header">
                    <h3>💻 Información del Sistema</h3>
                    <div class="refresh-button" on:click={loadInitialData} title="Actualizar información">
                        <span>🔄</span>
                    </div>
                </div>
                
                <div class="card-content">
                    <div class="system-grid">
                        <div class="system-item">
                            <div class="system-icon">🏠</div>
                            <div class="system-details">
                                <span class="system-label">Hostname</span>
                                <span class="system-value">{systemInfo.hostname || 'N/A'}</span>
                            </div>
                        </div>
                        
                        <div class="system-item">
                            <div class="system-icon">💿</div>
                            <div class="system-details">
                                <span class="system-label">Sistema</span>
                                <span class="system-value">{systemInfo.os || 'N/A'}</span>
                            </div>
                        </div>
                        
                        <div class="system-item">
                            <div class="system-icon">⚙️</div>
                            <div class="system-details">
                                <span class="system-label">Arquitectura</span>
                                <span class="system-value">{systemInfo.arch || 'N/A'}</span>
                            </div>
                        </div>
                        
                        <div class="system-item">
                            <div class="system-icon">🔧</div>
                            <div class="system-details">
                                <span class="system-label">CPUs</span>
                                <span class="system-value">{systemInfo.num_cpu || 'N/A'}</span>
                            </div>
                        </div>
                        
                        <div class="system-item">
                            <div class="system-icon">🌐</div>
                            <div class="system-details">
                                <span class="system-label">IP Local</span>
                                <span class="system-value">{systemInfo.local_ip || 'N/A'}</span>
                            </div>
                        </div>
                        
                        <div class="system-item">
                            <div class="system-icon">🔢</div>
                            <div class="system-details">
                                <span class="system-label">Versión</span>
                                <span class="system-value">{systemInfo.version || 'N/A'}</span>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
        
        <!-- Success Banner cuando PC está registrado -->
        {#if registrationStatus === 'registered'}
            <div class="success-banner">
                <div class="banner-content">
                    <div class="banner-icon">🎉</div>
                    <div class="banner-text">
                        <h3>¡Todo listo!</h3>
                        <p>Tu PC está registrado y conectado. Los administradores pueden solicitar control remoto cuando sea necesario.</p>
                    </div>
                    <div class="banner-status">
                        <div class="status-indicator active">
                            <div class="indicator-dot"></div>
                            <span>Sistema Activo</span>
                        </div>
                    </div>
                </div>
            </div>
        {/if}
    </main>
</div>

<style>
    .dashboard-container {
        min-height: 100vh;
        background: linear-gradient(135deg, #f8fafc 0%, #e2e8f0 100%);
        font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', sans-serif;
    }
    
    /* Header Styles */
    .dashboard-header {
        background: white;
        border-bottom: 1px solid #e2e8f0;
        box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
        position: sticky;
        top: 0;
        z-index: 100;
    }
    
    .header-content {
        max-width: 1400px;
        margin: 0 auto;
        padding: 16px 24px;
        display: flex;
        justify-content: space-between;
        align-items: center;
    }
    
    .header-left {
        display: flex;
        align-items: center;
    }
    
    .logo {
        display: flex;
        align-items: center;
        gap: 12px;
    }
    
    .logo-icon {
        font-size: 32px;
        filter: drop-shadow(0 2px 4px rgba(0, 0, 0, 0.1));
    }
    
    .logo-text h1 {
        margin: 0;
        font-size: 24px;
        font-weight: 700;
        color: #1a202c;
        letter-spacing: -0.5px;
    }
    
    .version {
        font-size: 12px;
        color: #718096;
        font-weight: 500;
        background: #edf2f7;
        padding: 2px 8px;
        border-radius: 12px;
        margin-left: 8px;
    }
    
    .header-right {
        display: flex;
        align-items: center;
        gap: 16px;
    }
    
    .user-profile {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 8px 16px;
        background: #f7fafc;
        border-radius: 12px;
        border: 1px solid #e2e8f0;
    }
    
    .user-avatar {
        width: 40px;
        height: 40px;
        border-radius: 50%;
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        display: flex;
        align-items: center;
        justify-content: center;
        color: white;
        font-weight: 700;
        font-size: 16px;
    }
    
    .user-info {
        display: flex;
        flex-direction: column;
    }
    
    .user-name {
        font-weight: 600;
        color: #2d3748;
        font-size: 14px;
    }
    
    .user-status {
        font-size: 12px;
        color: #10b981;
        font-weight: 500;
    }
    
    .logout-button {
        display: flex;
        align-items: center;
        gap: 8px;
        background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%);
        color: white;
        border: none;
        padding: 10px 16px;
        border-radius: 10px;
        cursor: pointer;
        font-weight: 600;
        font-size: 14px;
        transition: all 0.3s ease;
        box-shadow: 0 2px 8px rgba(239, 68, 68, 0.3);
    }
    
    .logout-button:hover {
        transform: translateY(-1px);
        box-shadow: 0 4px 12px rgba(239, 68, 68, 0.4);
    }
    
    /* Main Content */
    .dashboard-main {
        max-width: 1400px;
        margin: 0 auto;
        padding: 32px 24px;
    }
    
    .status-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
        gap: 24px;
        margin-bottom: 32px;
    }
    
    /* Card Styles */
    .status-card {
        background: white;
        border-radius: 16px;
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.05);
        border: 1px solid #e2e8f0;
        overflow: hidden;
        transition: all 0.3s ease;
    }
    
    .status-card:hover {
        transform: translateY(-2px);
        box-shadow: 0 8px 25px rgba(0, 0, 0, 0.1);
    }
    
    .card-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 20px 24px 16px 24px;
        border-bottom: 1px solid #f1f5f9;
    }
    
    .card-header h3 {
        margin: 0;
        font-size: 18px;
        font-weight: 700;
        color: #1a202c;
    }
    
    .status-badge {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 6px 12px;
        border-radius: 20px;
        font-size: 12px;
        font-weight: 600;
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }
    
    .status-badge.connected {
        background: linear-gradient(135deg, #10b981 0%, #059669 100%);
        color: white;
    }
    
    .status-badge.disconnected {
        background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%);
        color: white;
    }
    
    .status-dot {
        width: 8px;
        height: 8px;
        border-radius: 50%;
        background: currentColor;
        animation: pulse 2s infinite;
    }
    
    @keyframes pulse {
        0%, 100% { opacity: 1; }
        50% { opacity: 0.6; }
    }
    
    .card-content {
        padding: 20px 24px 24px 24px;
    }
    
    /* Connection Details */
    .connection-details {
        display: flex;
        flex-direction: column;
        gap: 12px;
    }
    
    .detail-row {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 8px 0;
        border-bottom: 1px solid #f7fafc;
    }
    
    .detail-row:last-child {
        border-bottom: none;
    }
    
    .detail-label {
        font-weight: 500;
        color: #4a5568;
        font-size: 14px;
    }
    
    .detail-value {
        font-weight: 600;
        color: #2d3748;
        font-size: 14px;
    }
    
    .detail-value.server {
        font-family: 'Courier New', monospace;
        background: #edf2f7;
        padding: 4px 8px;
        border-radius: 6px;
        font-size: 13px;
    }
    
    /* Registration Styles */
    .registration-success,
    .registration-pending,
    .registration-error {
        display: flex;
        align-items: flex-start;
        gap: 16px;
    }
    
    .success-icon,
    .pending-icon,
    .error-icon {
        font-size: 32px;
        flex-shrink: 0;
    }
    
    .success-details h4,
    .pending-details h4,
    .error-details h4 {
        margin: 0 0 8px 0;
        font-size: 16px;
        font-weight: 700;
        color: #1a202c;
    }
    
    .success-details p,
    .pending-details p {
        margin: 0 0 16px 0;
        color: #4a5568;
        line-height: 1.5;
    }
    
    .pc-details {
        display: flex;
        flex-direction: column;
        gap: 8px;
        margin-top: 16px;
    }
    
    /* Buttons */
    .register-button {
        display: flex;
        align-items: center;
        gap: 8px;
        background: linear-gradient(135deg, #3182ce 0%, #2b6cb0 100%);
        color: white;
        border: none;
        padding: 12px 20px;
        border-radius: 10px;
        cursor: pointer;
        font-weight: 600;
        font-size: 14px;
        transition: all 0.3s ease;
        margin-top: 16px;
        box-shadow: 0 4px 12px rgba(49, 130, 206, 0.3);
    }
    
    .register-button:hover:not(:disabled) {
        transform: translateY(-1px);
        box-shadow: 0 6px 16px rgba(49, 130, 206, 0.4);
    }
    
    .register-button:disabled {
        opacity: 0.6;
        cursor: not-allowed;
        transform: none;
    }
    
    .retry-button {
        display: flex;
        align-items: center;
        gap: 8px;
        background: linear-gradient(135deg, #ed8936 0%, #dd6b20 100%);
        color: white;
        border: none;
        padding: 10px 16px;
        border-radius: 8px;
        cursor: pointer;
        font-weight: 600;
        font-size: 14px;
        transition: all 0.3s ease;
        margin-top: 12px;
    }
    
    .retry-button:hover:not(:disabled) {
        transform: translateY(-1px);
        box-shadow: 0 4px 12px rgba(237, 137, 54, 0.3);
    }
    
    .spinner {
        width: 16px;
        height: 16px;
        border: 2px solid rgba(255, 255, 255, 0.3);
        border-top: 2px solid white;
        border-radius: 50%;
        animation: spin 1s linear infinite;
    }
    
    @keyframes spin {
        to { transform: rotate(360deg); }
    }
    
    .warning-message {
        display: flex;
        align-items: center;
        gap: 8px;
        background: #fef5e7;
        border: 1px solid #f6e05e;
        color: #975a16;
        padding: 12px;
        border-radius: 8px;
        font-size: 13px;
        margin-top: 12px;
    }
    
    .error-alert {
        display: flex;
        align-items: center;
        gap: 8px;
        background: #fed7d7;
        border: 1px solid #fc8181;
        color: #c53030;
        padding: 12px;
        border-radius: 8px;
        font-size: 13px;
        margin-top: 16px;
    }
    
    /* System Grid */
    .system-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
        gap: 16px;
    }
    
    .system-item {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 16px;
        background: #f7fafc;
        border-radius: 12px;
        border: 1px solid #e2e8f0;
        transition: all 0.3s ease;
    }
    
    .system-item:hover {
        background: #edf2f7;
        transform: translateY(-1px);
    }
    
    .system-icon {
        font-size: 24px;
        flex-shrink: 0;
    }
    
    .system-details {
        display: flex;
        flex-direction: column;
        min-width: 0;
    }
    
    .system-label {
        font-size: 12px;
        color: #718096;
        font-weight: 500;
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }
    
    .system-value {
        font-size: 14px;
        color: #2d3748;
        font-weight: 600;
        word-break: break-all;
    }
    
    .refresh-button {
        width: 32px;
        height: 32px;
        border-radius: 8px;
        background: #f7fafc;
        border: 1px solid #e2e8f0;
        display: flex;
        align-items: center;
        justify-content: center;
        cursor: pointer;
        transition: all 0.3s ease;
    }
    
    .refresh-button:hover {
        background: #edf2f7;
        transform: rotate(90deg);
    }
    
    /* Success Banner */
    .success-banner {
        background: linear-gradient(135deg, #10b981 0%, #059669 100%);
        color: white;
        border-radius: 16px;
        padding: 24px;
        box-shadow: 0 8px 25px rgba(16, 185, 129, 0.3);
        animation: slideIn 0.6s ease-out;
    }
    
    @keyframes slideIn {
        from {
            opacity: 0;
            transform: translateY(20px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }
    
    .banner-content {
        display: flex;
        align-items: center;
        gap: 20px;
    }
    
    .banner-icon {
        font-size: 48px;
        flex-shrink: 0;
    }
    
    .banner-text {
        flex: 1;
    }
    
    .banner-text h3 {
        margin: 0 0 8px 0;
        font-size: 24px;
        font-weight: 700;
    }
    
    .banner-text p {
        margin: 0;
        opacity: 0.9;
        line-height: 1.5;
    }
    
    .banner-status {
        flex-shrink: 0;
    }
    
    .status-indicator.active {
        display: flex;
        align-items: center;
        gap: 8px;
        background: rgba(255, 255, 255, 0.2);
        padding: 8px 16px;
        border-radius: 20px;
        font-weight: 600;
        font-size: 14px;
    }
    
    .indicator-dot {
        width: 8px;
        height: 8px;
        border-radius: 50%;
        background: white;
        animation: pulse 2s infinite;
    }
    
    /* Responsive Design */
    @media (max-width: 1200px) {
        .status-grid {
            grid-template-columns: 1fr;
        }
    }
    
    @media (max-width: 768px) {
        .dashboard-main {
            padding: 20px 16px;
        }
        
        .header-content {
            padding: 12px 16px;
            flex-direction: column;
            gap: 16px;
            align-items: stretch;
        }
        
        .header-right {
            justify-content: space-between;
        }
        
        .user-profile {
            flex: 1;
        }
        
        .logout-text {
            display: none;
        }
        
        .status-grid {
            grid-template-columns: 1fr;
            gap: 16px;
        }
        
        .card-header {
            padding: 16px 20px 12px 20px;
        }
        
        .card-content {
            padding: 16px 20px 20px 20px;
        }
        
        .system-grid {
            grid-template-columns: 1fr;
            gap: 12px;
        }
        
        .banner-content {
            flex-direction: column;
            text-align: center;
            gap: 16px;
        }
        
        .banner-text h3 {
            font-size: 20px;
        }
    }
    
    @media (max-width: 480px) {
        .status-grid {
            grid-template-columns: 1fr;
        }
        
        .logo-text h1 {
            font-size: 20px;
        }
        
        .user-avatar {
            width: 36px;
            height: 36px;
            font-size: 14px;
        }
        
        .system-item {
            padding: 12px;
        }
        
        .registration-success,
        .registration-pending,
        .registration-error {
            flex-direction: column;
            align-items: center;
            text-align: center;
            gap: 12px;
        }
    }
</style> 