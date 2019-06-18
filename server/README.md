# Server Module

The Server Module will serve as thin layer of communication between the outside world and the
recomender. It sets up an API using a HTTP server for basic communication with JSON.

TODO: Right now the entire application is called using the "treebuilder" package. A next step
would be to make this "server" package the main entrypoint and have it use the treebuilder, strategy
and schematree according to its need. Loading and tree construction would be deferred to
"treebuilder", and strategy customization would be handled by the "server" itself.

TODO: Call it server or api?