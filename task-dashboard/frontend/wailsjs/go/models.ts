export namespace main {
	
	export class AgentWorktree {
	    name: string;
	    status: string;
	    taskId?: string;
	    taskTitle?: string;
	    pid?: string;
	    started?: string;
	
	    static createFrom(source: any = {}) {
	        return new AgentWorktree(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.status = source["status"];
	        this.taskId = source["taskId"];
	        this.taskTitle = source["taskTitle"];
	        this.pid = source["pid"];
	        this.started = source["started"];
	    }
	}
	export class AgentStatusInfo {
	    worktrees: AgentWorktree[];
	    totalWorktrees: number;
	    idleCount: number;
	    busyCount: number;
	    maxSubagents: number;
	
	    static createFrom(source: any = {}) {
	        return new AgentStatusInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.worktrees = this.convertValues(source["worktrees"], AgentWorktree);
	        this.totalWorktrees = source["totalWorktrees"];
	        this.idleCount = source["idleCount"];
	        this.busyCount = source["busyCount"];
	        this.maxSubagents = source["maxSubagents"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class Task {
	    id: number;
	    title: string;
	    status: string;
	    priority: string;
	    deps: number[];
	    parent?: number;
	
	    static createFrom(source: any = {}) {
	        return new Task(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.status = source["status"];
	        this.priority = source["priority"];
	        this.deps = source["deps"];
	        this.parent = source["parent"];
	    }
	}

}

