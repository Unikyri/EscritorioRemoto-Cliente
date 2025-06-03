<script>
	import { onMount, onDestroy } from 'svelte';
	import { GetVideoRecordingStatus } from '../../wailsjs/go/main/App.js';
	import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime.js';

	// Estado de la grabación
	let isRecording = false;
	let videoId = '';
	let sessionId = '';
	let available = false;
	let isUploading = false;
	let uploadProgress = 0;
	let uploadMessage = '';
	let lastEvent = '';

	// Interval para actualizar estado
	let statusInterval;

	onMount(() => {
		updateStatus();
		statusInterval = setInterval(updateStatus, 1000);

		// Escuchar eventos de video
		EventsOn('video_recording_started', handleVideoRecordingStarted);
		EventsOn('video_recording_completed', handleVideoRecordingCompleted);
		EventsOn('video_upload_started', handleVideoUploadStarted);
		EventsOn('video_upload_progress', handleVideoUploadProgress);
		EventsOn('video_upload_completed', handleVideoUploadCompleted);
		EventsOn('video_upload_failed', handleVideoUploadFailed);
	});

	onDestroy(() => {
		if (statusInterval) {
			clearInterval(statusInterval);
		}
		
		EventsOff('video_recording_started');
		EventsOff('video_recording_completed');
		EventsOff('video_upload_started');
		EventsOff('video_upload_progress');
		EventsOff('video_upload_completed');
		EventsOff('video_upload_failed');
	});

	async function updateStatus() {
		try {
			const status = await GetVideoRecordingStatus();
			available = status.available;
			isRecording = status.isRecording;
			videoId = status.videoId || '';
			sessionId = status.sessionId || '';
			isUploading = status.isUploading;
			uploadProgress = status.uploadProgress || 0;
		} catch (error) {
			console.error('Error obteniendo estado de grabación:', error);
		}
	}

	function handleVideoRecordingStarted(data) {
		console.log('🎬 Grabación iniciada:', data);
		isRecording = true;
		videoId = data.videoId;
		sessionId = data.sessionId;
		lastEvent = `🎬 Grabación iniciada`;
	}

	function handleVideoRecordingCompleted(data) {
		console.log('🎬 Grabación completada:', data);
		isRecording = false;
		lastEvent = `✅ Grabación completada - ${data.duration}s`;
	}

	function handleVideoUploadStarted(data) {
		console.log('📤 Subida iniciada:', data);
		isUploading = true;
		uploadProgress = 0;
		uploadMessage = 'Iniciando subida...';
		lastEvent = `📤 Subiendo video...`;
	}

	function handleVideoUploadProgress(data) {
		uploadMessage = data.message || 'Subiendo...';
	}

	function handleVideoUploadCompleted(data) {
		console.log('✅ Subida completada:', data);
		isUploading = false;
		uploadProgress = 100;
		uploadMessage = 'Subida completada';
		lastEvent = `✅ Video subido exitosamente`;
		
		setTimeout(() => {
			uploadMessage = '';
		}, 3000);
	}

	function handleVideoUploadFailed(data) {
		console.error('❌ Error en subida:', data);
		isUploading = false;
		uploadProgress = 0;
		uploadMessage = `Error: ${data.error}`;
		lastEvent = `❌ Error: ${data.error}`;
	}
</script>

{#if available}
<div class="video-indicator" class:recording={isRecording} class:uploading={isUploading}>
	<div class="indicator-header">
		<span class="icon">
			{#if isRecording}
				🔴
			{:else if isUploading}
				📤
			{:else}
				🎬
			{/if}
		</span>
		<span class="status-text">
			{#if isRecording}
				GRABANDO
			{:else if isUploading}
				SUBIENDO
			{:else}
				VIDEO LISTO
			{/if}
		</span>
	</div>

	{#if isUploading}
		<div class="progress-container">
			<div class="progress-bar">
				<div class="progress-fill" style="width: {uploadProgress}%"></div>
			</div>
			<div class="progress-text">{uploadMessage}</div>
		</div>
	{/if}

	{#if lastEvent}
		<div class="last-event">
			<small>{lastEvent}</small>
		</div>
	{/if}
</div>
{/if}

<style>
	.video-indicator {
		position: fixed;
		top: 20px;
		right: 20px;
		background: rgba(0, 0, 0, 0.8);
		color: white;
		padding: 12px 16px;
		border-radius: 8px;
		font-size: 14px;
		z-index: 1000;
		min-width: 200px;
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
		backdrop-filter: blur(10px);
		transition: all 0.3s ease;
	}

	.video-indicator.recording {
		background: rgba(220, 38, 38, 0.9);
		animation: pulse 2s infinite;
	}

	.video-indicator.uploading {
		background: rgba(59, 130, 246, 0.9);
	}

	.indicator-header {
		display: flex;
		align-items: center;
		gap: 8px;
		font-weight: bold;
	}

	.icon {
		font-size: 16px;
	}

	.status-text {
		font-size: 12px;
		letter-spacing: 0.5px;
	}

	.progress-container {
		margin-top: 8px;
	}

	.progress-bar {
		width: 100%;
		height: 4px;
		background: rgba(255, 255, 255, 0.2);
		border-radius: 2px;
		overflow: hidden;
	}

	.progress-fill {
		height: 100%;
		background: linear-gradient(90deg, #10b981, #34d399);
		transition: width 0.3s ease;
	}

	.progress-text {
		font-size: 10px;
		margin-top: 4px;
		opacity: 0.8;
	}

	.last-event {
		margin-top: 6px;
		font-size: 10px;
		opacity: 0.7;
		border-top: 1px solid rgba(255, 255, 255, 0.1);
		padding-top: 4px;
	}

	@keyframes pulse {
		0%, 100% { opacity: 1; }
		50% { opacity: 0.7; }
	}
</style> 