package strategy

// This file is responsible for holding presets for strategy definitions.

import (
	"recommender/assessment"
	"recommender/schematree"
)

// Helper method to create a condition that always evaluates to true.
func makeAlwaysCondition() Condition {
	return func(asm *assessment.Instance) bool {
		return true
	}
}

// Helper method to create the above-threshold condition.
func makeAboveThresholdCondition(threshold int) Condition {
	return func(asm *assessment.Instance) bool {
		return len(asm.Props) > threshold
	}
}

// Helper Method to create too-many-recommendations-condition: When the standard recommender returns more than count many recommendations the condition is true, else false
func makeTooManyRecommendationsCondition(threshold int) Condition {
	return func(asm *assessment.Instance) bool {
		recommendation := asm.CalcRecommendations()
		if len(recommendation) > threshold {
			return true
		}
		return false
	}
}

// Helper Method to create too-few-recommendations-condition: When the standard recommender returns less than count many recommendations the condition is true, else false
func makeTooFewRecommendationsCondition(threshold int) Condition {
	return func(asm *assessment.Instance) bool {
		recommendation := asm.CalcRecommendations()
		if len(recommendation) < threshold {
			return true
		}
		return false
	}
}

// Helper method to create the direct SchemaTree procedure call.
func makeDirectProcedure(tree *schematree.SchemaTree) Procedure {
	return func(asm *assessment.Instance) schematree.PropertyRecommendations {
		return tree.RecommendProperty(asm.Props)
	}
}

// Helper method to create the direct SchemaTree procedure call.
func makeAssessmentAwareDirectProcedure() Procedure {
	return func(asm *assessment.Instance) schematree.PropertyRecommendations {
		return asm.CalcRecommendations()
	}
}

// Helper method to create the 'deletelowfrequency' backoff procedure.
func makeDeleteLowFrequencyProcedure(tree *schematree.SchemaTree, parExecs int) Procedure {
	b := schematree.NewBackoffDeleteLowFrequencyItems(tree, parExecs, schematree.StepsizeLinear)
	return func(asm *assessment.Instance) schematree.PropertyRecommendations {
		return b.Recommend(asm.Props)
	}
}

// Helper method to create the 'splitproperty' backoff procedure.
// TODO: This method could be changed to allow for customized splitter and merger functions.
func makeSplitPropertyProcedure(tree *schematree.SchemaTree) Procedure {
	b := schematree.NewBackoffSplitPropertySet(tree, schematree.TwoSupportRangesSplitter, schematree.AvgMerger)
	return func(asm *assessment.Instance) schematree.PropertyRecommendations {
		return b.Recommend(asm.Props)
	}
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

	// Test to show that recommendations can be called on conditions, and that a
	// assessment-aware procedure can use those recommendations.
	case "toofewrecommendations":
		wf.Push(
			makeTooFewRecommendationsCondition(10),
			makeDeleteLowFrequencyProcedure(tree, 4),
			"if less than 10 recommendations are generated, run the deletelowfrequency backoff",
		)
		wf.Push(
			makeAlwaysCondition(),
			makeAssessmentAwareDirectProcedure(),
			"default to direct algorithm, but use assessment cache if possible",
		)

	// Calls the schematree core algorithm directly.
	case "direct":
		wf.Push(
			makeAlwaysCondition(),
			makeDirectProcedure(tree),
			"always run direct algorithm",
		)

	default:
		panic("Given strategy name does not exist as a preset.")
	}

	return wf
}
