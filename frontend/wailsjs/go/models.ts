export namespace main {
	
	export class VideoRecordingState {
	    is_recording: boolean;
	    session_id: string;
	    video_id: string;
	    start_time: string;
	    duration: number;
	    frame_count: number;
	    is_uploading: boolean;
	    upload_progress: number;
	
	    static createFrom(source: any = {}) {
	        return new VideoRecordingState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.is_recording = source["is_recording"];
	        this.session_id = source["session_id"];
	        this.video_id = source["video_id"];
	        this.start_time = source["start_time"];
	        this.duration = source["duration"];
	        this.frame_count = source["frame_count"];
	        this.is_uploading = source["is_uploading"];
	        this.upload_progress = source["upload_progress"];
	    }
	}

}

