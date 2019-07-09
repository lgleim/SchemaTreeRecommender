# Preparation Module

The Preparation Module is capable of preparing datasets in N-Triples format.

It is able to split datasets into multiple files, or to filter a dataset and only output the filtered entries.
For each operation a method can be chosen that alters how the split and filter is executed.

## Splitting Methods

* **1-in-n:** Used for splitting datasets into training and test sets. It will filter out every Nth line into a new file.

* **by-type:** Takes a dataset a generates 3 files. One for all items, one for all properties, and one for entries which are neither of the two. This splitter needs to assume that all subjects come in contiguous lines. In other words, the dataset has to be grouped by the subject column.

## Filtering Methods

* **for-schematree:** Filters out entries that are not useful for the schematree build process.

* **for-glossary:** Filters out entries that are not useful for the glossary build process.

* **for-evaluation:** Filters out entries that make the evaluation of a schematree slower without adding information. This is the case when many labels are given, as to prevent the evaluation to iterate through all of the repeated label properties.


## Identifying items and properties

The wikidata dump has all subjects together, both items and properties. To identify whether a subject
is an item or a property we need to check the object of a specific predicate.

*Reminder, the N-Triples files comes in lines of `subject predicate object .`*

* Predicate: `<http://www.w3.org/1999/02/22-rdf-syntax-ns#type>`
* If item, then object is: `<http://wikiba.se/ontology#Item>`
* If property, then object is: `<http://wikiba.se/ontology#Property>`

In previous datasets (`10M.nt.gz` from June 2019), the items were defined with `<http://wikiba.se/ontology-beta#Item>` and `<http://www.wikidata.org/ontology#Property>`, which is different than those given in the `latest-truthy.nt.gz` (from July 2019)

Another simpler, but not so pedantic way, would be to check if the subject start with a prefix. This is hypothetical and not actually used.

* For entities: <http://www.wikidata.org/entity/Q   >
* For properties: <http://www.wikidata.org/entity/P   >