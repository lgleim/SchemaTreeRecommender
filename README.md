# SchemaTree Recommender

[![Build Status](https://git.rwth-aachen.de/kglab2019/recommender/badges/master/pipeline.svg)](https://git.rwth-aachen.de/kglab2019/recommender/commits/master)
[![GitHub issues](https://img.shields.io/github/issues/lgleim/SchemaTreeRecommender)](https://github.com/lgleim/SchemaTreeRecommender/issues)
[![Test Coverage](https://git.rwth-aachen.de/kglab2019/recommender/badges/master/coverage.svg)](https://git.rwth-aachen.de/kglab2019/recommender/commits/master)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0) 

Individual descriptions in subfolders.

### Publication

Further documentation and an evaluation of this project can be found in our [ESWC 2020](https://2020.eswc-conferences.org/) publication: https://doi.org/10.1007/978-3-030-49461-2_11

Cite as:

```
Gleim L.C. et al. (2020) SchemaTree: Maximum-Likelihood Property Recommendation for Wikidata. 
In: Harth A. et al. (eds) The Semantic Web. ESWC 2020. Lecture Notes in Computer Science, vol 12123. Springer, Cham
```

Or via BibTeX:

```tex
@InProceedings{10.1007/978-3-030-49461-2_11,
  author="Gleim, Lars C.
  and Schimassek, Rafael
  and H{\"u}ser, Dominik
  and Peters, Maximilian
  and Kr{\"a}mer, Christoph
  and Cochez, Michael
  and Decker, Stefan",
  editor="Harth, Andreas
  and Kirrane, Sabrina
  and Ngonga Ngomo, Axel-Cyrille
  and Paulheim, Heiko
  and Rula, Anisa
  and Gentile, Anna Lisa
  and Haase, Peter
  and Cochez, Michael",
  title="SchemaTree: Maximum-Likelihood Property Recommendation for Wikidata",
  booktitle="The Semantic Web",
  year="2020",
  publisher="Springer International Publishing",
  address="Cham",
  pages="179--195",
  abstract="Wikidata is a free and open knowledge base which can be read and edited by both humans and machines. It acts as a central storage for the structured data of several Wikimedia projects. To improve the process of manually inserting new facts, the Wikidata platform features an association rule-based tool to recommend additional suitable properties. In this work, we introduce a novel approach to provide such recommendations based on frequentist inference. We introduce a trie-based method that can efficiently learn and represent property set probabilities in RDF graphs. We extend the method by adding type information to improve recommendation precision and introduce backoff strategies which further increase the performance of the initial approach for entities with rare property combinations. We investigate how the captured structure can be employed for property recommendation, analogously to the Wikidata PropertySuggester. We evaluate our approach on the full Wikidata dataset and compare its performance to the state-of-the-art Wikidata PropertySuggester, outperforming it in all evaluated metrics. Notably we could reduce the average rank of the first relevant recommendation by 71{\%}.",
  isbn="978-3-030-49461-2"
}
```

## Installation

1. Install the go runtime (and VS Code + Golang tools)
1. Run `go get .` in this folder to install all dependencies
1. Run `go build .` in this folder to build the executable
1. Run `go install .` to install the executable in the $PATH

## Example 

```bash

# This example will assume that you are in the top directory.

# Download a dataset, for example the latest 32GB dataset from wikidata
# curl https://dumps.wikimedia.org/wikidatawiki/entities/latest-truthy.nt.gz --output latest-truthy.nt.gz
# (this example will assume that a dataset called `./testdata/handcrafted.nt` exists)

# Split the dataset for wikidata items and properties
# (TODO: The handcrafted dataset has to be improved  with a better combination of entries)
./recommender split-dataset by-prefix ./testdata/handcrafted.nt

# Prepare the dataset and build the Schema Tree (typed variant) (the sort is only required for future 1-in-n splits)
./recommender filter-dataset for-schematree ./testdata/handcrafted-item.nt.gz 
gzip -cd ./testdata/handcrafted-item-filtered.nt.gz | sort | gzip > ./testdata/handcrafted-item-filtered-sorted.nt.gz
./recommender build-tree-typed ./testdata/handcrafted-item-filtered-sorted.nt.gz

# Prepare the dataset and build the Glossary
./recommender filter-dataset for-glossary ./testdata/handcrafted-prop.nt.gz
gzip -cd ./testdata/handcrafted-prop-filtered.nt.gz | sed -r -e 's|^<http:\/\/www\.wikidata\.org\/entity\/P([^>]+)>|<http://www.wikidata.org/prop/direct/P\1>|g' | gzip > ./testdata/handcrafted-prop-filtered-altered.nt.gz
./recommender build-glossary ./testdata/handcrafted-prop-filtered-altered.nt.gz

# Start the server 
# (TODO: add information about workflow strategies)
./recommender serve ./testdata/handcrafted-item-filtered-sorted.schemaTree.typed.bin ./testdata/handcrafted-prop-filtered-altered.glossary.bin

# Test with a request 
curl -d '{"lang":"en","properties":["local://prop/Color"],"types":[]}' http://localhost:8080/recommender

```

### Note

If you want to run on the full wikidata dataset, grab the latest dump from https://dumps.wikimedia.org/wikidatawiki/entities/latest-truthy.nt.gz`

### Performance Evaluation Details

| Dataset | Results |
| ------ | ------ |
| Wikidata | [here](evaluation/visualization_single_evaluation_wiki.ipynb) |
| LOD-a-lot | [here](evaluation/visualization_single_evaluation-LOD.ipynb) |
| Backoff strategies | [here](evaluation/visualization_batch.ipynb) |
 

