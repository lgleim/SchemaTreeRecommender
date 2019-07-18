# Server Module

The Server Module will serve as thin layer of communication between the outside world and the
recomender. It sets up an API using a HTTP server for basic communication with JSON.

## Endpoints

### /recommender

Input JSON-Schema:

```json
    {
    	"title": "SchemaTree Recommendation Request",
    	"type": "object",
    	"properties": {
    		"lang": {
    			"type": "string"
    		},
    		"types": {
    			"type" : "array",
    			"items" : {
    				"type": "string"
    			}
    		},
    		"properties": {
    			"type" : "array",
    			"items" : {
    				"type": "string"
    			}
    		}
    	},
    	"required": ["lang","types","properties"]
    }
```

Example input: 

```json
{
  "lang": "en",
  "properties": [
    "http://www.wikidata.org/prop/direct/P31",
    "http://www.wikidata.org/prop/direct/P1476",
    "http://www.wikidata.org/prop/direct/P433"
  ],
  "types": [
    "http://www.wikidata.org/entity/Q13442814"
  ]
}
```

To make the typed SchemaTree more compatible with type-unaware clients, the P31 (instanceOf) property should also be allowed to exist in the `$.properties` attribute.

Output JSON-Schema:

```json
    {
    	"title": "SchemaTree Recommendation Response",
    	"type": "object",
    	"properties": {
			"recommendations": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"property": { "type": "string" },
						"label": { "type": "string" },
						"description": { "type": "string" },
						"probability": { "type": "number" }
					},
    				"required": ["property", "label", "description", "probability"]
				}
			}
		},
    	"required": ["recommendations"]
	}
```

Example Output:

```json
{
  "recommendations": [
    {
			"property": "http://www.wikidata.org/prop/direct/P828",
			"probability": 0.964267264,
			"label": "has cause",
			"description": "underlying cause, thing that ultimately resulted in this effect"
		},
    {
			"property": "http://www.wikidata.org/prop/direct/P1889",
			"probability": 0.874251324,
			"label": "different from",
			"description": "item that is different from another item, with which it is often confused"
		}
	]
}
```

### /lean-recommender

Recommendation endpoint following the initial method.