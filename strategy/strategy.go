package strategy

// TODO: Right now the Strategy workflow receives an assessment which it can use to
//       follow the workflow and query if needed. Ideally, the role of creating an
//       assessment could fall upon the Strategy itself. To make it work like that
//       I would need to create an additional struct that creates an assessment on
//       construction, and that holds a pointer to a workflow to follow. This change
//       would be pedantically more correct. For now, I haven't done it because it
//       provides no real benefits except for giving a guarantee that outside forces
//       do not alter the assessment from the outside (right now we are optimistic
//       that the user of Strategies will only make the right interactions with an
//       assessment - creating it and then only delivering it to the strategy)

import (
	"log"
	"recommender/assessment"
	"recommender/schematree"
)

// Condition : Evaluates is a given strategy entry should run.
type Condition func(*assessment.Instance) bool

// Procedure : Procedure to run as a strategy entry.
type Procedure func(*assessment.Instance) schematree.PropertyRecommendations

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
func (wf *Workflow) Recommend(asm *assessment.Instance) schematree.PropertyRecommendations {
	log.Println("Starting the strategy workflow:")
	for _, step := range *wf {
		if step.check(asm) {
			log.Printf("  Run entry '%s'", step.desc)
			return step.run(asm)
		}
		log.Printf("  Skip entry '%s'", step.desc)
	}
	log.Printf("  Failed to select any entry of the strategy workflow.")
	return nil
}
