package glossary

import (
	"recommender/schematree"
)

// LabeledRecommendation are recommendations with glossary information attached
type LabeledRecommendation struct {
	Property    *schematree.IItem
	Content     *Content
	Probability float64
}

// TranslateRecommendations adds glossary information to recommendations from the schematree
func TranslateRecommendations(glossary *Glossary, language string, recommendations schematree.PropertyRecommendations) []LabeledRecommendation {
	labeledRecommendations := make([]LabeledRecommendation, len(recommendations))
	for i, candidate := range recommendations {

		property := candidate.Property
		content, ok := (*glossary)[Key{*property.Str, language}]
		if !ok { // no reference in given language -> try english
			content, ok = (*glossary)[Key{*property.Str, "en"}]
			if !ok { //no english reference -> use template
				content = &Content{"", ""}
			}
		}

		// Whenever the label does not exist, use the actual property url
		if content.Label == "" {
			content.Label = *property.Str
		}

		labeledRecommendations[i] = LabeledRecommendation{property, content, candidate.Probability}
	}
	return labeledRecommendations
}
