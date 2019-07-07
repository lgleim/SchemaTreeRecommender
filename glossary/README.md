# Glossary Module

The Glossary Module is responsible for mapping the property URLs with their labels and descriptions
for each corresponding language.

It works by generating it's own mapping structure from a dataset that only contains properties. That
structure is stored and can later be re-used when the recommender is serving.

It also contains utilities for filtering lists of properties by certain glossary values.


## Notes about what used properties

Labels and descriptions are just a type of property that has special handling by wikidata.

* for Entities:
    * Labels: 
        * <http://www.w3.org/2000/01/rdf-schema#label>
        * <http://www.w3.org/2004/02/skos/core#prefLabel> (currently not used)
        * <http://schema.org/name> (currently not used)
    * Descriptions: 
        * <http://schema.org/description> 
* for Properties:
    * Labels:
        * <http://www.w3.org/2000/01/rdf-schema#label>
    * Descriptions: 
        * <http://schema.org/description>

Each value is given in the form of `"text"@language`. 
One example for Belgium in british english is: `"Belgium"@en-gb`

## TODO

* Glossary is using hard-coded properties, but they should ideally come from a configuration file
that is specific to wikidata.