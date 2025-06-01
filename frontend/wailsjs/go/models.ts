export namespace api {
	
	export class AuthResultDTO {
	    success: boolean;
	    token?: string;
	    userId?: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new AuthResultDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.token = source["token"];
	        this.userId = source["userId"];
	        this.error = source["error"];
	    }
	}
	export class PCRegistrationResultDTO {
	    success: boolean;
	    pcId?: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new PCRegistrationResultDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.pcId = source["pcId"];
	        this.error = source["error"];
	    }
	}

}

export namespace main {
	
	export class ConnectionStatus {
	    isConnected: boolean;
	    status: string;
	    lastHeartbeat: number;
	    serverUrl: string;
	    connectionTime: number;
	    errorMessage?: string;
	
	    static createFrom(source: any = {}) {
	        return new ConnectionStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isConnected = source["isConnected"];
	        this.status = source["status"];
	        this.lastHeartbeat = source["lastHeartbeat"];
	        this.serverUrl = source["serverUrl"];
	        this.connectionTime = source["connectionTime"];
	        this.errorMessage = source["errorMessage"];
	    }
	}

}

export namespace session {
	
	export class SessionData {
	    token: string;
	    userId: string;
	    username: string;
	    pcId?: string;
	
	    static createFrom(source: any = {}) {
	        return new SessionData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.token = source["token"];
	        this.userId = source["userId"];
	        this.username = source["username"];
	        this.pcId = source["pcId"];
	    }
	}

}

