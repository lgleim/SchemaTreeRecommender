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

## Example of a data preparation script (untested)

This is an example of how a complete data preparation pipeline could run. It also includes a 1:999 split of the dataset which is usually omitted for production usage.

```bash
#!/usr/bin/env bash

# Edit these values for personal usage
BIN="$HOME/go/src/recommender/recommender"
MAKEDOWNLOAD=false
URL="https://dumps.wikimedia.org/wikidatawiki/entities/latest-truthy.nt.gz"
DSBASE="$HOME/data/truthy" # file where dataset is stored without '.nt.gz' extension
MAKEGLOSSARY=false
STARTSERVER=false # also requires that glossary is made



# Only make download if flag is set
if [ "$MAKEDOWNLOAD" = true ] ; then
    curl $URL --output $DSBASE.nt.gz
fi

# Split and filter preparation steps
$BIN split-dataset  by-type $DBBASE.nt.gz
$BIN filter-dataset for-schematree $DBBASE-item.nt.gz
gzip -cd $DBBASE-item-filtered.nt.gz | sort | gzip > $DBBASE-item-filtered-sorted.nt.gz
$BIN split-dataset  1-in-n $DBBASE-item-filtered-sorted.nt.gz -n 1000

$BIN build-tree       $DBBASE-item-filtered-sorted-1in1000-train.nt.gz
echo "Standard Tree model was probably stored in: $DSBASE-item-filtered-sorted-1in1000-train.nt.gz.schemaTree.bin"

$BIN build-tree-typed $DBBASE-item-filtered-sorted-1in1000-train.nt.gz
echo "Typed Tree model was probably stored in: $DSBASE-item-filtered-sorted-1in1000-train.nt.gz.schemaTree.typed.bin"

# Only treat glossary if flag is set
if [ "$MAKEGLOSSARY" = true ] ; then
    $BIN filter-dataset for-glossary $DBBASE-prop.nt.gz
    $BIN build-glossary $DBBASE-prop-filtered.nt.gz
fi

# Only start webserver if flag is set
if [ "$STARTSERVER" = true ] ; then
    $BIN serve $DSBASE-item-filtered-sorted-1in1000-train.nt.gz.schemaTree.typed.bin $DBBASE-prop-filtered.nt.gz.glossary.bin
fi
```

## Example of an evaluation script

What follows is an example of an evaluation script that is useful to generate statistics in multiple views in one go. It will run evaluations with multiple combinations of models, workflows, handlers and statistic groups. After the evaluations are performed it will package the results for easier downloading.

```bash
#!/usr/bin/env bash

# Change this to zip all final files into another zip
BIN="$HOME/go/src/recommender/evaluation/evaluation"
RUN="r1"
TESTBASE="truthy-item-filtered-sorted-1pm"     # truthy-item-filtered-sorted-1in1000-test
MODELBASE="truthy-item-filtered-sorted-999pm"  # truthy-item-filtered-sorted-1in1000-train



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

