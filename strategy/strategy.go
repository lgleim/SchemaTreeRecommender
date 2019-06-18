package strategy

import (
	"log"
	"recommender/schematree"
)

// Condition : Evaluates is a given strategy entry should run.
type Condition func(schematree.IList) bool

// Procedure : Procedure to run as a strategy entry.
type Procedure func(schematree.IList) schematree.PropertyRecommendations

type entry struct {
	check Condition
	run   Procedure
	desc  string
}

// Workflow : Workflow to execute in order to get best recommendation given the input.
type Workflow []entry

// Push : Add a new strategy entry with less priority than all other existing entries.
func (wf *Workflow) Push(cond Condition, proc Procedure, desc string) {
	*wf = append(*wf, entry{cond, proc, desc})
}

// Recommend : Run the workflow and return the recommended properties.
// Go through the workflow and execute the first procedure that has a valid condition.
// That procedure will return the list of recommended properties.
func (wf *Workflow) Recommend(props schematree.IList) schematree.PropertyRecommendations {
	log.Println("Starting the strategy workflow:")
	for _, wfEntry := range *wf {
		if wfEntry.check(props) {
			log.Printf("  Run entry '%s'", wfEntry.desc)
			return wfEntry.run(props)
		}
		log.Printf("  Skip entry '%s'", wfEntry.desc)
	}
	log.Printf("  Failed to select any entry of the strategy workflow.")
	return nil
}

// Procedure - Run a split/merger style.
//
// TODO: Use Dominiks code and same style as conditions.
