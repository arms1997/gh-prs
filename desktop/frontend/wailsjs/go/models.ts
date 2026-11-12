export namespace core {
	
	export class AgentSummary {
	    Headline: string;
	    Changes: string[];
	    Pushed: string;
	    NeedsYou: string;
	
	    static createFrom(source: any = {}) {
	        return new AgentSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Headline = source["Headline"];
	        this.Changes = source["Changes"];
	        this.Pushed = source["Pushed"];
	        this.NeedsYou = source["NeedsYou"];
	    }
	}
	export class Check {
	    Name: string;
	    URL: string;
	
	    static createFrom(source: any = {}) {
	        return new Check(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.URL = source["URL"];
	    }
	}

}

export namespace main {
	
	export class CommandView {
	    Key: string;
	    Name: string;
	    Background: boolean;
	    Confirm: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CommandView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Key = source["Key"];
	        this.Name = source["Name"];
	        this.Background = source["Background"];
	        this.Confirm = source["Confirm"];
	    }
	}
	export class JobState {
	    Name: string;
	    Key: string;
	    State: string;
	    Activity: string[];
	    Result: string;
	    Summary?: core.AgentSummary;
	    Cost: number;
	    Turns: number;
	    Elapsed: string;
	
	    static createFrom(source: any = {}) {
	        return new JobState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.Key = source["Key"];
	        this.State = source["State"];
	        this.Activity = source["Activity"];
	        this.Result = source["Result"];
	        this.Summary = this.convertValues(source["Summary"], core.AgentSummary);
	        this.Cost = source["Cost"];
	        this.Turns = source["Turns"];
	        this.Elapsed = source["Elapsed"];
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
	export class PRView {
	    Number: number;
	    Title: string;
	    URL: string;
	    Body: string;
	    IsDraft: boolean;
	    Additions: number;
	    Deletions: number;
	    ChangedFiles: number;
	    HeadRefName: string;
	    BaseRefName: string;
	    HeadOID: string;
	    ReviewDecision: string;
	    Mergeable: string;
	    CI: string;
	    FailingChecks: core.Check[];
	    Repo: string;
	    Author: string;
	    Labels: string[];
	    Unresolved: number;
	    Comments: number;
	    Linear: string;
	    Mine: boolean;
	    UnresolvedNotMine: number;
	    ReviewedSincePush: boolean;
	    Key: string;
	    Status: string;
	    OnMe: boolean;
	    TurnWhy: string;
	    Project: string;
	    Pinned: boolean;
	    Hidden: boolean;
	    Snoozed: boolean;
	    LinearURL: string;
	
	    static createFrom(source: any = {}) {
	        return new PRView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Number = source["Number"];
	        this.Title = source["Title"];
	        this.URL = source["URL"];
	        this.Body = source["Body"];
	        this.IsDraft = source["IsDraft"];
	        this.Additions = source["Additions"];
	        this.Deletions = source["Deletions"];
	        this.ChangedFiles = source["ChangedFiles"];
	        this.HeadRefName = source["HeadRefName"];
	        this.BaseRefName = source["BaseRefName"];
	        this.HeadOID = source["HeadOID"];
	        this.ReviewDecision = source["ReviewDecision"];
	        this.Mergeable = source["Mergeable"];
	        this.CI = source["CI"];
	        this.FailingChecks = this.convertValues(source["FailingChecks"], core.Check);
	        this.Repo = source["Repo"];
	        this.Author = source["Author"];
	        this.Labels = source["Labels"];
	        this.Unresolved = source["Unresolved"];
	        this.Comments = source["Comments"];
	        this.Linear = source["Linear"];
	        this.Mine = source["Mine"];
	        this.UnresolvedNotMine = source["UnresolvedNotMine"];
	        this.ReviewedSincePush = source["ReviewedSincePush"];
	        this.Key = source["Key"];
	        this.Status = source["Status"];
	        this.OnMe = source["OnMe"];
	        this.TurnWhy = source["TurnWhy"];
	        this.Project = source["Project"];
	        this.Pinned = source["Pinned"];
	        this.Hidden = source["Hidden"];
	        this.Snoozed = source["Snoozed"];
	        this.LinearURL = source["LinearURL"];
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
	export class RowView {
	    Title: string;
	    Depth: number;
	    Count: number;
	    Path: string;
	    Stack: number;
	    StackedOn: number;
	    PR?: PRView;
	
	    static createFrom(source: any = {}) {
	        return new RowView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Title = source["Title"];
	        this.Depth = source["Depth"];
	        this.Count = source["Count"];
	        this.Path = source["Path"];
	        this.Stack = source["Stack"];
	        this.StackedOn = source["StackedOn"];
	        this.PR = this.convertValues(source["PR"], PRView);
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
	export class State {
	    Rows: RowView[];
	    Jobs: Record<string, JobState>;
	    Commands: CommandView[];
	    Status: string;
	    IsErr: boolean;
	    Loading: boolean;
	    CheckType: core.Check;
	
	    static createFrom(source: any = {}) {
	        return new State(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Rows = this.convertValues(source["Rows"], RowView);
	        this.Jobs = this.convertValues(source["Jobs"], JobState, true);
	        this.Commands = this.convertValues(source["Commands"], CommandView);
	        this.Status = source["Status"];
	        this.IsErr = source["IsErr"];
	        this.Loading = source["Loading"];
	        this.CheckType = this.convertValues(source["CheckType"], core.Check);
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

