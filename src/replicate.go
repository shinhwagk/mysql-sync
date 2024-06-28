package main

type Replicate struct {
	DoDBs      []string
	IgnoreDBs  []string
	DoTabs     []string
	IgnoreTabs []string
	// WildDoTable     string
	// WildIgnoreTable string
}

func NewReplicateFilter(config *ReplicateConfig) *Replicate {
	var DoDBs, IgnoreDBs, DoTabs, IgnoreTabs []string

	if config != nil {
		DoDBs = splitAndClean(config.DoDB)
		IgnoreDBs = splitAndClean(config.IgnoreDB)
		DoTabs = splitAndClean(config.DoTable)
		IgnoreTabs = splitAndClean(config.IgnoreTable)
	}

	IgnoreDBs = append(IgnoreDBs, "mysql")
	return &Replicate{DoDBs, IgnoreDBs, DoTabs, IgnoreTabs}
}

func (rf *Replicate) filter(db string, tab string) bool {
	dbTabName := db + "." + tab

	if len(rf.IgnoreTabs) > 0 && contains(dbTabName, rf.IgnoreTabs) {
		return false
	}

	if len(rf.IgnoreDBs) > 0 && contains(db, rf.IgnoreDBs) {
		return false
	}

	if len(rf.DoDBs) > 0 && !contains(db, rf.DoDBs) {
		return false
	}

	if len(rf.DoTabs) > 0 && !contains(dbTabName, rf.DoTabs) {
		return false
	}

	return true
}
