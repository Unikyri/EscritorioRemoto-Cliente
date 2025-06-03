package remotecontrol

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"EscritorioRemoto-Cliente/pkg/api"

	"github.com/go-vgo/robotgo"
)

// InputSimulator handles mouse and keyboard input simulation
type InputSimulator struct {
	// Configuration
	enableSafety bool // Enable safety checks to prevent dangerous commands
}

// NewInputSimulator creates a new InputSimulator instance
func NewInputSimulator() *InputSimulator {
	return &InputSimulator{
		enableSafety: true, // Enable safety by default
	}
}

// ProcessMouseCommand processes a mouse input command
func (is *InputSimulator) ProcessMouseCommand(command api.InputCommand) error {
	log.Printf("🖱️ Processing mouse command: action=%s", command.Action)

	// Parse payload as MouseEventPayload
	payload, err := is.parseMousePayload(command.Payload)
	if err != nil {
		return fmt.Errorf("invalid mouse payload: %w", err)
	}

	switch command.Action {
	case "move":
		return is.moveMouse(payload.X, payload.Y)
	case "click":
		return is.clickMouse(payload.X, payload.Y, payload.Button)
	case "scroll":
		return is.scrollMouse(payload.X, payload.Y, payload.Delta)
	default:
		return fmt.Errorf("unknown mouse action: %s", command.Action)
	}
}

// ProcessKeyboardCommand processes a keyboard input command
func (is *InputSimulator) ProcessKeyboardCommand(command api.InputCommand) error {
	log.Printf("⌨️ Processing keyboard command: action=%s", command.Action)

	// Parse payload as KeyboardEventPayload
	payload, err := is.parseKeyboardPayload(command.Payload)
	if err != nil {
		return fmt.Errorf("invalid keyboard payload: %w", err)
	}

	switch command.Action {
	case "keydown":
		return is.keyDown(payload.Key, payload.Modifiers)
	case "keyup":
		return is.keyUp(payload.Key, payload.Modifiers)
	case "type":
		return is.typeText(payload.Text)
	default:
		return fmt.Errorf("unknown keyboard action: %s", command.Action)
	}
}

// Mouse operations

func (is *InputSimulator) moveMouse(x, y int) error {
	if is.enableSafety && !is.isValidCoordinates(x, y) {
		return fmt.Errorf("invalid coordinates: (%d, %d)", x, y)
	}

	robotgo.MoveMouse(x, y)
	log.Printf("🖱️ Mouse moved to (%d, %d)", x, y)
	return nil
}

func (is *InputSimulator) clickMouse(x, y int, button string) error {
	if is.enableSafety && !is.isValidCoordinates(x, y) {
		return fmt.Errorf("invalid coordinates: (%d, %d)", x, y)
	}

	// Log screen dimensions for debugging
	width, height := robotgo.GetScreenSize()
	log.Printf("🖥️ Screen dimensions: %dx%d", width, height)
	log.Printf("🎯 Target coordinates: (%d, %d)", x, y)

	// Check current mouse position before moving
	currentX, currentY := robotgo.GetMousePos()
	log.Printf("🔍 Current mouse position: (%d, %d)", currentX, currentY)

	// Move to position first
	robotgo.MoveMouse(x, y)

	// Verify mouse moved to correct position
	newX, newY := robotgo.GetMousePos()
	log.Printf("✅ Mouse moved to: (%d, %d)", newX, newY)

	// Add small delay for system to register the movement
	robotgo.MilliSleep(100)

	// Convert button string to robotgo button
	robotgoButton := is.convertButtonToRobotgo(button)
	log.Printf("🔘 Using button: %s (robotgo: %s)", button, robotgoButton)

	// Use single click method to avoid crashes
	log.Printf("🖱️ Executing click...")
	robotgo.Click(robotgoButton, false) // false = single click

	// Add small delay after click
	robotgo.MilliSleep(100)

	log.Printf("🖱️ Mouse clicked at (%d, %d) with %s button - COMPLETED", x, y, button)

	// Optional: Try to get window information at click position for debugging
	if title := robotgo.GetTitle(); title != "" {
		log.Printf("🪟 Active window at click: '%s'", title)
	} else {
		log.Printf("⚠️ Could not get active window title - may indicate permission issues")
	}

	return nil
}

func (is *InputSimulator) scrollMouse(x, y int, delta int) error {
	if is.enableSafety && !is.isValidCoordinates(x, y) {
		return fmt.Errorf("invalid coordinates: (%d, %d)", x, y)
	}

	// Move to position first
	robotgo.MoveMouse(x, y)

	// Determine scroll direction
	direction := "up"
	scrollAmount := delta
	if delta < 0 {
		direction = "down"
		scrollAmount = -delta
	}

	// Perform scroll
	robotgo.Scroll(0, scrollAmount)

	log.Printf("🖱️ Mouse scrolled %s by %d at (%d, %d)", direction, scrollAmount, x, y)
	return nil
}

// Keyboard operations

func (is *InputSimulator) keyDown(key string, modifiers []string) error {
	if is.enableSafety && is.isDangerousKey(key) {
		return fmt.Errorf("dangerous key blocked: %s", key)
	}

	// Convert key to robotgo format
	robotgoKey := is.convertKeyToRobotgo(key)

	// Handle modifiers
	if len(modifiers) > 0 {
		// Build key combination
		keys := make([]string, 0, len(modifiers)+1)
		for _, mod := range modifiers {
			keys = append(keys, is.convertModifierToRobotgo(mod))
		}
		keys = append(keys, robotgoKey)

		robotgo.KeyTap(robotgoKey, keys[:len(keys)-1])
	} else {
		robotgo.KeyDown(robotgoKey)
	}

	log.Printf("⌨️ Key down: %s (modifiers: %v)", key, modifiers)
	return nil
}

func (is *InputSimulator) keyUp(key string, modifiers []string) error {
	// Convert key to robotgo format
	robotgoKey := is.convertKeyToRobotgo(key)

	robotgo.KeyUp(robotgoKey)

	log.Printf("⌨️ Key up: %s", key)
	return nil
}

func (is *InputSimulator) typeText(text string) error {
	if is.enableSafety && len(text) > 1000 {
		return fmt.Errorf("text too long: %d characters (max 1000)", len(text))
	}

	robotgo.TypeStr(text)

	log.Printf("⌨️ Typed text: %s", text)
	return nil
}

// Helper functions

func (is *InputSimulator) parseMousePayload(payload map[string]interface{}) (*api.MouseEventPayload, error) {
	// Convert map to JSON and back to struct for type safety
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	var mousePayload api.MouseEventPayload
	err = json.Unmarshal(jsonData, &mousePayload)
	if err != nil {
		return nil, err
	}

	return &mousePayload, nil
}

func (is *InputSimulator) parseKeyboardPayload(payload map[string]interface{}) (*api.KeyboardEventPayload, error) {
	// Convert map to JSON and back to struct for type safety
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	var keyboardPayload api.KeyboardEventPayload
	err = json.Unmarshal(jsonData, &keyboardPayload)
	if err != nil {
		return nil, err
	}

	return &keyboardPayload, nil
}

func (is *InputSimulator) isValidCoordinates(x, y int) bool {
	// Get screen dimensions
	width, height := robotgo.GetScreenSize()

	return x >= 0 && x < width && y >= 0 && y < height
}

func (is *InputSimulator) convertButtonToRobotgo(button string) string {
	switch strings.ToLower(button) {
	case "left", "":
		return "left"
	case "right":
		return "right"
	case "middle":
		return "center"
	default:
		return "left" // Default to left click
	}
}

func (is *InputSimulator) convertKeyToRobotgo(key string) string {
	// Map common key names to robotgo format
	keyMap := map[string]string{
		"Enter":      "enter",
		"Space":      "space",
		"Tab":        "tab",
		"Escape":     "esc",
		"Backspace":  "backspace",
		"Delete":     "delete",
		"ArrowUp":    "up",
		"ArrowDown":  "down",
		"ArrowLeft":  "left",
		"ArrowRight": "right",
		"Home":       "home",
		"End":        "end",
		"PageUp":     "pageup",
		"PageDown":   "pagedown",
		"F1":         "f1",
		"F2":         "f2",
		"F3":         "f3",
		"F4":         "f4",
		"F5":         "f5",
		"F6":         "f6",
		"F7":         "f7",
		"F8":         "f8",
		"F9":         "f9",
		"F10":        "f10",
		"F11":        "f11",
		"F12":        "f12",
	}

	if robotgoKey, exists := keyMap[key]; exists {
		return robotgoKey
	}

	// For single characters, return lowercase
	if len(key) == 1 {
		return strings.ToLower(key)
	}

	// For numeric keys
	if len(key) == 1 && key >= "0" && key <= "9" {
		return key
	}

	// Default to the key as-is
	return strings.ToLower(key)
}

func (is *InputSimulator) convertModifierToRobotgo(modifier string) string {
	switch strings.ToLower(modifier) {
	case "ctrl", "control":
		return "ctrl"
	case "alt":
		return "alt"
	case "shift":
		return "shift"
	case "meta", "cmd", "windows":
		return "cmd"
	default:
		return modifier
	}
}

func (is *InputSimulator) isDangerousKey(key string) bool {
	// List of potentially dangerous key combinations
	dangerousKeys := []string{
		"F4",     // Alt+F4 closes applications
		"Delete", // Ctrl+Alt+Delete
	}

	for _, dangerous := range dangerousKeys {
		if strings.EqualFold(key, dangerous) {
			return true
		}
	}

	return false
}

// SetSafety enables or disables safety checks
func (is *InputSimulator) SetSafety(enabled bool) {
	is.enableSafety = enabled
	log.Printf("🛡️ Input safety checks: %v", enabled)
}

// GetScreenInfo returns screen information
func (is *InputSimulator) GetScreenInfo() map[string]interface{} {
	width, height := robotgo.GetScreenSize()

	return map[string]interface{}{
		"width":  width,
		"height": height,
	}
}

// TestInput performs basic input tests
func (is *InputSimulator) TestInput() error {
	log.Printf("🧪 Testing input simulation...")

	// Test mouse movement (move to center of screen)
	width, height := robotgo.GetScreenSize()
	centerX, centerY := width/2, height/2

	log.Printf("🖥️ Screen size: %dx%d", width, height)
	log.Printf("🎯 Testing mouse movement to center: (%d, %d)", centerX, centerY)

	err := is.moveMouse(centerX, centerY)
	if err != nil {
		return fmt.Errorf("mouse movement test failed: %w", err)
	}

	// Verify mouse position
	actualX, actualY := robotgo.GetMousePos()
	log.Printf("✅ Mouse position after test move: (%d, %d)", actualX, actualY)

	// Test click functionality
	log.Printf("🖱️ Testing mouse click at center...")
	err = is.clickMouse(centerX, centerY, "left")
	if err != nil {
		return fmt.Errorf("mouse click test failed: %w", err)
	}

	// Test if we can detect if click was successful by checking if mouse is still at position
	afterClickX, afterClickY := robotgo.GetMousePos()
	log.Printf("🔍 Mouse position after click: (%d, %d)", afterClickX, afterClickY)

	// Check if robotgo has admin privileges (Windows specific test)
	log.Printf("🛡️ Testing system permissions...")

	// Try to get active window title as a privilege test
	if title := robotgo.GetTitle(); title != "" {
		log.Printf("✅ Can read active window title: '%s'", title)
	} else {
		log.Printf("⚠️ Cannot read active window title - may indicate permission issues")
	}

	// Test typing capability
	log.Printf("⌨️ Testing keyboard input...")
	testText := "test"
	err = is.typeText(testText)
	if err != nil {
		log.Printf("⚠️ Keyboard test failed: %v", err)
	} else {
		log.Printf("✅ Keyboard test successful")
	}

	log.Printf("✅ Input simulation test completed (screen: %dx%d)", width, height)
	log.Printf("💡 If clicks are not working, try running as Administrator on Windows")

	return nil
}
