export namespace main {
	
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

