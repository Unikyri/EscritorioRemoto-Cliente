<script>
    import { createEventDispatcher, onMount, onDestroy } from 'svelte';
    import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime.js';

    const dispatch = createEventDispatcher();
    
    let notifications = [];
    let isNotificationVisible = false;
    let currentNotification = null;

    // Configuración de notificaciones
    const NOTIFICATION_DURATION = 5000; // 5 segundos

    onMount(() => {
        // Escuchar eventos de archivos recibidos
        EventsOn('file_received', handleFileReceived);
        EventsOn('file_transfer_failed', handleFileTransferFailed);
    });

    onDestroy(() => {
        EventsOff('file_received');
        EventsOff('file_transfer_failed');
    });

    function handleFileReceived(data) {
        console.log('📁 File received:', data);
        
        const notification = {
            id: Date.now(),
            type: 'success',
            title: 'Archivo Recibido',
            message: `Se ha recibido el archivo: ${data.file_name}`,
            filePath: data.file_path,
            fileName: data.file_name,
            timestamp: new Date().toLocaleTimeString()
        };

        showNotificationToUser(notification);
    }

    function handleFileTransferFailed(data) {
        console.log('❌ File transfer failed:', data);
        
        const notification = {
            id: Date.now(),
            type: 'error',
            title: 'Error en Transferencia',
            message: `Error al recibir archivo ${data.file_name}: ${data.error}`,
            fileName: data.file_name,
            error: data.error,
            timestamp: new Date().toLocaleTimeString()
        };

        showNotificationToUser(notification);
    }

    function showNotificationToUser(notification) {
        notifications = [notification, ...notifications];
        currentNotification = notification;
        isNotificationVisible = true;

        // Auto-ocultar después del tiempo configurado
        setTimeout(() => {
            hideNotification();
        }, NOTIFICATION_DURATION);
    }

    function hideNotification() {
        isNotificationVisible = false;
        currentNotification = null;
    }

    function openFileLocation(filePath) {
        if (filePath) {
            // Para MVP, solo mostrar la ruta en consola
            console.log('📂 Opening file location:', filePath);
            alert(`Archivo guardado en: ${filePath}`);
        }
    }

    function clearNotifications() {
        notifications = [];
    }

    // Exponer función para que componentes padres puedan mostrar notificaciones
    export function showNotification(type, title, message, extra = {}) {
        const notification = {
            id: Date.now(),
            type,
            title,
            message,
            timestamp: new Date().toLocaleTimeString(),
            ...extra
        };
        showNotificationToUser(notification);
    }
</script>

<!-- Notificación flotante -->
{#if isNotificationVisible && currentNotification}
    <div class="notification-overlay">
        <div class="notification {currentNotification.type}">
            <div class="notification-header">
                <div class="notification-icon">
                    {#if currentNotification.type === 'success'}
                        📁
                    {:else if currentNotification.type === 'error'}
                        ❌
                    {:else}
                        ℹ️
                    {/if}
                </div>
                <div class="notification-title">
                    {currentNotification.title}
                </div>
                <button class="close-btn" on:click={hideNotification}>×</button>
            </div>
            
            <div class="notification-body">
                <p class="notification-message">{currentNotification.message}</p>
                {#if currentNotification.filePath}
                    <button 
                        class="action-btn" 
                        on:click={() => openFileLocation(currentNotification.filePath)}>
                        📂 Abrir Ubicación
                    </button>
                {/if}
                <div class="notification-timestamp">
                    {currentNotification.timestamp}
                </div>
            </div>
        </div>
    </div>
{/if}

<!-- Lista de notificaciones (opcional, para historial) -->
{#if notifications.length > 0}
    <div class="notifications-list">
        <div class="notifications-header">
            <span class="notifications-title">Transferencias Recientes</span>
            <button class="clear-btn" on:click={clearNotifications}>Limpiar</button>
        </div>
        
        {#each notifications.slice(0, 5) as notification}
            <div class="notification-item {notification.type}">
                <div class="item-icon">
                    {#if notification.type === 'success'}
                        ✅
                    {:else}
                        ❌
                    {/if}
                </div>
                <div class="item-content">
                    <div class="item-title">{notification.title}</div>
                    <div class="item-message">{notification.message}</div>
                    <div class="item-timestamp">{notification.timestamp}</div>
                </div>
            </div>
        {/each}
    </div>
{/if}

<style>
    .notification-overlay {
        position: fixed;
        top: 20px;
        right: 20px;
        z-index: 10000;
        animation: slideIn 0.3s ease-out;
    }

    .notification {
        background: white;
        border-radius: 8px;
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
        border-left: 4px solid #007bff;
        min-width: 320px;
        max-width: 400px;
    }

    .notification.success {
        border-left-color: #28a745;
    }

    .notification.error {
        border-left-color: #dc3545;
    }

    .notification-header {
        display: flex;
        align-items: center;
        padding: 12px 16px 8px;
        border-bottom: 1px solid #e9ecef;
    }

    .notification-icon {
        font-size: 20px;
        margin-right: 8px;
    }

    .notification-title {
        flex: 1;
        font-weight: 600;
        color: #343a40;
    }

    .close-btn {
        background: none;
        border: none;
        font-size: 18px;
        cursor: pointer;
        color: #6c757d;
        padding: 0;
        width: 24px;
        height: 24px;
        display: flex;
        align-items: center;
        justify-content: center;
    }

    .close-btn:hover {
        color: #dc3545;
    }

    .notification-body {
        padding: 8px 16px 12px;
    }

    .notification-message {
        margin: 0 0 12px;
        color: #495057;
        line-height: 1.4;
    }

    .action-btn {
        background: #007bff;
        color: white;
        border: none;
        padding: 6px 12px;
        border-radius: 4px;
        cursor: pointer;
        font-size: 12px;
        margin-bottom: 8px;
    }

    .action-btn:hover {
        background: #0056b3;
    }

    .notification-timestamp {
        font-size: 11px;
        color: #6c757d;
        text-align: right;
    }

    .notifications-list {
        position: fixed;
        bottom: 20px;
        right: 20px;
        background: white;
        border-radius: 8px;
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
        max-width: 320px;
        max-height: 400px;
        overflow-y: auto;
        z-index: 9999;
    }

    .notifications-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 12px 16px;
        border-bottom: 1px solid #e9ecef;
        background: #f8f9fa;
    }

    .notifications-title {
        font-weight: 600;
        color: #343a40;
        font-size: 14px;
    }

    .clear-btn {
        background: none;
        border: none;
        color: #007bff;
        cursor: pointer;
        font-size: 12px;
    }

    .clear-btn:hover {
        text-decoration: underline;
    }

    .notification-item {
        display: flex;
        align-items: flex-start;
        padding: 12px 16px;
        border-bottom: 1px solid #f1f3f4;
    }

    .notification-item:last-child {
        border-bottom: none;
    }

    .item-icon {
        margin-right: 12px;
        font-size: 16px;
    }

    .item-content {
        flex: 1;
        min-width: 0;
    }

    .item-title {
        font-weight: 600;
        font-size: 13px;
        color: #343a40;
        margin-bottom: 2px;
    }

    .item-message {
        font-size: 12px;
        color: #495057;
        line-height: 1.3;
        margin-bottom: 4px;
        word-wrap: break-word;
    }

    .item-timestamp {
        font-size: 11px;
        color: #6c757d;
    }

    @keyframes slideIn {
        from {
            transform: translateX(100%);
            opacity: 0;
        }
        to {
            transform: translateX(0);
            opacity: 1;
        }
    }
</style> 