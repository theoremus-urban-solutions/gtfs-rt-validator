//go:build race

package validator

// raceEnabled is true under -race, which slows the load too much for its time budget.
const raceEnabled = true
