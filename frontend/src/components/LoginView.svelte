<script>
    import { HandleClientLogin } from '../../wailsjs/go/main/App.js';
    import { setAuthenticated, setLoading, setError, clearError } from '../stores/app.js';
    
    let username = '';
    let password = '';
    let loading = false;
    let error = null;
    
    async function handleLogin() {
        if (!username.trim() || !password.trim()) {
            error = 'Por favor ingresa usuario y contraseña';
            return;
        }
        
        loading = true;
        error = null;
        setLoading(true);
        clearError();
        
        try {
            const result = await HandleClientLogin(username, password);
            
            if (result.success) {
                setAuthenticated(true, {
                    username: username,
                    userId: result.userId,
                    token: result.token
                });
            } else {
                error = result.error || 'Error de autenticación';
                setError(error);
            }
        } catch (err) {
            error = 'Error de conexión: ' + err.message;
            setError(error);
        } finally {
            loading = false;
            setLoading(false);
        }
    }
    
    function handleKeyPress(event) {
        if (event.key === 'Enter') {
            handleLogin();
        }
    }
</script>

<div class="login-container">
    <div class="login-card">
        <div class="login-header">
            <h1>Escritorio Remoto</h1>
            <p>Cliente de Conexión</p>
        </div>
        
        <form class="login-form" on:submit|preventDefault={handleLogin}>
            <div class="form-group">
                <label for="username">Usuario:</label>
                <input
                    id="username"
                    type="text"
                    bind:value={username}
                    on:keypress={handleKeyPress}
                    placeholder="Ingresa tu usuario"
                    disabled={loading}
                    required
                />
            </div>
            
            <div class="form-group">
                <label for="password">Contraseña:</label>
                <input
                    id="password"
                    type="password"
                    bind:value={password}
                    on:keypress={handleKeyPress}
                    placeholder="Ingresa tu contraseña"
                    disabled={loading}
                    required
                />
            </div>
            
            {#if error}
                <div class="error-message">
                    {error}
                </div>
            {/if}
            
            <button 
                type="submit" 
                class="login-button"
                disabled={loading || !username.trim() || !password.trim()}
            >
                {#if loading}
                    <span class="spinner"></span>
                    Conectando...
                {:else}
                    Iniciar Sesión
                {/if}
            </button>
        </form>
    </div>
</div>

<style>
    .login-container {
        display: flex;
        justify-content: center;
        align-items: center;
        min-height: 100vh;
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        padding: 20px;
    }
    
    .login-card {
        background: white;
        border-radius: 12px;
        box-shadow: 0 15px 35px rgba(0, 0, 0, 0.1);
        padding: 40px;
        width: 100%;
        max-width: 400px;
        animation: slideIn 0.3s ease-out;
    }
    
    @keyframes slideIn {
        from {
            opacity: 0;
            transform: translateY(-20px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }
    
    .login-header {
        text-align: center;
        margin-bottom: 30px;
    }
    
    .login-header h1 {
        color: #333;
        margin: 0 0 10px 0;
        font-size: 28px;
        font-weight: 600;
    }
    
    .login-header p {
        color: #666;
        margin: 0;
        font-size: 16px;
    }
    
    .login-form {
        display: flex;
        flex-direction: column;
        gap: 20px;
    }
    
    .form-group {
        display: flex;
        flex-direction: column;
        gap: 8px;
    }
    
    .form-group label {
        color: #333;
        font-weight: 500;
        font-size: 14px;
    }
    
    .form-group input {
        padding: 12px 16px;
        border: 2px solid #e1e5e9;
        border-radius: 8px;
        font-size: 16px;
        transition: all 0.3s ease;
        background: #f8f9fa;
    }
    
    .form-group input:focus {
        outline: none;
        border-color: #667eea;
        background: white;
        box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
    }
    
    .form-group input:disabled {
        background: #f5f5f5;
        color: #999;
        cursor: not-allowed;
    }
    
    .error-message {
        background: #fee;
        color: #c53030;
        padding: 12px 16px;
        border-radius: 8px;
        border: 1px solid #fed7d7;
        font-size: 14px;
        text-align: center;
    }
    
    .login-button {
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        color: white;
        border: none;
        padding: 14px 24px;
        border-radius: 8px;
        font-size: 16px;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.3s ease;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
        margin-top: 10px;
    }
    
    .login-button:hover:not(:disabled) {
        transform: translateY(-2px);
        box-shadow: 0 8px 25px rgba(102, 126, 234, 0.3);
    }
    
    .login-button:active:not(:disabled) {
        transform: translateY(0);
    }
    
    .login-button:disabled {
        opacity: 0.6;
        cursor: not-allowed;
        transform: none;
        box-shadow: none;
    }
    
    .spinner {
        width: 16px;
        height: 16px;
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