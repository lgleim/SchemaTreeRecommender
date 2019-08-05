# Recommender Evaluation suite

## Generate Config Files:
1) Create a Creater Config File (see ./configs/generate_all.json and ./configs/README.md)
2) Build go build . 
3) Run `./evaluation -createConfigs -creater currentCreater``

## Run Batch Test:
Runs all the config files ./configs/config_i.json in in 1...n
1) `go build .`
2) Run ` ./evaluation -batchTest -numberConfigs n -testSet ../testdata/10M.nt_1in2_test.gz -model ../testdata/10M.nt_1in2_train.gz.schemaTree.bin `

## Run Single Test: 

**Runs the standard recommender**
1) `go build .`
2) Run `./evaluation -testSet ../testdata/10M.nt_1in2_test.gz -model ../testdata/10M.nt_1in2_train.gz.schemaTree.bin`

**Run the recommender with types (note that the schematree needs to support type info  then)**
1) `go build .`
2) Run `./evaluation -model ../testdata/10M.nt_1in2_train.gz.schemaTree.typed.bin -testSet ../testdata/10M.nt_1in2_test.gz  -typed`

**Run the recommender with types and workflow config (note that the schematree needs to support type info  then)**
1) `go build .`
2) Run `./evaluation -model ../testdata/10M.nt_1in2_train.gz.schemaTree.typed.bin -testSet ../testdata/10M.nt_1in2_test.gz  -typed -workflow ../testdata/workflow.json`

note that you need to replace the names for the schematree the test set and the workflow config json file

## Example of an evaluation script

What follows is an example of an evaluation script that is useful to generate statistics in multiple views in one go. It will run evaluations with multiple combinations of models, workflows, handlers and statistic groups. After the evaluations are performed it will package the results for easier downloading.

```bash
#!/usr/bin/env bash

# Change this to zip all final files into another zip
RUN="r1"
TESTBASE="truthy-item-filtered-sorted-1pm"
MODELBASE="truthy-item-filtered-sorted-999pm"

BIN="$HOME/go/src/recommender/evaluation/evaluation"
COMMON="-testSet $HOME/data/$TESTBASE.nt.gz"

STDTREE="-model $HOME/data/$MODELBASE-stdTree.bin"
TYPEDTREE="-model $HOME/data/$MODELBASE-typedTree.bin -typed"
TYPEDBACKOFFTREE="-model $HOME/data/$MODELBASE-typedTree.bin -typed -workflow $HOME/data/tooFewRecommendations.json"

TAKEONE="-handler takeOneButType"
TAKEMANY="-handler takeAllButBest"
TAKEITER="-handler takeMoreButCommon"

BYSETSIZE="-groupBy setSize"
BYLEFTOUT="-groupBy numLeftOut"
BYNONTYPES="-groupBy numNonTypes"


$BIN $COMMON $STDTREE          $TAKEONE  $BYSETSIZE  -name $RUN-standard-takeOneButType-setSize
$BIN $COMMON $TYPEDTREE        $TAKEONE  $BYSETSIZE  -name $RUN-typed-takeOneButType-setSize
$BIN $COMMON $TYPEDBACKOFFTREE $TAKEONE  $BYSETSIZE  -name $RUN-typed-tooFewRecs-takeOneButType-setSize

$BIN $COMMON $STDTREE          $TAKEMANY $BYNONTYPES -name $RUN-standard-takeAllButBest-numNonTypes
$BIN $COMMON $TYPEDTREE        $TAKEMANY $BYNONTYPES -name $RUN-typed-takeAllButBest-numNonTypes
$BIN $COMMON $TYPEDBACKOFFTREE $TAKEMANY $BYNONTYPES -name $RUN-typed-tooFewRecs-takeAllButBest-numNonTypes

$BIN $COMMON $STDTREE          $TAKEMANY $BYLEFTOUT  -name $RUN-standard-takeAllButBest-numLeftOut
$BIN $COMMON $TYPEDTREE        $TAKEMANY $BYLEFTOUT  -name $RUN-typed-takeAllButBest-numLeftOut
$BIN $COMMON $TYPEDBACKOFFTREE $TAKEMANY $BYLEFTOUT  -name $RUN-typed-tooFewRecs-takeAllButBest-numLeftOut


# Make a zip of all these essential constellations. Then do some extra that might not finish in time.
(cd $HOME/data && tar cvzf $TESTBASE-evals-$RUN-essentials.tar.gz $TESTBASE-$RUN-*)


$BIN $COMMON $STDTREE          $TAKEONE  $BYNONTYPES -name $RUN-standard-takeOneButType-numNonTypes
$BIN $COMMON $TYPEDTREE        $TAKEONE  $BYNONTYPES -name $RUN-typed-takeOneButType-numNonTypes
$BIN $COMMON $TYPEDBACKOFFTREE $TAKEONE  $BYNONTYPES -name $RUN-typed-tooFewRecs-takeOneButType-numNonTypes

$BIN $COMMON $STDTREE          $TAKEITER $BYNONTYPES -name $RUN-standard-takeMoreButCommon-numNonTypes
$BIN $COMMON $TYPEDTREE        $TAKEITER $BYNONTYPES -name $RUN-typed-takeMoreButCommon-numNonTypes
$BIN $COMMON $TYPEDBACKOFFTREE $TAKEITER $BYNONTYPES -name $RUN-typed-tooFewRecs-takeMoreButCommon-numNonTypes


# Make a zip of everything
(cd $HOME/data && tar cvzf $TESTBASE-evals-$RUN-everything.tar.gz $TESTBASE-$RUN-*)
```

