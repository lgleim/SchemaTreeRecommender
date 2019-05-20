### SchemaTree Recommender
Individual descriptions in subfolders.

As part of the KGLab only the `treebuilder` subfolder and the `.go` files in this folder are really relevant, since the code for building the tree as well as code for computing recommendations resides in those files/folder.

The code quality is mediocre. More comments will follow.

### Installation
1. Install the go runtime (and VS Code + Golang tools)
2. Run `go get .` in this folder to install all dependencies
3. Compile the individual tools by running `go build .` in the respective folders, most importantly the `treebuilder` folder.