# Glossary Module

The Glossary Module is responsible for mapping the property URLs with their labels and descriptions
for each corresponding language.

It works by generating it's own mapping structure from a dataset that only contains properties. That
structure is stored and can later be re-used when the recommender is serving.

It also contains utilities for filtering lists of properties by certain glossary values.


## Properties that are used

Labels and descriptions are just a type of property that has special handling by wikidata. They are used
in the same way for both the Items and Properties.

* Labels: 
    * `<http://schema.org/name>` (used by glossary)
    * `<http://www.w3.org/2000/01/rdf-schema#label>`
    * `<http://www.w3.org/2004/02/skos/core#prefLabel>`
    * `<http://www.w3.org/2004/02/skos/core#altLabel>` (alternative name)
* Descriptions: 
    * `<http://schema.org/description>` (used by glossary)

Each value is given in the form of `"text"@language`. 
One example for Belgium in british english is: `"Belgium"@en-gb`

## Notes about Glossary usage

The Glossary is typically only used for properties only, but nothing prevents you from generating a glossary for items as well, though this will not be used by the server.

## TODO

* Glossary is using hard-coded properties, but they should ideally come from a configuration file
that is specific to wikidata.