export namespace db {
	
	export class AccessLog {
	    id: number;
	    cardId: string;
	    userName: string;
	    roleName: string;
	    roleCode: number;
	    studentNo: string;
	    eventType: string;
	    // Go type: time
	    timestamp: any;
	    durationSecond: number;
	    durationText: string;
	
	    static createFrom(source: any = {}) {
	        return new AccessLog(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.cardId = source["cardId"];
	        this.userName = source["userName"];
	        this.roleName = source["roleName"];
	        this.roleCode = source["roleCode"];
	        this.studentNo = source["studentNo"];
	        this.eventType = source["eventType"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
	        this.durationSecond = source["durationSecond"];
	        this.durationText = source["durationText"];
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
	export class DashboardStats {
	    currentInsideCount: number;
	    totalUserCount: number;
	    todayLogCount: number;
	
	    static createFrom(source: any = {}) {
	        return new DashboardStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currentInsideCount = source["currentInsideCount"];
	        this.totalUserCount = source["totalUserCount"];
	        this.todayLogCount = source["todayLogCount"];
	    }
	}
	export class MonthlyStatRow {
	    yearMonth: string;
	    roleOtherCount: number;
	    role0Count: number;
	    role1Count: number;
	    role9Count: number;
	    monthlyTotal: number;
	    fiscalYear: number;
	    fiscalYearCumulativeTotal: number;
	    quarterPeriod: string;
	    quarterTotal: number;
	
	    static createFrom(source: any = {}) {
	        return new MonthlyStatRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.yearMonth = source["yearMonth"];
	        this.roleOtherCount = source["roleOtherCount"];
	        this.role0Count = source["role0Count"];
	        this.role1Count = source["role1Count"];
	        this.role9Count = source["role9Count"];
	        this.monthlyTotal = source["monthlyTotal"];
	        this.fiscalYear = source["fiscalYear"];
	        this.fiscalYearCumulativeTotal = source["fiscalYearCumulativeTotal"];
	        this.quarterPeriod = source["quarterPeriod"];
	        this.quarterTotal = source["quarterTotal"];
	    }
	}
	export class MonthlyStatsResponse {
	    rows: MonthlyStatRow[];
	
	    static createFrom(source: any = {}) {
	        return new MonthlyStatsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rows = this.convertValues(source["rows"], MonthlyStatRow);
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
	export class RegisteredUser {
	    cardId: string;
	    name: string;
	    furigana: string;
	    gender: string;
	    roleName: string;
	    roleCode: number;
	    studentNo: string;
	    adminNo: string;
	    purpose: string;
	    contact: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new RegisteredUser(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cardId = source["cardId"];
	        this.name = source["name"];
	        this.furigana = source["furigana"];
	        this.gender = source["gender"];
	        this.roleName = source["roleName"];
	        this.roleCode = source["roleCode"];
	        this.studentNo = source["studentNo"];
	        this.adminNo = source["adminNo"];
	        this.purpose = source["purpose"];
	        this.contact = source["contact"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class UserStatus {
	    cardId: string;
	    userName: string;
	    roleName: string;
	    studentNo: string;
	    currentStatus: string;
	    // Go type: time
	    lastEventTime: any;
	    stayDuration: string;
	
	    static createFrom(source: any = {}) {
	        return new UserStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cardId = source["cardId"];
	        this.userName = source["userName"];
	        this.roleName = source["roleName"];
	        this.studentNo = source["studentNo"];
	        this.currentStatus = source["currentStatus"];
	        this.lastEventTime = this.convertValues(source["lastEventTime"], null);
	        this.stayDuration = source["stayDuration"];
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

}

export namespace service {
	
	export class SwipeResponse {
	    success: boolean;
	    errorMessage?: string;
	    cardId: string;
	    userName: string;
	    roleName: string;
	    roleCode: number;
	    studentNo: string;
	    eventType: string;
	    timestamp: string;
	    durationText: string;
	    soundType: string;
	    isDebounced: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SwipeResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.errorMessage = source["errorMessage"];
	        this.cardId = source["cardId"];
	        this.userName = source["userName"];
	        this.roleName = source["roleName"];
	        this.roleCode = source["roleCode"];
	        this.studentNo = source["studentNo"];
	        this.eventType = source["eventType"];
	        this.timestamp = source["timestamp"];
	        this.durationText = source["durationText"];
	        this.soundType = source["soundType"];
	        this.isDebounced = source["isDebounced"];
	    }
	}

}

