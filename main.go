package main

import (
	"embed"
	"flag"
	"fmt"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Definir parámetros de línea de comandos
	var (
		serverURL = flag.String("server-url", "http://localhost:8080", "URL del servidor (ej: http://192.168.1.100:8080)")
		showHelp  = flag.Bool("help", false, "Mostrar ayuda")
		username  = flag.String("username", "", "Usuario para autenticación automática")
		password  = flag.String("password", "", "Contraseña para autenticación automática")
		pcName    = flag.String("pc-name", "", "Nombre del PC para registro automático")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "EscritorioRemoto-Cliente - Cliente de Escritorio Remoto\n\n")
		fmt.Fprintf(os.Stderr, "Uso: %s [opciones]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Opciones:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEjemplos:\n")
		fmt.Fprintf(os.Stderr, "  %s --server-url http://192.168.1.100:8080\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --server-url http://10.0.0.5:8080 --username usuario --password pass\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nSi no se especifica server-url, se usa localhost:8080 por defecto.\n")
	}

	flag.Parse()

	if *showHelp {
		flag.Usage()
		return
	}

	// Mostrar configuración
	fmt.Printf("🌐 Servidor configurado: %s\n", *serverURL)
	if *username != "" {
		fmt.Printf("👤 Usuario: %s\n", *username)
	}
	if *pcName != "" {
		fmt.Printf("💻 Nombre PC: %s\n", *pcName)
	}

	// Create an instance of the app structure using MVC architecture
	// Pasar la configuración desde línea de comandos
	app := NewAppWithConfig(*serverURL, *username, *password, *pcName)

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "EscritorioRemoto-Cliente",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
