<script>
    import { onMount, onDestroy } from 'svelte';
    import { 
        HandlePCRegistration, 
        GetSessionInfo, 
        GetSystemInfo, 
        IsConnected, 
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
    
    let sessionData = {};
    let systemInfo = {};
    let loading = false;
    let error = null;
    let registrationStatus = 'pending'; // 'pending', 'registering', 'registered', 'error'
    let connectionStatus = {
        isConnected: false,
        status: 'disconnected',
        lastHeartbeat: 0,
        serverUrl: '',
        connectionTime: 0,
        errorMessage: ''
    };
    
    // Variable para cleanup del event listener
    let eventCleanup = null;
    
    onMount(async () => {
        await loadInitialData();
        await checkRegistration();
        await updateConnectionStatus();
        
        // Esperar un poco antes de suscribirse a eventos para asegurar que el contexto esté listo
        setTimeout(() => {
            try {
                // Suscribirse a eventos de estado de conexión
                eventCleanup = EventsOn("connection_status_update", handleConnectionStatusUpdate);
                console.log("✅ Event listener registrado para connection_status_update");
            } catch (err) {
                console.error("❌ Error registrando event listener:", err);
            }
        }, 1000);
        
        // Verificar conexión cada 30 segundos como fallback
        setInterval(updateConnectionStatus, 30000);
    });
    
    onDestroy(() => {
        // Limpiar event listener
        if (eventCleanup) {
            try {
                EventsOff("connection_status_update");
                console.log("✅ Event listener removido correctamente");
            } catch (err) {
                console.error("❌ Error removiendo event listener:", err);
            }
        }
    });
    
    async function loadInitialData() {
        try {
            // Cargar información de sesión
            sessionData = await GetSessionInfo();
            userInfo.set({
                username: sessionData.username,
                userId: sessionData.userId,
                token: sessionData.token
            });
            
            // Cargar información del sistema
            systemInfo = await GetSystemInfo();
            
        } catch (err) {
            console.error('Error loading initial data:', err);
            setError('Error cargando datos iniciales');
        }
    }
    
    async function updateConnectionStatus() {
        try {
            // Usar IsConnected como base y construir el objeto connectionStatus
            const connected = await IsConnected();
            connectionStatus = {
                isConnected: connected,
                status: connected ? 'connected' : 'disconnected',
                lastHeartbeat: Date.now() / 1000,
                serverUrl: 'localhost:8080',
                connectionTime: Date.now() / 1000,
                errorMessage: ''
            };
            setConnected(connected);
            
            // Log para debug
            console.log('Connection status updated:', connectionStatus);
        } catch (err) {
            console.error('Error getting connection status:', err);
            connectionStatus = {
                isConnected: false,
                status: 'error',
                lastHeartbeat: Date.now() / 1000,
                serverUrl: 'localhost:8080',
                connectionTime: Date.now() / 1000,
                errorMessage: err.message
            };
            setConnected(false);
        }
    }
    
    function handleConnectionStatusUpdate(newStatus) {
        console.log("🔄 Connection status event received:", newStatus);
        
        if (!newStatus) {
            console.warn("⚠️ Event recibido sin datos");
            return;
        }
        
        connectionStatus = { ...newStatus };
        setConnected(newStatus.isConnected);
        
        // Mostrar notificación si hay cambio de estado
        if (newStatus.isConnected) {
            console.log('✅ Conexión establecida con el servidor');
        } else {
            console.log('❌ Conexión perdida con el servidor');
            if (newStatus.errorMessage) {
                console.log('💥 Error:', newStatus.errorMessage);
            }
        }
    }
    
    async function checkRegistration() {
        if (sessionData.pcId) {
            registrationStatus = 'registered';
            setRegistered(true, {
                pcId: sessionData.pcId,
                identifier: systemInfo.hostname || 'unknown'
            });
        } else {
            registrationStatus = 'pending';
            setRegistered(false);
        }
    }
    
    async function handleRegisterPC() {
        registrationStatus = 'registering';
        loading = true;
        error = null;
        setLoading(true);
        
        try {
            const result = await HandlePCRegistration();
            
            if (result.success) {
                registrationStatus = 'registered';
                setRegistered(true, {
                    pcId: result.pcId,
                    identifier: systemInfo.hostname || 'unknown'
                });
                
                // Actualizar datos de sesión
                sessionData = await GetSessionInfo();
            } else {
                registrationStatus = 'error';
                error = result.error || 'Error registrando PC';
                setError(error);
            }
        } catch (err) {
            registrationStatus = 'error';
            error = 'Error de conexión: ' + err.message;
            setError(error);
        } finally {
            loading = false;
            setLoading(false);
        }
    }
    
    async function handleLogout() {
        try {
            await Logout();
            setAuthenticated(false);
        } catch (err) {
            console.error('Error during logout:', err);
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
</script>

<div class="dashboard-container">
    <header class="dashboard-header">
        <div class="header-content">
            <h1>Escritorio Remoto - Cliente</h1>
            <div class="user-info">
                <span>Bienvenido, {sessionData.username || 'Usuario'}</span>
                <button class="logout-button" on:click={handleLogout}>
                    Cerrar Sesión
                </button>
            </div>
        </div>
    </header>
    
    <main class="dashboard-main">
        <div class="status-grid">
            <!-- Estado de Conexión -->
            <div class="status-card">
                <div class="status-header">
                    <h3>Estado de Conexión</h3>
                    <div class="status-indicator" style="background-color: {getConnectionStatusColor(connectionStatus.status)}">
                        <div class="status-dot" class:connected={connectionStatus.isConnected} class:disconnected={!connectionStatus.isConnected}></div>
                        {connectionStatus.isConnected ? 'Conectado' : 'Desconectado'}
                    </div>
                </div>
                <div class="status-details">
                    <div class="connection-info">
                        <div class="info-item">
                            <span class="info-label">Servidor:</span>
                            <span class="info-value">localhost:8080</span>
                        </div>
                        <div class="info-item">
                            <span class="info-label">Usuario ID:</span>
                            <span class="info-value">{sessionData.userId || 'N/A'}</span>
                        </div>
                        <div class="info-item">
                            <span class="info-label">Estado:</span>
                            <span class="info-value" style="color: {getConnectionStatusColor(connectionStatus.status)}">
                                {connectionStatus.status === 'connected' ? 'Conectado' : 
                                 connectionStatus.status === 'disconnected' ? 'Desconectado' : 'Error'}
                            </span>
                        </div>
                        <div class="info-item">
                            <span class="info-label">Último heartbeat:</span>
                            <span class="info-value">{formatTimestamp(connectionStatus.lastHeartbeat)}</span>
                        </div>
                        {#if connectionStatus.errorMessage}
                            <div class="info-item error">
                                <span class="info-label">Error:</span>
                                <span class="info-value error">{connectionStatus.errorMessage}</span>
                            </div>
                        {/if}
                    </div>
                </div>
            </div>
            
            <!-- Estado de Registro del PC -->
            <div class="status-card">
                <div class="status-header">
                    <h3>Registro del PC</h3>
                    <div class="status-indicator" style="background-color: {getStatusColor(registrationStatus)}">
                        {getStatusText(registrationStatus)}
                    </div>
                </div>
                <div class="status-details">
                    {#if registrationStatus === 'registered'}
                        <p><strong>PC ID:</strong> {sessionData.pcId}</p>
                        <p><strong>Identificador:</strong> {systemInfo.hostname || 'N/A'}</p>
                    {:else if registrationStatus === 'pending'}
                        <p>El PC no está registrado en el servidor</p>
                        <button 
                            class="register-button" 
                            on:click={handleRegisterPC}
                            disabled={loading || !$isConnected}
                        >
                            {#if loading}
                                <span class="spinner"></span>
                                Registrando...
                            {:else}
                                Registrar PC
                            {/if}
                        </button>
                    {:else if registrationStatus === 'error'}
                        <p class="error-text">{error}</p>
                        <button 
                            class="register-button retry" 
                            on:click={handleRegisterPC}
                            disabled={loading || !$isConnected}
                        >
                            Reintentar
                        </button>
                    {/if}
                </div>
            </div>
            
            <!-- Información del Sistema -->
            <div class="status-card">
                <div class="status-header">
                    <h3>Información del Sistema</h3>
                </div>
                <div class="status-details">
                    <div class="system-info">
                        <div class="info-item">
                            <span class="info-label">Hostname:</span>
                            <span class="info-value">{systemInfo.hostname || 'N/A'}</span>
                        </div>
                        <div class="info-item">
                            <span class="info-label">Sistema:</span>
                            <span class="info-value">{systemInfo.os || 'N/A'}</span>
                        </div>
                        <div class="info-item">
                            <span class="info-label">Arquitectura:</span>
                            <span class="info-value">{systemInfo.arch || 'N/A'}</span>
                        </div>
                        <div class="info-item">
                            <span class="info-label">CPUs:</span>
                            <span class="info-value">{systemInfo.num_cpu || 'N/A'}</span>
                        </div>
                        <div class="info-item">
                            <span class="info-label">IP Local:</span>
                            <span class="info-value">{systemInfo.local_ip || 'N/A'}</span>
                        </div>
                    </div>
                </div>
            </div>
        </div>
        
        {#if registrationStatus === 'registered'}
            <div class="success-message">
                <h2>🎉 ¡PC Registrado Exitosamente!</h2>
                <p>Tu PC está ahora registrado y conectado al servidor. Puedes ser controlado remotamente por usuarios autorizados.</p>
            </div>
        {/if}
    </main>
</div>

<style>
    .dashboard-container {
        min-height: 100vh;
        background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
    }
    
    .dashboard-header {
        background: white;
        box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
        padding: 0;
    }
    
    .header-content {
        max-width: 1200px;
        margin: 0 auto;
        padding: 20px;
        display: flex;
        justify-content: space-between;
        align-items: center;
    }
    
    .header-content h1 {
        color: #333;
        margin: 0;
        font-size: 24px;
        font-weight: 600;
    }
    
    .user-info {
        display: flex;
        align-items: center;
        gap: 15px;
        color: #666;
    }
    
    .logout-button {
        background: #ef4444;
        color: white;
        border: none;
        padding: 8px 16px;
        border-radius: 6px;
        cursor: pointer;
        font-size: 14px;
        transition: background 0.3s ease;
    }
    
    .logout-button:hover {
        background: #dc2626;
    }
    
    .dashboard-main {
        max-width: 1200px;
        margin: 0 auto;
        padding: 30px 20px;
    }
    
    .status-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(350px, 1fr));
        gap: 25px;
        margin-bottom: 30px;
    }
    
    .status-card {
        background: white;
        border-radius: 12px;
        box-shadow: 0 4px 15px rgba(0, 0, 0, 0.1);
        padding: 25px;
        transition: transform 0.3s ease;
    }
    
    .status-card:hover {
        transform: translateY(-2px);
    }
    
    .status-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 20px;
        padding-bottom: 15px;
        border-bottom: 2px solid #f1f5f9;
    }
    
    .status-header h3 {
        margin: 0;
        color: #333;
        font-size: 18px;
        font-weight: 600;
    }
    
    .status-indicator {
        padding: 6px 12px;
        border-radius: 20px;
        font-size: 12px;
        font-weight: 600;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        display: flex;
        align-items: center;
        gap: 6px;
        color: white;
    }
    
    .status-dot {
        width: 8px;
        height: 8px;
        border-radius: 50%;
        animation: pulse 2s infinite;
    }
    
    .status-dot.connected {
        background: #10b981;
    }
    
    .status-dot.disconnected {
        background: #ef4444;
    }
    
    @keyframes pulse {
        0%, 100% { opacity: 1; }
        50% { opacity: 0.6; }
    }
    
    .connection-info {
        display: flex;
        flex-direction: column;
        gap: 8px;
    }
    
    .info-item {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 4px 0;
    }
    
    .info-item.error {
        margin-top: 8px;
        padding: 8px;
        background: #fef2f2;
        border-radius: 6px;
        border: 1px solid #fecaca;
    }
    
    .info-label {
        font-weight: 500;
        color: #4b5563;
        font-size: 14px;
    }
    
    .info-value {
        font-weight: 600;
        color: #1f2937;
        font-size: 14px;
    }
    
    .info-value.error {
        color: #dc2626;
        font-size: 12px;
    }
    
    .status-details {
        color: #666;
        line-height: 1.6;
    }
    
    .status-details p {
        margin: 8px 0;
    }
    
    .register-button {
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        color: white;
        border: none;
        padding: 12px 20px;
        border-radius: 8px;
        cursor: pointer;
        font-size: 14px;
        font-weight: 600;
        margin-top: 15px;
        display: flex;
        align-items: center;
        gap: 8px;
        transition: all 0.3s ease;
    }
    
    .register-button:hover:not(:disabled) {
        transform: translateY(-1px);
        box-shadow: 0 4px 15px rgba(102, 126, 234, 0.3);
    }
    
    .register-button:disabled {
        opacity: 0.6;
        cursor: not-allowed;
        transform: none;
    }
    
    .register-button.retry {
        background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
    }
    
    .system-info {
        display: flex;
        flex-direction: column;
        gap: 12px;
    }
    
    .info-item {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 8px 0;
        border-bottom: 1px solid #f1f5f9;
    }
    
    .info-item:last-child {
        border-bottom: none;
    }
    
    .info-label {
        font-weight: 500;
        color: #374151;
    }
    
    .info-value {
        color: #6b7280;
        font-family: monospace;
        background: #f8fafc;
        padding: 4px 8px;
        border-radius: 4px;
        font-size: 13px;
    }
    
    .success-message {
        background: linear-gradient(135deg, #10b981 0%, #059669 100%);
        color: white;
        padding: 30px;
        border-radius: 12px;
        text-align: center;
        box-shadow: 0 8px 25px rgba(16, 185, 129, 0.3);
    }
    
    .success-message h2 {
        margin: 0 0 15px 0;
        font-size: 24px;
        font-weight: 600;
    }
    
    .success-message p {
        margin: 0;
        font-size: 16px;
        opacity: 0.9;
        line-height: 1.6;
    }
    
    .error-text {
        color: #ef4444;
        font-size: 14px;
        margin: 10px 0;
    }
    
    .spinner {
        width: 14px;
        height: 14px;
        border: 2px solid transparent;
        border-top: 2px solid currentColor;
        border-radius: 50%;
        animation: spin 1s linear infinite;
    }
    
    @keyframes spin {
        to {
            transform: rotate(360deg);
        }
    }
</style> 