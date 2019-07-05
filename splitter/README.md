# Splitter Module

The Splitter Module is capable of splitting datasets in N-Triples format into multiple files using several methods.

Methods:

* **1 in N:** Used for splitting datasets into training and test sets. It will filter out every Nth line into a new file.

* **items and properties:** Takes a dataset a generates 3 files. One for all items, one for all properties, and one for entries which are neither of the two.


## Identifying items and properties

The wikidata dump has all subjects together, both items and properties. To identify whether a subject
is an item or a property we need to check the object of a specific predicate.

*Reminder, the N-Triples files comes in lines of `subject predicate object .`*

* Predicate: <http://www.w3.org/1999/02/22-rdf-syntax-ns#type>
* If item, then object is: <http://wikiba.se/ontology-beta#Item>
* If property, then object is: <http://www.wikidata.org/ontology#Property>

Another simpler, but not so pedantic way, would be to check if the subject start with a prefix. This is hypothetical and not actually used.

* For entities: <http://www.wikidata.org/entity/Q   >
* For properties: <http://www.wikidata.org/entity/P   >