# Preparation Module

The Preparation Module is capable of preparing datasets in N-Triples format.

It is able to split datasets into multiple files, or to filter a dataset and only output the filtered entries.
For each operation a method can be chosen that alters how the split and filter is executed.

## Splitting Methods

* **1-in-n:** Used for splitting datasets into training and test sets. It will filter out every Nth line into a new file.

* **by-type:** Takes a dataset and generates 3 files. One for all items, one for all properties, and one for entries which are neither of the two. **Note** that this splitter needs to assume that all subjects come in contiguous lines. In other words, the dataset has to be grouped by the subject column.

* **by-prefix:** Takes a dataset and generates 3 files. Split is made according to the prefix of the subject. 

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

* For entities: `<http://www.wikidata.org/entity/Q`
* For properties: `<http://www.wikidata.org/entity/P`


## Prefix mismatch on properties

Wikidata uses (at least) two different URL prefixes to refer to the properties, and this creates an incompatibility on the glossary which needs to be fixed with an extra preparation step on the property dataset.

When an Item subject refers to a Property predicate, Wikidata will use `<http://www.wikidata.org/prop/direct/Pxxx>` to refer to the property, but when Wikidata is defining the Property (in other words, when Property is used as a subject), Wikidata will refer to it with `<http://www.wikidata.org/entity/Pxxx>`. Notice the mismatch between `/prop/direct/` and `/entity/`.

Without a proper preparation step, this mismatch will cause the glossary to store all labels using the `/entity/` key, while the server requests will actually try to fetch `/prop/direct/` keys from the glossary, resulting in showing no labels at all.

These two different url prefixes are described in the data by using a specific predicate. An example is:

    <http://www.wikidata.org/entity/Pxxx> <http://wikiba.se/ontology#directClaim> <http://www.wikidata.org/prop/direct/Pxxx> .

The current extra preparation step makes a simple prefix change, but assumes a specific URL is used. It is not pedantic.

```bash
gzip -cd dataset.nt.gz | sed -r -e 's|^<http:\/\/www\.wikidata\.org\/entity\/P([^>]+)>|<http://www.wikidata.org/prop/direct/P\1>|g' | gzip > ./dataset-altered.nt.gz
```

## Requirement of contiguous subject entries

Some splitters that work in block of entries require that all subjects have their definitions in contiguous lines. To guarantee this requirement you can add an extra preparation step to sort the dataset.

```bash
gzip -cd ./dataset-filtered.nt.gz | sort | gzip > dataset-filtered-sorted.nt.gz
```
