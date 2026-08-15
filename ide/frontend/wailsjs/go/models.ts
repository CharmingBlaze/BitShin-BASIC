export namespace main {
	
	export class FileNode {
	    name: string;
	    path: string;
	    isDir: boolean;
	    children?: FileNode[];
	    size?: number;
	
	    static createFrom(source: any = {}) {
	        return new FileNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.isDir = source["isDir"];
	        this.children = this.convertValues(source["children"], FileNode);
	        this.size = source["size"];
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
	export class FileResult {
	    path: string;
	    name: string;
	    content: string;
	    isBinary: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FileResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.content = source["content"];
	        this.isBinary = source["isBinary"];
	    }
	}
	export class UserSettings {
	    theme: string;
	    fontSize: number;
	    tabSize: number;
	    minimap: boolean;
	    wordWrap: string;
	    autoSave: boolean;
	    targetOS: string;
	    outputHeight: number;
	    sidebarWidth: number;
	
	    static createFrom(source: any = {}) {
	        return new UserSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.theme = source["theme"];
	        this.fontSize = source["fontSize"];
	        this.tabSize = source["tabSize"];
	        this.minimap = source["minimap"];
	        this.wordWrap = source["wordWrap"];
	        this.autoSave = source["autoSave"];
	        this.targetOS = source["targetOS"];
	        this.outputHeight = source["outputHeight"];
	        this.sidebarWidth = source["sidebarWidth"];
	    }
	}

}

