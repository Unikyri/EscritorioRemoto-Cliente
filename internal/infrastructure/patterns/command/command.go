package command

import (
	"fmt"
	"time"
)

// Command interface para el patrón Command
type Command interface {
	Execute() error
	Undo() error
	GetDescription() string
	GetTimestamp() time.Time
	CanUndo() bool
}

// CommandResult representa el resultado de ejecutar un comando
type CommandResult struct {
	Success   bool
	Error     error
	Data      interface{}
	Timestamp time.Time
}

// CommandHistory mantiene el historial de comandos ejecutados
type CommandHistory struct {
	commands []Command
	maxSize  int
}

// NewCommandHistory crea un nuevo historial de comandos
func NewCommandHistory(maxSize int) *CommandHistory {
	if maxSize <= 0 {
		maxSize = 100 // Tamaño por defecto
	}

	return &CommandHistory{
		commands: make([]Command, 0, maxSize),
		maxSize:  maxSize,
	}
}

// ExecuteCommand ejecuta un comando y lo agrega al historial
func (ch *CommandHistory) ExecuteCommand(cmd Command) CommandResult {
	timestamp := time.Now().UTC()

	err := cmd.Execute()

	result := CommandResult{
		Success:   err == nil,
		Error:     err,
		Timestamp: timestamp,
	}

	// Agregar al historial solo si se ejecutó correctamente
	if result.Success {
		ch.addToHistory(cmd)
	}

	return result
}

// UndoLastCommand deshace el último comando si es posible
func (ch *CommandHistory) UndoLastCommand() error {
	if len(ch.commands) == 0 {
		return fmt.Errorf("no commands to undo")
	}

	lastCommand := ch.commands[len(ch.commands)-1]

	if !lastCommand.CanUndo() {
		return fmt.Errorf("command cannot be undone: %s", lastCommand.GetDescription())
	}

	err := lastCommand.Undo()
	if err != nil {
		return fmt.Errorf("failed to undo command: %w", err)
	}

	// Remover del historial
	ch.commands = ch.commands[:len(ch.commands)-1]

	return nil
}

// GetHistory retorna el historial de comandos
func (ch *CommandHistory) GetHistory() []Command {
	// Retornar una copia para evitar modificaciones externas
	historyCopy := make([]Command, len(ch.commands))
	copy(historyCopy, ch.commands)
	return historyCopy
}

// GetLastCommand retorna el último comando ejecutado
func (ch *CommandHistory) GetLastCommand() Command {
	if len(ch.commands) == 0 {
		return nil
	}
	return ch.commands[len(ch.commands)-1]
}

// Clear limpia el historial
func (ch *CommandHistory) Clear() {
	ch.commands = ch.commands[:0]
}

// GetSize retorna el tamaño actual del historial
func (ch *CommandHistory) GetSize() int {
	return len(ch.commands)
}

// addToHistory agrega un comando al historial
func (ch *CommandHistory) addToHistory(cmd Command) {
	// Si el historial está lleno, remover el más antiguo
	if len(ch.commands) >= ch.maxSize {
		ch.commands = ch.commands[1:]
	}

	ch.commands = append(ch.commands, cmd)
}

// BaseCommand proporciona implementación base para comandos
type BaseCommand struct {
	description string
	timestamp   time.Time
	canUndo     bool
}

// NewBaseCommand crea un comando base
func NewBaseCommand(description string, canUndo bool) BaseCommand {
	return BaseCommand{
		description: description,
		timestamp:   time.Now().UTC(),
		canUndo:     canUndo,
	}
}

// GetDescription implementa Command.GetDescription
func (bc *BaseCommand) GetDescription() string {
	return bc.description
}

// GetTimestamp implementa Command.GetTimestamp
func (bc *BaseCommand) GetTimestamp() time.Time {
	return bc.timestamp
}

// CanUndo implementa Command.CanUndo
func (bc *BaseCommand) CanUndo() bool {
	return bc.canUndo
}
