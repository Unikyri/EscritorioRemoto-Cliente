# Proyecto: Administración Remota de Equipos de Cómputo - Aplicación Cliente

## 1. Descripción General

Este repositorio contiene el código fuente y la documentación de la Aplicación Cliente de escritorio. Esta aplicación se ejecuta en el PC del Usuario Cliente, se encarga de registrar el PC con el servidor, gestionar las solicitudes de control remoto, capturar la pantalla, grabar video de la sesión, ejecutar comandos remotos recibidos del administrador y comunicarse con el Servidor Backend.

## 2. Tecnologías Utilizadas

* **Wails (v2):** Framework para construir aplicaciones de escritorio multiplataforma utilizando Go para el backend y tecnologías web (HTML, CSS, JavaScript/Svelte/Vue/React) para el frontend embebido.
* **Go (Golang):** Para toda la lógica de la aplicación cliente: 
    * Comunicación WebSocket (WSS) con el Servidor Backend.
    * Captura de pantalla y gestión de eventos de input (interacción con el S.O.).
    * Grabación de video de la sesión.
    * Gestión de la recepción de archivos.
    * Lógica de autenticación y registro del PC.
* **Tecnologías Web (HTML, CSS, JS/Svelte - para la UI Wails):** Para construir la interfaz gráfica de usuario que se ejecuta dentro de la ventana de Wails.
  
## 3. Requerimientos Específicos de la Aplicación Cliente

* Permitir al Usuario Cliente iniciar sesión con sus credenciales.
* Registrar el PC con el servidor (automáticamente o con un botón) después de iniciar sesión.
* Mostrar el estado de conexión con el servidor.
* Recibir y mostrar notificaciones cuando un administrador quiera iniciar una sesión de control remoto. 
* Permitir al Usuario Cliente aceptar o rechazar peticiones de control remoto. 
* Cuando se inicie una sesión de control remoto, comenzar a grabar la pantalla.
* Detener la grabación cuando finalice la sesión de control remoto. 
* Al finalizar la grabación, enviar el video desde el cliente al servidor.
* Almacenar el video temporalmente en un directorio local del cliente antes y durante la subida, y borrarlo tras la subida exitosa.
* Recibir archivos enviados por el Administrador y guardarlos en un directorio predefinido.
* Mostrar notificaciones básicas (ej. "El admin X quiere controlar su PC", "Archivo Y recibido"). 

## 4. Casos de Uso de la Aplicación Cliente

* **CU-C1: Autenticar Usuario Cliente**
Permite a un Usuario Cliente iniciar sesión en la aplicación cliente de escritorio.
 Actor Primario: Usuario Cliente
Precondiciones:
○ El Usuario Cliente tiene una cuenta válida creada por un Administrador. 
○ La aplicación cliente está instalada y en ejecución. 
○ El PC tiene conexión a la red para comunicarse con el servidor. 
● Flujo Principal: 
○ El Usuario Cliente abre la aplicación cliente. 
○ La aplicación cliente presenta una interfaz de inicio de sesión. 
○ El Usuario Cliente ingresa su nombre de usuario y contraseña. 
○ El Usuario Cliente envía la información. 
○ La aplicación cliente envía las credenciales al Sistema Servidor 
para verificación. 
○ El Sistema Servidor valida las credenciales. 
○ Si son válidas, el Sistema Servidor notifica a la aplicación cliente. 
○ La aplicación cliente establece una sesión local y registra el PC (Ver CU-C2). 
● Postcondiciones (Éxito): 
○ El Usuario Cliente ha iniciado sesión en la aplicación cliente. 
○ La aplicación cliente está conectada y autenticada con el servidor. 
● Flujos Alternativos/Excepciones: 
○ Credenciales Inválidas: El Sistema Servidor lo notifica, la aplicación cliente muestra un error. 
○ Servidor Inaccesible: La aplicación cliente muestra un error de conexión.

* **CU-C2: Registrar PC Cliente con el Servidor**
Permite que el PC del Usuario Cliente se registre con el servidor para ser visible y administrable.
Precondiciones: 
○ El Usuario Cliente ha iniciado sesión exitosamente en la aplicación cliente (CU-C1). 
● Flujo Principal: 
○ Inmediatamente después de un inicio de sesión exitoso (CU-C1), la aplicación cliente recopila información básica del PC (ej: un identificador único, nombre de host - MVP básico). 
○ La aplicación cliente envía esta información de registro al Sistema Servidor, asociándola con el Usuario Cliente autenticado. 
○ El Sistema Servidor almacena la información del PC en la base de datos y lo marca como "online". 
○ La aplicación cliente muestra un estado de "Conectado y 
Registrado". 
● Postcondiciones (Éxito): 
○ El PC Cliente está registrado en el servidor y asociado al Usuario Cliente. 
○ El PC Cliente aparece como "online" para los Administradores.

* **CU-C3: Gestionar Solicitud de Control Remoto Entrante**
Permite al Usuario Cliente aceptar o rechazar una solicitud de control remoto iniciada por un Administrador.
Precondiciones: 
○ El Usuario Cliente ha iniciado sesión y su PC está registrado y online (CU-C1, CU-C2). 
○ Un Administrador ha iniciado una solicitud de control remoto para este PC (parte de CU-A4). 
● Flujo Principal: 
○ La aplicación cliente recibe una notificación del Sistema Servidor indicando una solicitud de control remoto entrante (incluyendo el nombre del Administrador solicitante). 
○ La aplicación cliente muestra una notificación/diálogo prominente al Usuario Cliente con opciones para "Aceptar" o "Rechazar". 
○ El Usuario Cliente selecciona una opción. 
○ Si el Usuario Cliente selecciona "Aceptar": a. La aplicación cliente notifica al Sistema Servidor la aceptación. b. La aplicación cliente se prepara para transmitir el video de pantalla y recibir comandos de input. c. La aplicación cliente comienza la grabación de la sesión (Ver CU-C4). ○ Si el Usuario Cliente selecciona "Rechazar": a. La aplicación cliente notifica al Sistema Servidor el rechazo. b. No se establece 
a la sesión de control.
Postcondiciones (Éxito al Aceptar): 
○ Se ha notificado la aceptación al servidor. La sesión de control remoto se establece (continuación en CU-A4). 
● Postcondiciones (Éxito al Rechazar): 
○ Se ha notificado el rechazo al servidor. La sesión de control remoto no se establece. 
● Flujos Alternativos/Excepciones: 
○ Timeout (Usuario no responde): La aplicación cliente podría rechazar automáticamente la solicitud después de un tiempo

* **CU-C4: Grabar y Enviar Video de Sesión de Control Remoto**
La aplicación cliente graba la pantalla durante una sesión de control remoto y envía el video al servidor al finalizar.
Actor Primario: Sistema Cliente
Flujo Principal: 
○ Al iniciarse la sesión de control remoto (tras la aceptación del Usuario Cliente), la aplicación cliente comienza a grabar la pantalla en un archivo de video local (formato simple, ej: MP4/WebM a 720p 30fps si es posible). 
○ La grabación continúa mientras la sesión de control remoto esté activa. 
○ Al recibir la señal de finalización de sesión del Sistema Servidor (parte de CU-A4), la aplicación cliente detiene la grabación y finaliza el archivo de video. 
○ La aplicación cliente inicia la transferencia del archivo de video grabado al Sistema Servidor. 
○ Una vez que el Sistema Servidor confirma la recepción exitosa del video, la aplicación cliente puede eliminar el archivo de video local. 
● Postcondiciones (Éxito): 
○ El video de la sesión de control remoto ha sido grabado. 
○ El video ha sido enviado (o está en proceso de envío) al servidor. 
○ El video local puede ser eliminado tras confirmación de subida. 
● Flujos Alternativos/Excepciones: 
○ Error de Grabación: La aplicación cliente podría no poder grabar (ej: permisos, recursos). Debería notificarlo. MVP asume que puede grabar. 
○ Error de Envío de Video: La aplicación cliente reintenta el envío (MVP podría no tener reintentos sofisticados) o marca el video para envío manual/posterior (V2). 
○ Cierre Inesperado de la Aplicación Cliente: El video podría quedar parcialmente grabado o no enviado.

* **CU-C5: Recibir Archivo desde el Servidor**
La aplicación cliente recibe un archivo enviado por un Administrador desde el servidor.
Precondiciones: 
○ Una sesión de control remoto está activa. 
○ Un Administrador ha iniciado una transferencia de archivo hacia este PC Cliente (parte de CU-A5). 
● Flujo Principal: 
○ La aplicación cliente recibe una notificación del Sistema Servidor sobre una transferencia de archivo entrante (incluyendo metadatos como nombre de archivo, tamaño). 
○ La aplicación cliente se prepara para recibir los datos del archivo. 
○ La aplicación cliente recibe los datos del archivo y los guarda en un directorio local predefinido (ej: Descargas/RecibidosDelServidor). 
○ Una vez completada la recepción, la aplicación cliente notifica al Sistema Servidor el éxito. 
○ La aplicación cliente muestra una notificación al Usuario Cliente indicando que se ha recibido un archivo. 
● Postcondiciones (Éxito): 
○ El archivo ha sido recibido y guardado en el directorio predefinido. 
○ El Usuario Cliente ha sido notificado. 
● Flujos Alternativos/Excepciones: 
○ Error de Recepción: La transferencia falla. El Sistema Cliente lo notifica al Servidor. 
○ Espacio Insuficiente (MVP no lo verifica proactivamente): La escritura del archivo falla. 

* **CU-C6: Visualizar Estado de Conexión**
Permite al Usuario Cliente ver el estado de su conexión con el servidor.
Actor Primario: Usuario Cliente
Flujo Principal: 
○ La aplicación cliente mantiene una conexión (WebSocket) con el 
Sistema Servidor después del inicio de sesión. 
○ La interfaz de la aplicación cliente muestra un indicador visual (ej: ícono, texto) del estado actual de esta conexión (Conectado, Desconectado, Conectando).
● Postcondiciones (Éxito): 
○ El Usuario Cliente puede ver el estado de su conexión




## 5. Modelo de Componentes y Datos (Aplicación Cliente Wails)

La aplicación cliente Wails tiene dos partes principales: el frontend (UI) y el backend Go.

* **Frontend (UI - Tecnologías Web):**
    * Responsable de la presentación visual y la interacción directa con el Usuario Cliente. 
    * Componentes UI (ej. `LoginView.svelte`, `MainDashboardView.svelte`, `NotificationPopup.svelte`). 
    * Lógica de Vista y Manejadores de Eventos (JavaScript/TypeScript) que interactúan con el backend Go a través del "Wails Bridge".
    * Ver Diagrama de Clases - Modelo Frontend (UI/Views) Cliente. 
    ```plantuml
    @startuml Client Frontend (UI/Views) Class Diagram
    !theme materia-outline
    '--- Contenido del diagrama de clases del Frontend Cliente ---
    package "UI Views / Components" <<ViewLayer>> {
      class LoginView {
        + onLoginClick(): void
      }
      class MainDashboardView {
        + showRemoteControlRequest(adminName: string): void
      }
    }
    package "UI Logic / Controllers / State" <<LogicLayer>> {
      class AuthController {
        + login(username: string, password: string): Promise<boolean>
      }
    }
    package "Wails Bridge (JS API)" <<BridgeLayer>> {
      interface WailsAppBindings {
        + HandleLoginRequest(username: string, password: string): Promise<AuthResultDTO>
      }
    }
    ' ... (resto de las clases y relaciones)
    @enduml
    ```

* **Backend (Lógica en Go):**
    * `App Core / Event Bus`: Orquesta la lógica principal, expone funciones a la UI.
    * `API Client / Communicator`: Gestiona la comunicación WebSocket con el Servidor Backend.
    * `Session Manager`: Gestiona la sesión del usuario (token, estado). 
    * `PC Registrar`: Recopila info del PC y lo registra.
    * `Remote Control Agent`: Captura pantalla, procesa comandos, graba y sube video. 
    * `File Transfer Agent`: Gestiona la recepción de archivos.
    * `Notification Service`: Muestra notificaciones al usuario.
    * `Local Config/Storage`: Carga y guarda configuraciones locales. 
    * Ver Diagrama de Clases - Modelo Backend (Go Cliente).
    ```plantuml
    @startuml Client Go Backend Class Diagram
    !theme materia-outline
    '--- Contenido del diagrama de clases del Backend Go Cliente ---
    package "App Core (appcore)" {
      struct App {
        + HandleLoginRequest(username string, password string): AuthResultDTO
      }
    }
    package "API Client (api)" {
      struct Client {
        + Authenticate(correo string, password string): (string, error)
      }
    }
    ' ... (resto de los structs y paquetes Go)
    @enduml
    ```
* **Diagrama General de Paquetes Cliente:**
```
@startuml Client Application Component/Package Diagram 
!theme materia-outline 
skinparam defaultFontName Segoe UI 
skinparam rectangle { 
BackgroundColor LightBlue 
BorderColor DarkBlue 
ArrowColor DarkBlue 
} 
skinparam component { 
BackgroundColor LightCyan 
BorderColor DarkCyan 
ArrowColor DarkCyan 
} 
skinparam package { 
BackgroundColor LightYellow 
BorderColor Orange 
ArrowColor Orange 
} 
skinparam node { 
BackgroundColor LightGray 
BorderColor Gray 
} 
skinparam interface { 
BackgroundColor LightGreen 
BorderColor DarkGreen 
} 
skinparam agent { 
  BackgroundColor LightPink 
  BorderColor DarkRed 
} 
skinparam note { 
  BackgroundColor LightGoldenRodYellow 
  BorderColor Orange 
  FontColor #333333 
} 
 
package "Aplicación Cliente (Wails/Go)" <<DesktopApp>> { 
 
  package "Frontend (UI - Tecnologías Web)" <<WebView>> { 
    component "UI Components\n(HTML, CSS, JS/Svelte)" as WebUI <<View>> 
    component "View Logic & Event Handlers\n(JavaScript/TypeScript)" as ViewLogic 
<<Controller/ViewModel>> 
    interface "Wails Bridge (JS API)" as WailsJSBridge <<Bridge>> 
    note top of WebUI : Responsable de la presentación visual\ny la interacción directa del 
usuario. 
    note top of ViewLogic : Maneja eventos de la UI, prepara datos\npara la vista y llama a 
funciones Go\na través del Wails Bridge. 
  } 
 
  package "Backend (Lógica en Go)" <<GoBackend>> { 
    component "App Core / Event Bus (Go)" as GoAppCore 
    note right of GoAppCore : Orquesta la lógica principal de la aplicación cliente,\nexpone 
funciones a la UI y maneja eventos. 
 
    component "API Client / Communicator (Go)" as GoAPIClient 
    note bottom of GoAPIClient : Gestiona la comunicación WebSocket\ncon el Servidor 
Backend. 
 
    component "Session Manager (Go)" as GoSessionManager 
    note right of GoSessionManager : Gestiona la sesión del usuario (token, 
estado).\nOperaciones: StoreCredentials, GetAuthToken, ClearSession, IsUserLoggedIn. 
 
    component "PC Registrar (Go)" as GoPCRegistrar 
    note right of GoPCRegistrar : Recopila info del PC y lo registra.\nOperaciones: 
CollectPCInfo, AttemptRegistration. 
 
    component "Remote Control Agent (Go)" as GoRemoteAgent 
    note bottom of GoRemoteAgent : Captura pantalla, procesa comandos remotos,\ngraba y 
sube video.\nOperaciones: StartScreenCapture, StopScreenCapture, 
ProcessIncomingCommand, StartVideoRecording, StopVideoRecording, UploadVideo. 
 
    component "File Transfer Agent (Go)" as GoFileAgent 
    note left of GoFileAgent : Gestiona la recepción de archivos.\nOperaciones: 
ReceiveFileChunk, SaveFile. 
 
    component "Notification Service (Go)" as GoNotifier 
    note left of GoNotifier : Muestra notificaciones al usuario.\nOperaciones: 
ShowDesktopNotification, UpdateUINotificationArea. 
 
    component "Local Config/Storage (Go)" as GoLocalConfig 
    note left of GoLocalConfig : Carga y guarda configuraciones locales.\nOperaciones: 
LoadSettings, SaveSettings. 
  } 
 
  ' Interacciones Frontend 
  WebUI --> ViewLogic : UserInteractions 
  ViewLogic --> WailsJSBridge : CallsGoFunctions 
  WailsJSBridge ..> GoAppCore : BindsToDo 
 
  ' Interacciones Backend Go 
  GoAppCore --> GoAPIClient : Uses > 
  GoAppCore --> GoSessionManager : Manages > 
  GoAppCore --> GoPCRegistrar : Uses > 
  GoAppCore --> GoRemoteAgent : Controls > 
  GoAppCore --> GoFileAgent : Manages > 
  GoAppCore --> GoNotifier : Triggers > 
  GoAppCore --> GoLocalConfig : Uses > 
 
  GoPCRegistrar --> GoAPIClient : SendsPCInfo > 
  GoRemoteAgent --> GoAPIClient : SendsScreenData, SendsVideo > 
  GoRemoteAgent ..> GoNotifier : ShowsRecordingStatus > 
  GoFileAgent ..> GoNotifier : ShowsFileTransferStatus > 
} 
 
' Dependencias Externas 
agent "Usuario Cliente" as UserAgent 
UserAgent --> WebUI : InteractsWith 
 
node "Sistema Operativo del Cliente" as ClientOS 
GoRemoteAgent ..> ClientOS : AccessScreen, AccessInput 
GoFileAgent ..> ClientOS : AccessFileSystem 
GoNotifier ..> ClientOS : ShowDesktopNotifications 
 
node "Servidor Backend (Go)" as ServerBackend 
GoAPIClient ..> ServerBackend : Communicates (WebSocket WSS) 
note on link : Autenticación, Registro PC, Control Remoto,\nTransferencia Archivos, Subida 
Video. 
 
@enduml
```



## 6. Estructura de Carpetas (Proyecto Wails Cliente)

Basada en una estructura típica de Wails y el modelo de paquetes del cliente.
```
/ (raíz del proyecto Cliente)
|-- frontend/                   # Código del frontend (HTML, CSS, JS/Svelte)
|   |-- src/
|   |   |-- components/
|   |   |-- stores/
|   |   |-- main.js             # Punto de entrada JS del frontend
|   |   |-- App.svelte          # Componente Svelte raíz
|   |-- wailsjs/                # Código generado por Wails para el bridge JS-Go
|   |-- package.json
|   |-- ... (configuración de svelte/vite/rollup)
|-- app.go                      # Estructura Go principal de la app Wails, con métodos "bindeados"
|-- pkg/                        # Paquetes Go reutilizables si los hubiera (o usar internal/)
|   |-- api/                    # Cliente WebSocket para comunicarse con el servidor 
|   |-- session/                # Gestión de sesión de usuario local 
|   |-- registrar/              # Lógica de registro del PC 
|   |-- remotecontrol/          # Agente de control remoto (captura, grabación, input) 
|   |-- filetransfer/           # Agente de transferencia de archivos (recepción) 
|   |-- notifications/          # Servicio de notificaciones de escritorio 
|   |-- config/                 # Gestión de configuración local 
|-- main.go                     # Punto de entrada de la aplicación Go
|-- go.mod
|-- go.sum
|-- wails.json                  # Configuración del proyecto Wails
|-- build/                      # Directorio de salida de los binarios compilados
|-- README.md
```
