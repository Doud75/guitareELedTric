export namespace main {
	
	export class UniverseDetail {
	    universe: number;
	    ranges: string[];
	
	    static createFrom(source: any = {}) {
	        return new UniverseDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.universe = source["universe"];
	        this.ranges = source["ranges"];
	    }
	}

}

