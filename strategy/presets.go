package strategy

import (
	"recommender/schematree"
)

// This file is responsible for holding presets for strategy definitions.

// Helper method to create a condition that always evaluates to true.
func makeAlwaysCondition() Condition {
	return func(props schematree.IList) bool {
		return true
	}
}

// Helper method to create the above-threshold condition.
func makeAboveThresholdCondition(threshold int) Condition {
	return func(props schematree.IList) bool {
		return len(props) > threshold
	}
}

// Helper method to create the direct SchemaTree procedure call.
func makeDirectProcedure(tree *schematree.SchemaTree) Procedure {
	return tree.RecommendProperty
}

// Helper method to create the 'deletelowfrequency' backoff procedure.
func makeDeleteLowFrequencyProcedure(tree *schematree.SchemaTree, parExecs int) Procedure {
	b := schematree.NewBackoffDeleteLowFrequencyItems(tree, parExecs, schematree.StepsizeLinear)
	return b.Recommend
}

// Helper method to create the 'splitproperty' backoff procedure.
// TODO: This method could be changed to allow for customized splitter and merger functions.
func makeSplitPropertyProcedure(tree *schematree.SchemaTree) Procedure {
	b := schematree.NewBackoffSplitPropertySet(tree, schematree.TwoSupportRangesSplitter, schematree.AvgMerger)
	return b.Recommend
}

// MakePresetStrategy : Build a preset strategy that is hard-coded.
func MakePresetStrategy(name string, tree *schematree.SchemaTree) Workflow {
	wf := Workflow{}

	switch name {

	// Will always call the deleteLowFrequency backoff algorithm.
	case "deletelowfrequency":
		wf.Push(
			makeAlwaysCondition(),
			makeDeleteLowFrequencyProcedure(tree, 4),
			"always run deletelowfrequency with 4 parallel processes",
		)

	// Will always call the splitProperty backoff algorithm.
	case "splitproperty":
		wf.Push(
			makeAboveThresholdCondition(2),
			makeSplitPropertyProcedure(tree),
			"with 3 or more properties run splitproperty",
		)
		wf.Push(
			makeAlwaysCondition(),
			makeDirectProcedure(tree),
			"default to running direct algorithm",
		)

	// Calls the schematree core algorithm directly.
	case "direct":
		wf.Push(
			makeAlwaysCondition(),
			makeDirectProcedure(tree),
			"always run direct algorithm",
		)

	}

	return wf
}
