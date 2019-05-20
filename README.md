### SchemaTree Recommender
Individual descriptions in subfolders.

As part of the KGLab only the `treebuilder` subfolder and the `.go` files in this folder are really relevant, since the code for building the tree as well as code for computing recommendations resides in those files/folder.

The code quality is mediocre. More comments will follow.

### Installation
1. Install the go runtime (and VS Code + Golang tools)
2. Run `go get .` in this folder to install all dependencies
3. Compile the individual tools by running `go build .` in the respective folders, most importantly the `treebuilder` folder.

### Example
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