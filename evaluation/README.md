## Generate Config Files:
1) Create a Creater Config File (see ./configs/generate_all.json and ./configs/README.md)
2) Build go build . 
3) Run `./evaluation -createConfigs -creater currentCreater``

## Run Batch Test:
Runs all the config files ./configs/config_i.json in in 1...n
1) go build .
2) Run ` ./evaluation -batchTest -numberConfigs n -testSet ../testdata/10M.nt_1in2_test.gz -model ../testdata/10M.nt_1in2_train.gz.schemaTree.bin `

## Run Single Test: 
**Runs the standard recommender**
1) go build .
2) Run `./evaluation -testSet ../testdata/10M.nt_1in2_test.gz -model ../testdata/10M.nt_1in2_train.gz.schemaTree.bin`

**Run the recommender with types (note that the schematree needs to support type info  then)**
1) go build . 
2) Run `./evaluation -model ../testdata/10M.nt_1in2_train.gz.schemaTree.typed.bin -testSet ../testdata/10M.nt_1in2_test.gz  -typed`

**Run the recommender with types and workflow config (note that the schematree needs to support type info  then)**
1) go build . 
2) Run `./evaluation -model ../testdata/10M.nt_1in2_train.gz.schemaTree.typed.bin -testSet ../testdata/10M.nt_1in2_test.gz  -typed -workflow ../testdata/workflow.json`

note that you need to replace the names for the schematree the test set and the workflow config json file