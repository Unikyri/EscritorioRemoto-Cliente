<script>
  import { createEventDispatcher } from 'svelte';
  import { AcceptControlRequest, RejectControlRequest } from '../../wailsjs/go/main/App.js';

  export let visible = false;
  export let adminUsername = '';
  export let sessionId = '';

  const dispatch = createEventDispatcher();

  let processing = false;
  let error = '';

  async function acceptRequest() {
    if (processing) return;
    
    processing = true;
    error = '';
    
    try {
      const result = await AcceptControlRequest(sessionId);
      if (result.success) {
        dispatch('accepted', { sessionId });
        closeDialog();
      } else {
        error = result.error || 'Error al aceptar la solicitud';
      }
    } catch (err) {
      error = 'Error de conexión: ' + err.message;
    } finally {
      processing = false;
    }
  }

  async function rejectRequest() {
    if (processing) return;
    
    processing = true;
    error = '';
    
    try {
      const result = await RejectControlRequest(sessionId, 'Usuario rechazó la solicitud');
      if (result.success) {
        dispatch('rejected', { sessionId });
        closeDialog();
      } else {
        error = result.error || 'Error al rechazar la solicitud';
      }
    } catch (err) {
      error = 'Error de conexión: ' + err.message;
    } finally {
      processing = false;
    }
  }

  function closeDialog() {
    visible = false;
    error = '';
    processing = false;
  }

  // Cerrar con Escape
  function handleKeydown(event) {
    if (event.key === 'Escape' && !processing) {
      rejectRequest();
    }
  }
</script>

<svelte:window on:keydown={handleKeydown} />

{#if visible}
  <div class="dialog-overlay" on:click={rejectRequest}>
    <div class="dialog" on:click|stopPropagation>
      <div class="dialog-header">
        <h2>🖥️ Solicitud de Control Remoto</h2>
      </div>
      
      <div class="dialog-content">
        <div class="admin-info">
          <div class="admin-avatar">
            <span class="admin-initial">{adminUsername.charAt(0).toUpperCase()}</span>
          </div>
          <div class="admin-details">
            <p class="admin-name"><strong>{adminUsername}</strong></p>
            <p class="request-text">desea controlar remotamente su PC</p>
          </div>
        </div>

        <div class="warning-box">
          <div class="warning-icon">⚠️</div>
          <div class="warning-text">
            <p><strong>¡Atención!</strong></p>
            <p>El administrador podrá ver y controlar su pantalla durante la sesión.</p>
          </div>
        </div>

        {#if error}
          <div class="error-message">
            <span class="error-icon">❌</span>
            {error}
          </div>
        {/if}
      </div>

      <div class="dialog-actions">
        <button 
          class="btn btn-reject" 
          on:click={rejectRequest}
          disabled={processing}
        >
          {processing ? 'Procesando...' : 'Rechazar'}
        </button>
        
        <button 
          class="btn btn-accept" 
          on:click={acceptRequest}
          disabled={processing}
        >
          {processing ? 'Procesando...' : 'Aceptar'}
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .dialog-overlay {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: rgba(0, 0, 0, 0.7);
    display: flex;
    justify-content: center;
    align-items: center;
    z-index: 1000;
    backdrop-filter: blur(4px);
  }

  .dialog {
    background: white;
    border-radius: 16px;
    box-shadow: 0 20px 40px rgba(0, 0, 0, 0.3);
    max-width: 480px;
    width: 90%;
    max-height: 90vh;
    overflow: hidden;
    animation: slideIn 0.3s ease-out;
  }

  @keyframes slideIn {
    from {
      opacity: 0;
      transform: translateY(-20px) scale(0.95);
    }
    to {
      opacity: 1;
      transform: translateY(0) scale(1);
    }
  }

  .dialog-header {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: white;
    padding: 20px;
    text-align: center;
  }

  .dialog-header h2 {
    margin: 0;
    font-size: 20px;
    font-weight: 600;
  }

  .dialog-content {
    padding: 24px;
  }

  .admin-info {
    display: flex;
    align-items: center;
    margin-bottom: 20px;
    padding: 16px;
    background: #f8f9fa;
    border-radius: 12px;
    border-left: 4px solid #667eea;
  }

  .admin-avatar {
    width: 50px;
    height: 50px;
    border-radius: 50%;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    display: flex;
    align-items: center;
    justify-content: center;
    margin-right: 16px;
    flex-shrink: 0;
  }

  .admin-initial {
    color: white;
    font-size: 20px;
    font-weight: bold;
  }

  .admin-details {
    flex: 1;
  }

  .admin-name {
    margin: 0 0 4px 0;
    font-size: 18px;
    color: #2c3e50;
  }

  .request-text {
    margin: 0;
    color: #6c757d;
    font-size: 14px;
  }

  .warning-box {
    display: flex;
    align-items: flex-start;
    background: #fff3cd;
    border: 1px solid #ffeaa7;
    border-radius: 8px;
    padding: 16px;
    margin-bottom: 20px;
  }

  .warning-icon {
    font-size: 20px;
    margin-right: 12px;
    flex-shrink: 0;
  }

  .warning-text p {
    margin: 0 0 8px 0;
    color: #856404;
    font-size: 14px;
  }

  .warning-text p:last-child {
    margin-bottom: 0;
  }

  .error-message {
    display: flex;
    align-items: center;
    background: #f8d7da;
    border: 1px solid #f5c6cb;
    border-radius: 8px;
    padding: 12px;
    margin-bottom: 16px;
    color: #721c24;
    font-size: 14px;
  }

  .error-icon {
    margin-right: 8px;
  }

  .dialog-actions {
    display: flex;
    gap: 12px;
    padding: 0 24px 24px 24px;
  }

  .btn {
    flex: 1;
    padding: 12px 24px;
    border: none;
    border-radius: 8px;
    font-size: 16px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s ease;
    position: relative;
    overflow: hidden;
  }

  .btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .btn-reject {
    background: #dc3545;
    color: white;
  }

  .btn-reject:hover:not(:disabled) {
    background: #c82333;
    transform: translateY(-1px);
    box-shadow: 0 4px 12px rgba(220, 53, 69, 0.3);
  }

  .btn-accept {
    background: #28a745;
    color: white;
  }

  .btn-accept:hover:not(:disabled) {
    background: #218838;
    transform: translateY(-1px);
    box-shadow: 0 4px 12px rgba(40, 167, 69, 0.3);
  }

  .btn:active:not(:disabled) {
    transform: translateY(0);
  }
</style> 