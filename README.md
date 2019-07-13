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
# (this example will assume that a dataset called `./data/10M.nt.gz` exists)

# Split the dataset for wikidata items and properties
# (TODO: The handcrafted dataset has to be improved  with a better combination of entries)
./recommender split-dataset by-type ./testdata/handcrafted.nt

# Prepare the dataset and build the Schema Tree
# (TODO: add commands for Typed Schema Trees)
./recommender filter-dataset for-schematree ./testdata/handcrafted.nt.item.gz 
./recommender build-tree ./testdata/handcrafted.nt.item.gz.filtered.gz

# Prepare the dataset and build the Glossary
./recommender filter-dataset for-glossary ./testdata/handcrafted.nt.prop.gz
./recommender build-glossary ./testdata/handcrafted.nt.prop.gz.filtered.gz

# Start the server 
# (TODO: shorten the big file names - should edit the names before the extension: data.nt.gz to data-item.nt.gz)
# (TODO: add information about workflow strategies)
./recommender serve ./testdata/handcrafted.nt.item.gz.filtered.gz.schemaTree.bin ./testdata/handcrafted.nt.prop.gz.filtered.gz.glossary.bin

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