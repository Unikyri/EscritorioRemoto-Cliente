# 🌐 **CONFIGURACIÓN DE RED - Cliente Configurable**

## 📋 **Resumen**
El cliente ahora soporta **conexión a servidores remotos** en la misma red local o en internet, no solo `localhost`.

## 🔧 **Nuevas Funcionalidades**

### **Parámetros de Línea de Comandos:**

```bash
cliente-configurable.exe [opciones]

Opciones:
  --server-url string    URL del servidor (ej: http://192.168.1.100:8080)
  --username string      Usuario para autenticación automática  
  --password string      Contraseña para autenticación automática
  --pc-name string       Nombre del PC para registro automático
  --help                 Mostrar ayuda
```

## 🌍 **Escenarios de Uso**

### **1. Servidor en la Misma PC (Original):**
```bash
# Estas dos formas son equivalentes:
.\cliente-configurable.exe
.\cliente-configurable.exe --server-url http://localhost:8080
```

### **2. Servidor en Otra PC de la Red Local:**

#### **📍 Paso 1: Obtener IP del Servidor**
En la PC donde está el servidor, ejecutar:

**Windows:**
```cmd
ipconfig | findstr IPv4
```

**Linux/Mac:**
```bash
ip addr show | grep inet
```

**Ejemplo de salida:**
```
IPv4 Address: 192.168.1.100
```

#### **📍 Paso 2: Verificar que el Servidor esté Ejecutándose**
En la PC del servidor:
```bash
cd EscritorioRemoto-Backend
.\escritorio-remoto-backend.exe
```

**Debe mostrar:**
```
Servidor iniciando en puerto 8080
```

#### **📍 Paso 3: Conectar Cliente desde Otra PC**
En la PC del cliente:
```bash
.\cliente-configurable.exe --server-url http://192.168.1.100:8080
```

### **3. Servidor en Internet (IP Pública):**
```bash
.\cliente-configurable.exe --server-url http://203.0.113.25:8080
```

### **4. Autenticación Automática:**
```bash
.\cliente-configurable.exe \
  --server-url http://192.168.1.100:8080 \
  --username mi-usuario \
  --password mi-contraseña \
  --pc-name "PC-Oficina-Norte"
```

## 🚨 **Solución de Problemas**

### **Error: "Connection Failed"**

#### **Posibles Causas y Soluciones:**

1. **🔥 Firewall Bloqueando**
   ```bash
   # Verificar si el puerto 8080 está abierto
   telnet 192.168.1.100 8080
   ```
   
   **Solución:** Abrir puerto 8080 en Windows Firewall:
   - Panel de Control → Sistema y Seguridad → Firewall de Windows
   - Permitir aplicación → Agregar puerto 8080 TCP

2. **❌ Servidor No Ejecutándose**
   ```bash
   # Verificar estado del servidor
   curl http://192.168.1.100:8080/health
   ```
   
   **Debe responder:** `{"status": "ok"}`

3. **🌐 IP Incorrecta**
   ```bash
   # Verificar conectividad de red
   ping 192.168.1.100
   ```

4. **🔌 Puerto Ocupado**
   ```bash
   # En la PC del servidor, verificar que 8080 esté libre
   netstat -an | findstr 8080
   ```

### **Error: "Authentication Failed"**

- Verificar que las credenciales sean correctas
- Asegurar que el usuario existe en la base de datos del servidor

## 📊 **Ejemplos Completos**

### **Red Doméstica Típica:**
```bash
# Servidor en PC principal (192.168.1.100)
# Cliente en laptop (192.168.1.105)

# En la laptop:
.\cliente-configurable.exe \
  --server-url http://192.168.1.100:8080 \
  --username juan \
  --password mipassword \
  --pc-name "Laptop-Juan"
```

### **Oficina con Múltiples Clientes:**
```bash
# Servidor central: 10.0.0.10
# Cliente 1 (Recepción):
.\cliente-configurable.exe \
  --server-url http://10.0.0.10:8080 \
  --username recepcion \
  --password rec2025 \
  --pc-name "PC-Recepcion"

# Cliente 2 (Contabilidad):  
.\cliente-configurable.exe \
  --server-url http://10.0.0.10:8080 \
  --username contador \
  --password cont2025 \
  --pc-name "PC-Contabilidad"
```

## 🔒 **Consideraciones de Seguridad**

### **Red Local:**
- ✅ Seguro para redes privadas confiables
- ⚠️ Asegurar que la red WiFi tenga contraseña

### **Internet:**
- 🛡️ Considerar usar HTTPS (puerto 443)
- 🔐 Usar contraseñas fuertes
- 🌐 Configurar puerto forwarding en router si es necesario

## 📈 **Verificación de Conectividad**

### **Script de Prueba Rápida:**
```bash
# Prueba conexión básica
curl -s http://192.168.1.100:8080/health && echo "✅ Servidor Accesible" || echo "❌ Servidor No Accesible"

# Prueba login
curl -X POST http://192.168.1.100:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"testuser123"}'
```

## 🎯 **Configuración Recomendada**

### **Para Desarrollo/Testing:**
```bash
.\cliente-configurable.exe --server-url http://localhost:8080
```

### **Para Producción Local:**
```bash
.\cliente-configurable.exe --server-url http://[IP-SERVIDOR]:8080
```

### **Para Deploy Remoto:**
```bash
.\cliente-configurable.exe --server-url https://[DOMINIO]:443
```

---

## ✅ **Conclusión**

El cliente ahora es **completamente configurable** y puede conectarse a servidores en:
- ✅ Misma PC (`localhost`)
- ✅ Red local (`192.168.x.x`, `10.x.x.x`)  
- ✅ Internet (IP pública o dominio)

**FASE 8+ COMPLETADA:** Cliente configurable para entornos multi-PC ✅ 