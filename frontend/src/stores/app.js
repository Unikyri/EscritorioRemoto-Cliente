import { writable } from 'svelte/store';

// Estado de autenticación
export const isAuthenticated = writable(false);
export const isRegistered = writable(false);
export const isConnected = writable(false);

// Información del usuario
export const userInfo = writable({
    username: '',
    userId: '',
    token: ''
});

// Información del PC
export const pcInfo = writable({
    pcId: '',
    identifier: '',
    systemInfo: {}
});

// Estado de la aplicación
export const appState = writable({
    loading: false,
    error: null,
    currentView: 'login' // 'login', 'dashboard'
});

// Funciones para actualizar el estado
export function setAuthenticated(authenticated, userData = {}) {
    isAuthenticated.set(authenticated);
    if (authenticated) {
        userInfo.set(userData);
        appState.update(state => ({ ...state, currentView: 'dashboard' }));
    } else {
        userInfo.set({ username: '', userId: '', token: '' });
        appState.update(state => ({ ...state, currentView: 'login' }));
    }
}

export function setRegistered(registered, pcData = {}) {
    isRegistered.set(registered);
    if (registered) {
        pcInfo.update(info => ({ ...info, ...pcData }));
    }
}

export function setConnected(connected) {
    isConnected.set(connected);
}

export function setLoading(loading) {
    appState.update(state => ({ ...state, loading }));
}

export function setError(error) {
    appState.update(state => ({ ...state, error }));
}

export function clearError() {
    appState.update(state => ({ ...state, error: null }));
} 