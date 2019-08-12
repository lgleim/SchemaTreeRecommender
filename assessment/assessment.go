package assessment

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"recommender/schematree"
	"strings"
	"time"
)

var netClient = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:       10,
		MaxConnsPerHost:    5,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	},
	Timeout: time.Second * 120,
}

// Instance - An assessment on properties
//
// COMMENT: The assessment module might be too tighly coupled with the SchemaTree. In the
//          early days the SchemaTree delegate the job of creating the input arguments to
//          the caller of the method. The caller would create the IList and then pass it
//          to the SchemaTree. This behaviour should be avoid because it makes too much
//          internal information visible to the outside. The correct behaviour is to accept
//          an array of strings (or byte-arrays) and then construct the IList oneself.
//          To fix this issue in the future, please consider making IList and IItem private
//          and then use the schematree.Recommend(props []string, types []string).
//          Likewise, assessments should be working with arrays of strings and not IItems.
//          The benefit in the current method is a faster evaluation since the IList
//          construction does not need to be done multiple times.
type Instance struct {
	Props                 schematree.IList
	tree                  *schematree.SchemaTree
	useOptimisticCache    bool // using cache will make an optimistic assumption that `props` are not altered
	cachedRecommendations schematree.PropertyRecommendations
}

// NewInstance : constructor method
func NewInstance(argProps schematree.IList, argTree *schematree.SchemaTree, argUseCache bool) *Instance {
	return &Instance{
		Props:                 argProps,
		tree:                  argTree,
		useOptimisticCache:    argUseCache,
		cachedRecommendations: nil,
	}
}

// NewInstanceFromInput : constructor method to receive strings and convert them into the current
// assessment format that uses IList.
func NewInstanceFromInput(argProps []string, argTypes []string, argTree *schematree.SchemaTree, argUseCache bool) *Instance {
	propList := argTree.BuildPropertyList(argProps, argTypes)

	return &Instance{
		Props:                 propList,
		tree:                  argTree,
		useOptimisticCache:    argUseCache,
		cachedRecommendations: nil,
	}
}

// CalcPropertyLength : Calculate the amount of properties.
func (inst *Instance) CalcPropertyLength() int {
	return len(inst.Props)
}

// CalcRecommendations : Will execute the core schematree recommender on the properties and return
// the list of recommendations. Cache-enabled operation.
func (inst *Instance) CalcRecommendations() schematree.PropertyRecommendations {
	if inst.useOptimisticCache == true {
		if inst.cachedRecommendations == nil {
			inst.cachedRecommendations = inst.tree.RecommendProperty(inst.Props)
		}
		return inst.cachedRecommendations
	}
	return inst.tree.RecommendProperty(inst.Props)
}

// GetWikiRecs computes recommendations from a local wikidata PropertySuggester
func (inst *Instance) GetWikiRecs(Properties []string) schematree.PropertyRecommendations {
	// url := "https://www.wikidata.org/w/api.php?action=wbsgetsuggestions&limit=10&format=json&properties=" + strings.Join(Properties, "|")
	url := "http://localhost:8181/w/api.php?action=wbsgetsuggestions&format=json&properties=" + strings.Join(Properties, "|")
	res, err := netClient.Get(url)
	if err != nil {
		panic(err)
	}
	if res.StatusCode != 200 {
		b, _ := ioutil.ReadAll(res.Body)
		panic(fmt.Sprint(url, string(b)))
	}
	var recs struct {
		Search []struct {
			ID string `json:"id"`
			// Rating float64 `json:"rating"`
		} `json:"search"`
	}
	err = json.NewDecoder(res.Body).Decode(&recs)
	if err != nil {
		panic(fmt.Sprintf("received malformatted response from wikidata recommender for property set %v. Error: %v", Properties, err))
	}
	// type RankedPropertyCandidate struct {
	// 	Property    *IItem
	// 	Probability float64
	// }
	ranked := make([]schematree.RankedPropertyCandidate, 0, len(recs.Search))
	for _, r := range recs.Search {
		item, ok := inst.tree.PropMap["http://www.wikidata.org/prop/direct/"+r.ID]
		// if !ok {
		// 	item, ok = inst.tree.PropMap["http://www.wikidata.org/prop/"+r.ID]
		// }
		if ok {
			ranked = append(ranked, schematree.RankedPropertyCandidate{
				Property:    item,
				Probability: 0, //r.Rating,
			})
		}
	}
	// fmt.Println(url, ranked)
	return ranked
}
