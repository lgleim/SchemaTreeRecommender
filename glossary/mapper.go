package glossary

import (
	"recommender/schematree"
)

type LabeledRecommendation struct {
	Content     *Content
	Probability float64
}

func translateRecommendations(glossary *Glossary, language string, recommendations schematree.PropertyRecommendations) []LabeledRecommendation {
	labeledRecommendations := make([]LabeledRecommendation, len(recommendations))
	for i, candidate := range recommendations {

		property := candidate.Property.Str
		content, ok := (*glossary)[Key{*property, language}]
		if !ok { // no reference in given language -> try english
			content, ok = (*glossary)[Key{*property, "en"}]
			if !ok { //no english reference -> use template
				content = &Content{"label", "description"}
			}
		}
		labeledRecommendations[i] = LabeledRecommendation{content, candidate.Probability}
	}
	return labeledRecommendations
}
