package services

// The limits of the plan and task file search.

const (
	// maxPlanFileBytes: a bigger file is not a plan or task document.
	maxPlanFileBytes = 256 << 10
	// maxPlanWalkDepth bounds the search of a whole project (maxPlanWalkEntries, below, bounds its size).
	maxPlanWalkDepth = 6
	// maxPlanFolderDepth is how deep below a plan or task folder files are read.
	maxPlanFolderDepth = 3
)

// maxPlanWalkEntries is how many files and folders the search of a whole project looks at before it
// gives up; tests lower it.
var maxPlanWalkEntries = 20000

// planSkipDirs are never searched for plan files.
var planSkipDirs = map[string]bool{
	"node_modules": true, "vendor": true, "build": true, "dist": true, "target": true,
	"__pycache__": true, "coverage": true, "out": true, "tmp": true,
}
