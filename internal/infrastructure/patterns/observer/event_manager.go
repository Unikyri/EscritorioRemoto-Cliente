package observer

import (
	"fmt"
	"sync"
)

// Event representa un evento genérico
type Event struct {
	Type string
	Data interface{}
}

// Observer interface para observadores de eventos
type Observer interface {
	OnEvent(event Event) error
	GetID() string
}

// EventManager implementa el patrón Observer (Singleton)
type EventManager struct {
	observers map[string][]Observer
	mutex     sync.RWMutex
}

// Singleton instance
var (
	instance *EventManager
	once     sync.Once
)

// GetInstance retorna la instancia singleton del EventManager
func GetInstance() *EventManager {
	once.Do(func() {
		instance = &EventManager{
			observers: make(map[string][]Observer),
			mutex:     sync.RWMutex{},
		}
	})
	return instance
}

// Subscribe registra un observer para un tipo de evento específico
func (em *EventManager) Subscribe(eventType string, observer Observer) error {
	if observer == nil {
		return fmt.Errorf("observer cannot be nil")
	}
	if eventType == "" {
		return fmt.Errorf("event type cannot be empty")
	}

	em.mutex.Lock()
	defer em.mutex.Unlock()

	// Verificar si el observer ya está registrado
	for _, obs := range em.observers[eventType] {
		if obs.GetID() == observer.GetID() {
			return nil // Ya está registrado
		}
	}

	em.observers[eventType] = append(em.observers[eventType], observer)
	return nil
}

// Unsubscribe desregistra un observer de un tipo de evento
func (em *EventManager) Unsubscribe(eventType string, observer Observer) error {
	if observer == nil {
		return fmt.Errorf("observer cannot be nil")
	}

	em.mutex.Lock()
	defer em.mutex.Unlock()

	observers, exists := em.observers[eventType]
	if !exists {
		return nil // No hay observers para este tipo
	}

	// Filtrar el observer a remover
	newObservers := make([]Observer, 0)
	for _, obs := range observers {
		if obs.GetID() != observer.GetID() {
			newObservers = append(newObservers, obs)
		}
	}

	em.observers[eventType] = newObservers
	return nil
}

// Publish publica un evento a todos los observers registrados
func (em *EventManager) Publish(event Event) error {
	em.mutex.RLock()
	observers, exists := em.observers[event.Type]
	em.mutex.RUnlock()

	if !exists || len(observers) == 0 {
		return nil // No hay observers para este evento
	}

	// Notificar a todos los observers de forma asíncrona
	for _, observer := range observers {
		go func(obs Observer) {
			if err := obs.OnEvent(event); err != nil {
				// En una implementación real, aquí iría logging
				fmt.Printf("Error notifying observer %s: %v\n", obs.GetID(), err)
			}
		}(observer)
	}

	return nil
}

// GetObserverCount retorna el número de observers para un tipo de evento
func (em *EventManager) GetObserverCount(eventType string) int {
	em.mutex.RLock()
	defer em.mutex.RUnlock()

	return len(em.observers[eventType])
}

// Clear limpia todos los observers
func (em *EventManager) Clear() {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	em.observers = make(map[string][]Observer)
}

// GetEventTypes retorna todos los tipos de eventos registrados
func (em *EventManager) GetEventTypes() []string {
	em.mutex.RLock()
	defer em.mutex.RUnlock()

	types := make([]string, 0, len(em.observers))
	for eventType := range em.observers {
		types = append(types, eventType)
	}
	return types
}
