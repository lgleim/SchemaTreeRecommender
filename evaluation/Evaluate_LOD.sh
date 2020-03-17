#!/usr/bin/env bash

DOWNLOAD=false
TRAIN=false

if [ "$DOWNLOAD" = true ]; then                                 # Warning! ~600GB 
    wget http://lod-a-lot.lod.labs.vu.nl/data/LOD_a_lot_v1.hdt
    hdt2rdf LOD_a_lot_v1.hdt >(gzip > LOD_a_lot_v1.nt.gz)       # needs https://github.com/rdfhdt/hdt-cpp
fi
if [ "$TRAIN" = true ]; then                                    # Warning! Needs at least 32GB of RAM, better 64
    ../recommender split-dataset 1-in-n LOD_a_lot_v1.nt.gz -n 10000000               # took 4h51m48s.
    zcat LOD_a_lot_v1-1in10000000-test.nt.gz | cut -d " " -f 1 | sort -u | wc -l    # 18952 subjects

    # count the occurrence frequency of each subject set size. columns: (freq.) \t (numNonTypes)
    zcat LOD_a_lot_v1-1in10000000-test.nt.gz \
        | grep -v "<http://www.w3.org/1999/02/22-rdf-syntax-ns#type>" \
        | grep -v "/P31>" \
        | grep -v "<http://dbpedia.org/ontology/type>" \
        | cut -d " " -f 1| sort | uniq -c | cut -d "<" -f 1 | sort | uniq -c \
        > LOD_counts.csv

    # We still build the tree for the full dataset. 
    # Error for including test examples is minimal and we can reuse the tree.
    ../recommender build-tree-typed -f -t ./LOD_a_lot_v1.nt.gz # took around 10h
    ../recommender build-tree -t ./LOD_a_lot_v1.nt.gz          # about the same


    # create an empty glossary (metadata such as rdfs:label) - not needed for eval
    echo "" | gzip > empty.gz
    ../recommender build-glossary empty.gz
fi

# Model may be served as follows:
#../recommender serve LOD_a_lot_v1.nt.gz.schemaTree.typed.bin empty.gz.glossary.bin -w backoff.json # 18GB Ram needed


# Evaluation

# # wait for 1-minute system load average to drop below 100% utilization
# while [ $(cat /proc/loadavg | cut -d " " -f 1 | tr -d '.') -gt 100 ]; do
#     sleep 5
# done

# Change this to zip all final files into another zip
RUN="r1"
EVALBASE="LOD_a_lot_v1.nt.gz"
BIN="./evaluation"
COMMON="-results -testSet LOD_a_lot_v1-1in10000000-test.nt.gz"

STDTREE="-model $EVALBASE.schemaTree.bin"
TYPEDTREE="-model $EVALBASE.schemaTree.typed.bin -typed"
TYPEDBACKOFFTREE="-model $EVALBASE.schemaTree.typed.bin -typed -workflow backoff.json"

# TAKEONE="-handler takeOneButType"
TAKEITER="-handler takeMoreButCommon"

# BYSETSIZE="-groupBy setSize"
# BYLEFTOUT="-groupBy numLeftOut"
BYNONTYPES="-groupBy numNonTypes"

# Makes no sense for the evaluation
# $BIN $COMMON $STDTREE          $TAKEONE  $BYSETSIZE  -name $RUN-standard-takeOneButType-setSize
# $BIN $COMMON $TYPEDTREE        $TAKEONE  $BYSETSIZE  -name $RUN-typed-takeOneButType-setSize
# $BIN $COMMON $TYPEDBACKOFFTREE $TAKEONE  $BYSETSIZE  -name $RUN-typed-tooFewRecs-takeOneButType-setSize

# $BIN $COMMON $STDTREE          $TAKEONE  $BYNONTYPES  -name $RUN-standard-takeOneButType-numNonTypes
# $BIN $COMMON $TYPEDTREE        $TAKEONE  $BYNONTYPES  -name $RUN-typed-takeOneButType-numNonTypes
# $BIN $COMMON $TYPEDBACKOFFTREE $TAKEONE  $BYNONTYPES  -name $RUN-typed-tooFewRecs-takeOneButType-numNonTypes

$BIN $COMMON $STDTREE          $TAKEITER  $BYNONTYPES  -name $RUN-standard-takeMoreButCommon-numNonTypes
$BIN $COMMON $TYPEDTREE        $TAKEITER  $BYNONTYPES  -name $RUN-typed-takeMoreButCommon-numNonTypes
$BIN $COMMON $TYPEDBACKOFFTREE $TAKEITER  $BYNONTYPES  -name $RUN-typed-tooFewRecs-takeMoreButCommon-numNonTypes

# $BIN $COMMON $STDTREE          $TAKEMANY $BYNONTYPES -name $RUN-standard-takeAllButBest-numNonTypes
# $BIN $COMMON $TYPEDTREE        $TAKEMANY $BYNONTYPES -name $RUN-typed-takeAllButBest-numNonTypes
# $BIN $COMMON $TYPEDBACKOFFTREE $TAKEMANY $BYNONTYPES -name $RUN-typed-tooFewRecs-takeAllButBest-numNonTypes

# $BIN $COMMON $STDTREE          $TAKEMANY $BYLEFTOUT  -name $RUN-standard-takeAllButBest-numLeftOut
# $BIN $COMMON $TYPEDTREE        $TAKEMANY $BYLEFTOUT  -name $RUN-typed-takeAllButBest-numLeftOut
# $BIN $COMMON $TYPEDBACKOFFTREE $TAKEMANY $BYLEFTOUT  -name $RUN-typed-tooFewRecs-takeAllButBest-numLeftOut

# # Make a zip of all these essential constellations. Then do some extra that might not finish in time.
# (cd $HOME/data && tar cvzf $TESTBASE-evals-$RUN-essentials.tar.gz $TESTBASE-$RUN-*)


# $BIN $COMMON $STDTREE          $TAKEONE  $BYNONTYPES -name $RUN-standard-takeOneButType-numNonTypes
# $BIN $COMMON $TYPEDTREE        $TAKEONE  $BYNONTYPES -name $RUN-typed-takeOneButType-numNonTypes
# $BIN $COMMON $TYPEDBACKOFFTREE $TAKEONE  $BYNONTYPES -name $RUN-typed-tooFewRecs-takeOneButType-numNonTypes

# $BIN $COMMON $STDTREE          $TAKEITER $BYNONTYPES -name $RUN-standard-takeMoreButCommon-numNonTypes
# $BIN $COMMON $TYPEDTREE        $TAKEITER $BYNONTYPES -name $RUN-typed-takeMoreButCommon-numNonTypes
# $BIN $COMMON $TYPEDBACKOFFTREE $TAKEITER $BYNONTYPES -name $RUN-typed-tooFewRecs-takeMoreButCommon-numNonTypes


# # Make a zip of everything
# (cd $HOME/data && tar cvzf $TESTBASE-evals-$RUN-everything.tar.gz $TESTBASE-$RUN-*)

