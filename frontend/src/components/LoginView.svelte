<script>
    import { Login } from '../../wailsjs/go/main/App.js';
    import { setAuthenticated, setLoading, setError, clearError } from '../stores/app.js';
    
    let username = '';
    let password = '';
    let loading = false;
    let error = null;
    let connectionStep = 'ready'; // ready, connecting, authenticating, success, error
    
    // Mensajes de estado para cada paso
    const stepMessages = {
        ready: 'Listo para conectar',
        connecting: 'Conectando al servidor...',
        authenticating: 'Autenticando credenciales...',
        success: '¡Conectado exitosamente!',
        error: 'Error de conexión'
    };
    
    async function handleLogin() {
        if (!username.trim() || !password.trim()) {
            error = 'Por favor ingresa usuario y contraseña';
            return;
        }
        
        loading = true;
        error = null;
        connectionStep = 'connecting';
        setLoading(true);
        clearError();
        
        try {
            // Simular progreso de conexión
            await new Promise(resolve => setTimeout(resolve, 1000));
            connectionStep = 'authenticating';
            
            const result = await Login(username, password);
            
            if (result.success) {
                connectionStep = 'success';
                await new Promise(resolve => setTimeout(resolve, 500));
                
                setAuthenticated(true, {
                    username: result.user?.username || username,
                    userId: result.user?.id || '',
                    sessionId: result.session_id || '',
                    serverUrl: result.server_url || 'localhost:8080'
                });
            } else {
                connectionStep = 'error';
                error = result.error || 'Error de autenticación';
                setError(error);
            }
        } catch (err) {
            connectionStep = 'error';
            error = 'Error de conexión: ' + err.message;
            setError(error);
        } finally {
            if (connectionStep !== 'success') {
                loading = false;
                setLoading(false);
                // Reset estado después de un tiempo
                setTimeout(() => {
                    if (connectionStep === 'error') {
                        connectionStep = 'ready';
                    }
                }, 3000);
            }
        }
    }
    
    function handleKeyPress(event) {
        if (event.key === 'Enter' && !loading) {
            handleLogin();
        }
    }

    // Resetear estado de conexión cuando se cambian las credenciales
    $: if (username || password) {
        if (connectionStep === 'error') {
            connectionStep = 'ready';
            error = null;
        }
    }
</script>

<div class="login-container">
    <div class="login-background">
        <div class="background-pattern"></div>
        <div class="background-gradient"></div>
    </div>
    
    <div class="login-content">
        <div class="login-card">
            <div class="login-header">
                <div class="logo">
                    <div class="logo-icon">🖥️</div>
                    <div class="logo-text">
                        <h1>Escritorio Remoto</h1>
                        <p>Cliente de Conexión</p>
                    </div>
                </div>
            </div>
            
            <form class="login-form" on:submit|preventDefault={handleLogin}>
                <div class="form-group">
                    <label for="username">Usuario</label>
                    <div class="input-wrapper">
                        <input
                            id="username"
                            type="text"
                            bind:value={username}
                            on:keypress={handleKeyPress}
                            placeholder="Ingresa tu nombre de usuario"
                            disabled={loading}
                            required
                            autocomplete="username"
                        />
                        <div class="input-icon">👤</div>
                    </div>
                </div>
                
                <div class="form-group">
                    <label for="password">Contraseña</label>
                    <div class="input-wrapper">
                        <input
                            id="password"
                            type="password"
                            bind:value={password}
                            on:keypress={handleKeyPress}
                            placeholder="Ingresa tu contraseña"
                            disabled={loading}
                            required
                            autocomplete="current-password"
                        />
                        <div class="input-icon">🔒</div>
                    </div>
                </div>
                
                <!-- Estado de conexión -->
                <div class="connection-status" class:visible={loading || error}>
                    <div class="status-indicator" class:connecting={connectionStep === 'connecting'} 
                         class:authenticating={connectionStep === 'authenticating'} 
                         class:success={connectionStep === 'success'} 
                         class:error={connectionStep === 'error'}>
                        {#if connectionStep === 'connecting'}
                            <div class="spinner"></div>
                        {:else if connectionStep === 'authenticating'}
                            <div class="authenticating-icon">🔐</div>
                        {:else if connectionStep === 'success'}
                            <div class="success-icon">✅</div>
                        {:else if connectionStep === 'error'}
                            <div class="error-icon">❌</div>
                        {/if}
                        <span class="status-text">{stepMessages[connectionStep]}</span>
                    </div>
                </div>
                
                {#if error}
                    <div class="error-message">
                        <span class="error-icon">⚠️</span>
                        <span class="error-text">{error}</span>
                    </div>
                {/if}
                
                <button 
                    type="submit" 
                    class="login-button"
                    disabled={loading || !username.trim() || !password.trim()}
                    class:loading={loading}
                    class:success={connectionStep === 'success'}
                >
                    {#if loading}
                        <span class="button-content">
                            <span class="spinner small"></span>
                            {stepMessages[connectionStep]}
                        </span>
                    {:else if connectionStep === 'success'}
                        <span class="button-content">
                            <span class="success-icon">✅</span>
                            ¡Conectado!
                        </span>
                    {:else}
                        <span class="button-content">
                            <span class="login-icon">🚀</span>
                            Iniciar Sesión
                        </span>
                    {/if}
                </button>
            </form>
            
            <div class="login-footer">
                <div class="server-info">
                    <span class="server-label">Servidor:</span>
                    <span class="server-value">localhost:8080</span>
                </div>
            </div>
        </div>
    </div>
</div>

<style>
    .login-container {
        position: relative;
        min-height: 100vh;
        display: flex;
        align-items: center;
        justify-content: center;
        padding: 20px;
        overflow: hidden;
    }
    
    .login-background {
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        z-index: 0;
    }
    
    .background-gradient {
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    }
    
    .background-pattern {
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background-image: 
            radial-gradient(circle at 20% 50%, rgba(255, 255, 255, 0.1) 0%, transparent 50%),
            radial-gradient(circle at 80% 20%, rgba(255, 255, 255, 0.1) 0%, transparent 50%),
            radial-gradient(circle at 40% 80%, rgba(255, 255, 255, 0.1) 0%, transparent 50%);
        animation: float 6s ease-in-out infinite;
    }
    
    @keyframes float {
        0%, 100% { transform: translateY(0px); }
        50% { transform: translateY(-10px); }
    }
    
    .login-content {
        position: relative;
        z-index: 1;
        width: 100%;
        max-width: 480px;
    }
    
    .login-card {
        background: rgba(255, 255, 255, 0.95);
        backdrop-filter: blur(20px);
        border-radius: 24px;
        box-shadow: 
            0 20px 40px rgba(0, 0, 0, 0.1),
            0 0 0 1px rgba(255, 255, 255, 0.2);
        padding: 40px;
        animation: slideUp 0.6s ease-out;
    }
    
    @keyframes slideUp {
        from {
            opacity: 0;
            transform: translateY(30px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }
    
    .login-header {
        text-align: center;
        margin-bottom: 40px;
    }
    
    .logo {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 16px;
        margin-bottom: 8px;
    }
    
    .logo-icon {
        font-size: 48px;
        filter: drop-shadow(0 4px 8px rgba(0, 0, 0, 0.1));
    }
    
    .logo-text h1 {
        color: #2d3748;
        margin: 0;
        font-size: 28px;
        font-weight: 700;
        letter-spacing: -0.5px;
    }
    
    .logo-text p {
        color: #718096;
        margin: 4px 0 0 0;
        font-size: 16px;
        font-weight: 500;
    }
    
    .login-form {
        display: flex;
        flex-direction: column;
        gap: 24px;
    }
    
    .form-group {
        display: flex;
        flex-direction: column;
        gap: 8px;
    }
    
    .form-group label {
        color: #2d3748;
        font-weight: 600;
        font-size: 14px;
        letter-spacing: 0.5px;
        text-transform: uppercase;
    }
    
    .input-wrapper {
        position: relative;
        display: flex;
        align-items: center;
    }
    
    .input-wrapper input {
        width: 100%;
        padding: 16px 20px;
        padding-right: 50px;
        border: 2px solid #e2e8f0;
        border-radius: 12px;
        font-size: 16px;
        font-weight: 500;
        background: #f7fafc;
        transition: all 0.3s ease;
        outline: none;
    }
    
    .input-wrapper input:focus {
        border-color: #667eea;
        background: white;
        box-shadow: 
            0 0 0 4px rgba(102, 126, 234, 0.1),
            0 4px 12px rgba(0, 0, 0, 0.05);
        transform: translateY(-1px);
    }
    
    .input-wrapper input:disabled {
        background: #edf2f7;
        color: #a0aec0;
        cursor: not-allowed;
    }
    
    .input-icon {
        position: absolute;
        right: 16px;
        font-size: 18px;
        opacity: 0.6;
        pointer-events: none;
    }
    
    .connection-status {
        opacity: 0;
        visibility: hidden;
        transform: translateY(-10px);
        transition: all 0.3s ease;
        margin: -8px 0;
    }
    
    .connection-status.visible {
        opacity: 1;
        visibility: visible;
        transform: translateY(0);
    }
    
    .status-indicator {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 12px 16px;
        border-radius: 12px;
        font-size: 14px;
        font-weight: 600;
        transition: all 0.3s ease;
    }
    
    .status-indicator.connecting {
        background: linear-gradient(135deg, #3182ce 0%, #2b6cb0 100%);
        color: white;
    }
    
    .status-indicator.authenticating {
        background: linear-gradient(135deg, #ed8936 0%, #dd6b20 100%);
        color: white;
    }
    
    .status-indicator.success {
        background: linear-gradient(135deg, #38a169 0%, #2f855a 100%);
        color: white;
    }
    
    .status-indicator.error {
        background: linear-gradient(135deg, #e53e3e 0%, #c53030 100%);
        color: white;
    }
    
    .spinner {
        width: 20px;
        height: 20px;
        border: 3px solid rgba(255, 255, 255, 0.3);
        border-top: 3px solid white;
        border-radius: 50%;
        animation: spin 1s linear infinite;
    }
    
    .spinner.small {
        width: 16px;
        height: 16px;
        border-width: 2px;
    }
    
    @keyframes spin {
        to { transform: rotate(360deg); }
    }
    
    .authenticating-icon,
    .success-icon,
    .error-icon {
        font-size: 20px;
        animation: pulse 1s ease-in-out infinite;
    }
    
    @keyframes pulse {
        0%, 100% { transform: scale(1); }
        50% { transform: scale(1.1); }
    }
    
    .error-message {
        display: flex;
        align-items: center;
        gap: 12px;
        background: linear-gradient(135deg, #fed7d7 0%, #feb2b2 100%);
        border: 1px solid #fc8181;
        color: #c53030;
        padding: 16px 20px;
        border-radius: 12px;
        font-size: 14px;
        font-weight: 600;
        animation: shake 0.5s ease-in-out;
    }
    
    @keyframes shake {
        0%, 100% { transform: translateX(0); }
        25% { transform: translateX(-5px); }
        75% { transform: translateX(5px); }
    }
    
    .login-button {
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        color: white;
        border: none;
        padding: 18px 32px;
        border-radius: 12px;
        font-size: 16px;
        font-weight: 700;
        cursor: pointer;
        transition: all 0.3s ease;
        margin-top: 16px;
        box-shadow: 0 8px 20px rgba(102, 126, 234, 0.3);
    }
    
    .login-button:hover:not(:disabled) {
        transform: translateY(-2px);
        box-shadow: 0 12px 30px rgba(102, 126, 234, 0.4);
    }
    
    .login-button:active:not(:disabled) {
        transform: translateY(0);
    }
    
    .login-button:disabled {
        opacity: 0.6;
        cursor: not-allowed;
        transform: none;
        box-shadow: 0 4px 12px rgba(102, 126, 234, 0.2);
    }
    
    .login-button.loading {
        pointer-events: none;
    }
    
    .login-button.success {
        background: linear-gradient(135deg, #38a169 0%, #2f855a 100%);
        box-shadow: 0 8px 20px rgba(56, 161, 105, 0.3);
    }
    
    .button-content {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 10px;
    }
    
    .login-footer {
        margin-top: 32px;
        text-align: center;
        padding-top: 24px;
        border-top: 1px solid #e2e8f0;
    }
    
    .server-info {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
        font-size: 14px;
    }
    
    .server-label {
        color: #718096;
        font-weight: 500;
    }
    
    .server-value {
        color: #2d3748;
        font-weight: 700;
        font-family: 'Courier New', monospace;
        background: #edf2f7;
        padding: 4px 8px;
        border-radius: 6px;
    }
    
    /* Responsive Design */
    @media (max-width: 768px) {
        .login-container {
            padding: 16px;
        }
        
        .login-card {
            padding: 32px 24px;
        }
        
        .logo {
            flex-direction: column;
            gap: 12px;
        }
        
        .logo-icon {
            font-size: 40px;
        }
        
        .logo-text h1 {
            font-size: 24px;
        }
        
        .form-group input {
            padding: 14px 18px;
            padding-right: 48px;
            font-size: 16px; /* Prevent zoom on iOS */
        }
    }
    
    @media (max-width: 480px) {
        .login-card {
            padding: 24px 20px;
            border-radius: 20px;
        }
        
        .login-form {
            gap: 20px;
        }
        
        .logo-text h1 {
            font-size: 22px;
        }
        
        .logo-text p {
            font-size: 14px;
        }
    }
</style> 