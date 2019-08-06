# Schematree Module

The Schematree Module contains the main datastructure and the standart recommendation algorithm.

SchemaTree.go, SchemaNode.go and DataTypes.go form the tree data structure. SubjectReader.go reads property / type informations from a rdf file. Recommendation.go performs the recommendations.

## Interface

Create(filename string, firstNsubjects uint64, typed bool, minSup uint32) creates a new Schematree from a rdf file
Load(filePath string) loads a schematree from a encoded file

Recommend(properties []string, types []string) recommends a list of property candidates
