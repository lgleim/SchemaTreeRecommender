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

### /lean-recommender

Recommendation endpoint following the initial method.