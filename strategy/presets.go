package strategy

// This file is responsible for holding presets for strategy definitions.

import (
	"recommender/assessment"
	"recommender/backoff"
	"recommender/schematree"
)

// Helper method to create a condition that always evaluates to true.
func MakeAlwaysCondition() Condition {
	return func(asm *assessment.Instance) bool {
		return true
	}
}

//Not needed anylonger
// Helper method to create the above-threshold condition.
func MakeAboveThresholdCondition(threshold int) Condition {
	return func(asm *assessment.Instance) bool {
		return len(asm.Props) > threshold
	}
}

func MakeBelowThresholdCondition(threshold int) Condition {
	return func(asm *assessment.Instance) bool {
		return len(asm.Props) < threshold
	}
}

//Not needed anylonger
// Helper Method to create too-many-recommendations-condition: When the standard recommender returns more than count many recommendations the condition is true, else false
func MakeTooManyRecommendationsCondition(threshold int) Condition {
	return func(asm *assessment.Instance) bool {
		recommendation := asm.CalcRecommendations()
		if len(recommendation) > threshold {
			return true
		}
		return false
	}
}

// Helper Method to create too-few-recommendations-condition: When the standard recommender returns less than count many recommendations the condition is true, else false
func MakeTooFewRecommendationsCondition(threshold int) Condition {
	return func(asm *assessment.Instance) bool {
		recommendation := asm.CalcRecommendations()
		if len(recommendation) < threshold {
			return true
		}
		return false
	}
}

// Helper Method to create too-unlikely-recommendations-condition: When the standard recommender returns a recommendation where the top 10 has lower probability than threshhold (in decimal percentage eg 0.5)
func MakeTooUnlikelyRecommendationsCondition(threshold float32) Condition {
	return func(asm *assessment.Instance) bool {
		recommendation := asm.CalcRecommendations()
		if recommendation.Top10AvgProbibility() < threshold {
			return true
		}
		return false
	}
}

// Helper method to create the direct SchemaTree procedure call.
func MakeDirectProcedure(tree *schematree.SchemaTree) Procedure {
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
func MakeDeleteLowFrequencyProcedure(tree *schematree.SchemaTree, parExecs int, stepsize backoff.StepsizeFunc, condition backoff.InternalCondition) Procedure {
	b := backoff.NewBackoffDeleteLowFrequencyItems(tree, parExecs, stepsize, condition)
	return func(asm *assessment.Instance) schematree.PropertyRecommendations {
		return b.Recommend(asm.Props)
	}
}

// Helper method to create the 'splitproperty' backoff procedure.
func MakeSplitPropertyProcedure(tree *schematree.SchemaTree, splitter backoff.SplitterFunc, merger backoff.MergerFunc) Procedure {
	b := backoff.NewBackoffSplitPropertySet(tree, splitter, merger)
	return func(asm *assessment.Instance) schematree.PropertyRecommendations {
		return b.Recommend(asm.Props)
	}
}

// MakePresetWorkflow : Build a preset strategy that is hard-coded.
func MakePresetWorkflow(name string, tree *schematree.SchemaTree) *Workflow {
	wf := Workflow{}

	switch name {

	// Will always call the deleteLowFrequency backoff algorithm.
	case "deletelowfrequency":
		wf.Push(
			MakeAlwaysCondition(),
			MakeDeleteLowFrequencyProcedure(tree, 4, backoff.StepsizeProportional, backoff.MakeMoreThanInternalCondition(10)),
			"always run deletelowfrequency with 4 parallel processes",
		)

	case "best":
		wf.Push(
			MakeTooFewRecommendationsCondition(1),
			MakeDeleteLowFrequencyProcedure(tree, 4, backoff.StepsizeLinear, backoff.MakeMoreThanInternalCondition(4)),
			"run deletelowfrequency with 4 parallel processes",
		)
		wf.Push(
			MakeAlwaysCondition(),
			MakeDirectProcedure(tree),
			"always run direct algorithm",
		)

	// Will always call the splitProperty backoff algorithm.
	case "splitproperty":
		wf.Push(
			MakeAboveThresholdCondition(2),
			MakeSplitPropertyProcedure(tree, backoff.EverySecondItemSplitter, backoff.MaxMerger),
			"with 3 or more properties run splitproperty",
		)
		wf.Push(
			MakeAlwaysCondition(),
			MakeDirectProcedure(tree),
			"default to running direct algorithm",
		)

	// Test to show that recommendations can be called on conditions, and that a
	// assessment-aware procedure can use those recommendations.
	case "toofewrecommendations":
		wf.Push(
			MakeTooFewRecommendationsCondition(10),
			MakeDeleteLowFrequencyProcedure(tree, 4, backoff.StepsizeProportional, backoff.MakeMoreThanInternalCondition(10)),
			"if less than 10 recommendations are generated, run the deletelowfrequency backoff",
		)
		wf.Push(
			MakeAlwaysCondition(),
			makeAssessmentAwareDirectProcedure(),
			"default to direct algorithm, but use assessment cache if possible",
		)

	// Calls the schematree core algorithm directly.
	case "direct":
		wf.Push(
			MakeAlwaysCondition(),
			MakeDirectProcedure(tree),
			"always run direct algorithm",
		)

	default:
		panic("Given strategy name does not exist as a preset.")
	}

	return &wf
}
