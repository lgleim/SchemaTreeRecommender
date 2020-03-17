## Generate Config Files:
go build . 
./evaluation -createConfigs -creater configs/wikiEval.json

## Run Batch Test:
# Runs all the config files ./configs/config_i.json in in 0...96
./evaluation -batchTest -numberConfigs 96 \
    -testSet ../testdata/10M.nt_1in2_test.gz \
    -model ../testdata/10M.nt_1in2_train.gz.schemaTree.bin
