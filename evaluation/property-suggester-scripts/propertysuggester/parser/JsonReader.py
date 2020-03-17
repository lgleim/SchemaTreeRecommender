"""
read_json returns a generator that yields Entities)

usage:
with open("file.csv", "r") as f:
    for entity in read_json(f):
        do_things()

"""
import logging
from propertysuggester.utils.datamodel import Claim, Entity, Snak
import sys

try:
    import ujson as json
except ImportError:
    logging.info("ujson not found")
    import json as json

data_types = [
    "string",
    "commonsMedia",
    "geo-shape",
    "url",
    "external-id",
    "math",
    "tabular-data",
    "musical-notation"
]


def compatible_str(term):
    # TODO: Remove this when migrated to python3
    if isinstance(term, str):
        return term
    if sys.version_info < (3,):
        return str(term.encode('utf-8'))
    else:
        return str(term, 'utf-8')


def read_json(input_file):
    """
    @rtype : collections.Iterable[Entity]
    @type input_file:  file or GzipFile or StringIO.StringIO
    """
    count = 0
    for jsonline in input_file:
        count += 1
        jsonline = compatible_str(jsonline)
        if count % 3000 == 0:
            logging.info(
                "processed %.2fMB" %
                (input_file.tell() / 1024.0 ** 2))
        if jsonline[0] == "{":
            jsonline = jsonline.rstrip(",\r\n")
            data = json.loads(jsonline)
            if data["type"] == "item":
                yield _process_json(data)


def _process_json(data):
    title = data["id"]
    if "claims" not in data:
        return Entity(title, [])
    claims = []
    for property_id, statements in sorted(list(data["claims"].items())):
        for statement in statements:
            references = []
            if "references" in statement:
                # TODO: group reference snaks correctly
                for reference in statement["references"]:
                    if not reference["snaks"]:
                        continue
                    for ref_id, snaks in sorted(
                            list(reference["snaks"].items())):
                        for snak in snaks:
                            ref = _parse_json_snak(snak)
                            if ref:
                                references.append(ref)
            qualifiers = []
            if "qualifiers" in statement:
                for qual_id, snaks in sorted(
                        list(statement["qualifiers"].items())):
                    for snak in snaks:
                        qualifier = _parse_json_snak(snak)
                        if qualifier:
                            qualifiers.append(qualifier)
            claim = _parse_json_snak(statement["mainsnak"])
            if claim:
                claims.append(Claim(claim, qualifiers, references))

    return Entity(title, claims)


def _parse_json_snak(claim_json):
    if claim_json["snaktype"] == "value":
        if 'datatype' not in claim_json:
            logging.warning(
                "encountered snak without datatype: " + str(claim_json))
            return None
        datatype = claim_json["datatype"]
        datavalue = claim_json["datavalue"]["value"]

        try:
            if datatype in data_types:
                value = datavalue
            elif datatype == "wikibase-item":
                if datavalue["entity-type"] == "item":
                    value = datavalue["id"]
                else:
                    logging.warning(
                        "unknown entitytype: {0}".format(
                            datavalue["entity-type"]))
            elif datatype == "wikibase-property":
                if datavalue["entity-type"] == "property":
                    value = datavalue["id"]
                else:
                    logging.warning(
                        "unknown entitytype: {0}".format(
                            datavalue["entity-type"]))
            elif datatype == "wikibase-lexeme":
                if datavalue["entity-type"] == "lexeme":
                    value = datavalue["id"]
                else:
                    logging.warning(
                        "unknown entitytype: {0}".format(
                            datavalue["entity-type"]))
            elif datatype == "wikibase-sense":
                if datavalue["entity-type"] == "sense":
                    value = datavalue["id"]
                else:
                    logging.warning(
                        "unknown entitytype: {0}".format(
                            datavalue["entity-type"]))
            elif datatype == "wikibase-form":
                if datavalue["entity-type"] == "form":
                    value = datavalue["id"]
                else:
                    logging.warning(
                        "unknown entitytype: {0}".format(
                            datavalue["entity-type"]))
            elif datatype == "time":
                value = datavalue["time"]
            elif datatype == "quantity":
                value = datavalue["amount"]
            elif datatype == "globe-coordinate":
                value = "N{0[latitude]}, E{0[longitude]}".format(datavalue)
            elif datatype == "monolingualtext":
                value = u"{0[text]} ({0[language]})".format(datavalue)
            elif datatype == "bad":
                # for example in Q2241
                return None
            else:
                logging.warning("unknown wikidata datatype: %s" % datatype)
                return None
        except KeyError as e:
            logging.warning(
                "unrecognized or mismatching datavalue for datatype: %s" %
                e)
            return None
        except TypeError as e:
            logging.warning("type error: %s" % e)
            return None
    else:  # novalue, somevalue, ...
        datatype = "unknown"
        value = claim_json["snaktype"]
    property_id = int(claim_json["property"][1:])
    return Snak(property_id, datatype, value)
