# Cliente Desktop - Escritorio Remoto

Cliente desktop desarrollado con Wails v2 (Go + Svelte) para el sistema de administración remota de equipos.

## Funcionalidades Implementadas

### FASE 2 ✅
- Autenticación de usuario cliente
- Registro automático del PC en el servidor
- Conexión WebSocket persistente
- Heartbeat automático
- Gestión de sesión local
- Dashboard con información del sistema

## Tecnologías

- **Backend**: Go con Wails v2
- **Frontend**: Svelte + Vite
- **Comunicación**: WebSocket para tiempo real
- **UI**: Componentes Svelte modernos

## Desarrollo

```bash
# Instalar dependencias
go mod download

# Ejecutar en modo desarrollo
wails dev

# Compilar para producción
wails build
```

## Estructura del Proyecto

```
pkg/
├── api/        # Cliente WebSocket y DTOs
├── session/    # Gestión de sesión local
└── utils/      # Utilidades del sistema

frontend/
├── src/
│   ├── components/  # Componentes Svelte
│   ├── stores/      # Estados globales
│   └── App.svelte   # Componente principal
```

## Estado del Proyecto

- **FASE 1**: ✅ Completada (Backend + WebAdmin)
- **FASE 2**: ✅ Completada (Cliente Desktop)
- **FASE 3**: 🔄 En progreso (Visualización PCs)

## Requisitos

- Go 1.21+
- Node.js 16+
- Wails v2.10.1+
