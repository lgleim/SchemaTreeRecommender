## Downloaded at version: 1d25e76f894796bfd57dd107102cf39088885138
## from https://github.com/wikimedia/wikibase-property-suggester-scripts


[![Build Status](https://travis-ci.org/wikimedia/wikibase-property-suggester-scripts.svg?branch=master)](https://travis-ci.org/wikimedia/wikibase-property-suggester-scripts)
[![Coverage Status](https://coveralls.io/repos/github/wikimedia/wikibase-property-suggester-scripts/badge.svg?branch=master)](https://coveralls.io/github/wikimedia/wikibase-property-suggester-scripts?branch=master)

# PropertySuggester Scripts
Contains scripts for PropertySuggester to preprocess the wikidata dump

## Install
Run the command:
```
sudo apt-get install build-essential python-pip python-dev
python setup.py install
```
## Usage 
- use dumpconverter.py to convert a wikidata JSON dump to csv (this can be obtained using extensions/Wikibase/repo/maintenance/dumpJson.php)
- use analyzer.py to create a csv file with the suggestion data that can be loaded into a sql table
- the PropertySuggester extension provides a maintenance script (maintenance/UpdateTable.php) that allows to load the csv into the database

```
python scripts/dumpconverter.py latest-all.json.bz2 dump.csv
python scripts/analyzer.py dump.csv wbs_propertypairs.csv
php extensions/PropertySuggester/maintenance/UpdateTable.php --file wbs_propertypairs.csv
```

### Run tests
```
pytest .
```

## Release Notes

### 3.0.0
* Restructure repository
* Using pytest instead of nosetests

### 2.0.0
* Consider classifying Properties
* use Json dumps for analysis

### 1.1
* Generate associationrules for qualifier and references
* Improve ranking to avoid suggestions of human properties
* remove very unlikely rules (<1%)

### 1.0
* Converts a wikidata dump to a csv file with associationrules between properties
