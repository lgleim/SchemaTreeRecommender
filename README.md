# SchemaTree Recommender

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

## Example (old case)

```
# Build the treebuilder
cd treebuilder
go build .
cd ..

# Construct the schematree
./treebuilder/treebuilder -file 10M.nt.gz

# Serve recommender REST API for that file on port 8080
./treebuilder/treebuilder -load 10M.nt.gz.schemaTree.bin -api -port 8080

# Make a request to the recommender
curl -d '["http://www.wikidata.org/prop/direct/P31","http://www.wikidata.org/prop/direct/P21","http://www.wikidata.org/prop/direct/P27"]' http://localhost:8080/recommender

# Make the same request to the "special" wikiRecommender endpoint
curl -d '["P31","P21","P27"]' http://localhost:8080/wikiRecommender
```

### Note

If you want to run on the full wikidata dataset, grab the latest dump from https://dumps.wikimedia.org/wikidatawiki/entities/latest-truthy.nt.gz