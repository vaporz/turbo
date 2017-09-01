.. _rules:

Rules and Conventions
=====================

There are some rules when you use turbo.

*    When defining a gRPC service, if the name of a gRPC method is "methodName", then the name of request message and response message **MUST** be "MethodNameRequest" and "MethodNameResponse".
*    When defining a thrift service, the response message's name **MUST** be "MethodNameResponse".
*    If multiple paths are assigned to $GOPATH(divided by ':'), then the first path is used by turbo as GOPATH.
*    If the value to a param in a request presents a list, it **MUST** be seperated by ",".
*    When parsing request parameters, values from URL path has a higher priority than those from query string, body or context.Context.
    e.g. In a request like "GET /book/1234?id=5678", both "1234" and "5678" are values to "id", but "1234" is picked as value to key "id".

*    The value of a key with all lower case characters has a higher priority to the value of a key with upper case characters.
    e.g. In a request like "GET /book?id=1234&ID=5678", "1234" is used for key "id".

*    A parameter's key is case-insensitive to turbo, in fact, internally turbo will cast keys to lower case characters before further use.
    e.g. In a request like "GET /book?ID=1234", turbo will see this query string as "id=1234".
