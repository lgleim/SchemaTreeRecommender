#!/usr/bin/env bash

VERSION="20190729"
BASE="wikidata-$VERSION-all"
DUMP_PREFIX="https://dumps.wikimedia.org/wikidatawiki/entities/$VERSION/$BASE"
REC="../recommender"
DOWNLOAD=false
TRAIN=false
N=${1:-10000}

if [ "$DOWNLOAD" = true ]; then
    wget ${DUMP_PREFIX}.json.bz2
    wget ${DUMP_PREFIX}.nt.gz
fi

if [ "$TRAIN" = true ]; then
    # prepare wikidata PropertySuggester
    # sudo apt-get install build-essential python-pip python-dev
    python property-suggester-scripts/setup.py install
    
    # convert wikidata JSON dump to csv
    python scripts/dumpconverter.py latest-all.json.bz2 dump.csv
    # create a csv file with the association rules for the suggester that can be loaded into a sql table
    python scripts/analyzer.py dump.csv wbs_propertypairs.csv

    # start wikibase services
    docker-compose up -d
    # NOTE: It may be that startup fails. In that case, comment out the line containing "LocalSettings.php" in the docker-compose.yml file, rerun `docker-compose up`, stop it again, comment the line back in and everything should work again.

    # install composer
#    docker exec -it wikibase_wikibase_1  sh -c "curl --silent https://getcomposer.org/installer | php --"
#    docker exec -it -w /var/www/html/extensions/PropertySuggester/ wikibase_wikibase_1 php ../../composer.phar dump-autoload
    docker exec -it wikibase_wikibase_1  php maintenance/update.php
    # load the csv into the database
    gunzip -c wbs_propertypairs.csv.gz > wbs_propertypairs.scv
    docker exec -it wikibase_wikibase_1 php extensions/PropertySuggester/maintenance/UpdateTable.php --file ../wbs_propertypairs.csv

    # prepare SchemaTree recommender
    $REC split-dataset by-prefix wikidata-$VERSION-all.nt.gz
    $REC filter-dataset for-glossary wikidata-$VERSION-all-prop.nt.gz
    $REC build-glossary wikidata-$VERSION-all-prop-filtered.nt.gz > build-glossary.log
    $REC filter-dataset for-schematree wikidata-$VERSION-all-item.nt.gz
    gzip -cd wikidata-$VERSION-all-item-filtered.nt.gz | sort | gzip > wikidata-$VERSION-all-item-filtered-sorted.nt.gz

    $REC split-dataset 1-in-n wikidata-$VERSION-all-item-filtered-sorted.nt.gz -n $N
    $REC build-tree wikidata-$VERSION-all-item-filtered-sorted-1in$N-train.nt.gz -f
    $REC build-tree-typed wikidata-$VERSION-all-item-filtered-sorted-1in$N-train.nt.gz -f

    # count the occurrence frequency of each subject set size. columns: (freq.) \t (numNonTypes)
    zcat wikidata-$VERSION-all-item-filtered-sorted-1in$N-test.nt.gz \
        | grep -v "<http://www.w3.org/1999/02/22-rdf-syntax-ns#type>" \
        | grep -v "/P31>" \
        | grep -v "<http://dbpedia.org/ontology/type>" \
        | cut -d " " -f 1| sort | uniq -c | cut -d "<" -f 1 | sort | uniq -c \
        > wiki_counts.csv
fi

# Change this to zip all final files into another zip
RUN="r1"
EVALBASE="$BASE-item-filtered-sorted-1in$N"
BIN="./evaluation"
COMMON="-results -testSet $EVALBASE-test.nt.gz"
go build .

STDTREE="-model $EVALBASE-train.nt.gz.schemaTree.bin"
TYPEDTREE="-model $EVALBASE-train.nt.gz.schemaTree.typed.bin -typed"
TYPEDBACKOFFTREE="-model $EVALBASE-train.nt.gz.schemaTree.typed.bin -typed -workflow Wiki_backoff.json"
WIKIDATA="-model $EVALBASE-train.nt.gz.schemaTree.typed.bin -wikiEvaluation -typed"
WIKIDATA2="-model $EVALBASE-train.nt.gz.schemaTree.bin -wikiEvaluation"

TAKEONE="-handler takeOneButType"
# TAKEMANY="-handler takeAllButBest"
TAKEITER="-handler takeMoreButCommon"

BYSETSIZE="-groupBy setSize"
BYLEFTOUT="-groupBy numLeftOut"
BYNONTYPES="-groupBy numNonTypes"

# Makes no sense for the evaluation
# $BIN $COMMON $STDTREE          $TAKEONE  $BYSETSIZE  -name $RUN-standard-takeOneButType-setSize
# $BIN $COMMON $TYPEDTREE        $TAKEONE  $BYSETSIZE  -name $RUN-typed-takeOneButType-setSize
# $BIN $COMMON $TYPEDBACKOFFTREE $TAKEONE  $BYSETSIZE  -name $RUN-typed-tooFewRecs-takeOneButType-setSize
# $BIN $COMMON $WIKIDATA         $TAKEONE  $BYSETSIZE  -name $RUN-wikidata-takeOneButType-setSize

# $BIN $COMMON $STDTREE          $TAKEONE  $BYNONTYPES  -name $RUN-standard-takeOneButType-numNonTypes
# $BIN $COMMON $TYPEDTREE        $TAKEONE  $BYNONTYPES  -name $RUN-typed-takeOneButType-numNonTypes
# $BIN $COMMON $TYPEDBACKOFFTREE $TAKEONE  $BYNONTYPES  -name $RUN-typed-tooFewRecs-takeOneButType-numNonTypes
# $BIN $COMMON $WIKIDATA         $TAKEONE  $BYNONTYPES  -name $RUN-wikidata-takeOneButType-numNonTypes
# $BIN $COMMON $WIKIDATA2         $TAKEONE  $BYNONTYPES  -name $RUN-wikidata2-takeOneButType-numNonTypes

$BIN $COMMON $STDTREE          $TAKEITER  $BYNONTYPES  -name $RUN-standard-takeMoreButCommon-numNonTypes
$BIN $COMMON $TYPEDTREE        $TAKEITER  $BYNONTYPES  -name $RUN-typed-takeMoreButCommon-numNonTypes
$BIN $COMMON $TYPEDBACKOFFTREE $TAKEITER  $BYNONTYPES  -name $RUN-typed-tooFewRecs-takeMoreButCommon-numNonTypes
$BIN $COMMON $WIKIDATA         $TAKEITER  $BYNONTYPES  -name $RUN-wikidata-takeMoreButCommon-numNonTypes
$BIN $COMMON $WIKIDATA2         $TAKEITER  $BYNONTYPES  -name $RUN-wikidata2-takeMoreButCommon-numNonTypes

# $BIN $COMMON $STDTREE          $TAKEMANY $BYNONTYPES -name $RUN-standard-takeAllButBest-numNonTypes
# $BIN $COMMON $TYPEDTREE        $TAKEMANY $BYNONTYPES -name $RUN-typed-takeAllButBest-numNonTypes
# $BIN $COMMON $TYPEDBACKOFFTREE $TAKEMANY $BYNONTYPES -name $RUN-typed-tooFewRecs-takeAllButBest-numNonTypes
# $BIN $COMMON $WIKIDATA         $TAKEMANY $BYNONTYPES -name $RUN-wikidata-takeAllButBest-numNonTypes

# $BIN $COMMON $STDTREE          $TAKEMANY $BYLEFTOUT  -name $RUN-standard-takeAllButBest-numLeftOut
# $BIN $COMMON $TYPEDTREE        $TAKEMANY $BYLEFTOUT  -name $RUN-typed-takeAllButBest-numLeftOut
# $BIN $COMMON $TYPEDBACKOFFTREE $TAKEMANY $BYLEFTOUT  -name $RUN-typed-tooFewRecs-takeAllButBest-numLeftOut
# $BIN $COMMON $WIKIDATA         $TAKEMANY $BYLEFTOUT  -name $RUN-wikidata-takeAllButBest-numLeftOut


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
