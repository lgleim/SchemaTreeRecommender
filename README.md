# SchemaTree Recommender

[![Build Status](https://git.rwth-aachen.de/kglab2019/recommender/badges/master/pipeline.svg)](https://git.rwth-aachen.de/kglab2019/recommender/commits/master)
[![GitHub issues](https://img.shields.io/github/issues/lgleim/SchemaTreeRecommender)](https://github.com/lgleim/SchemaTreeRecommender/issues)
[![Test Coverage](https://git.rwth-aachen.de/kglab2019/recommender/badges/master/coverage.svg)](https://git.rwth-aachen.de/kglab2019/recommender/commits/master)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0) 

Individual descriptions in subfolders.

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

If you want to run on the full wikidata dataset, grab the latest dump from https://dumps.wikimedia.org/wikidatawiki/entities/latest-truthy.nt.gz