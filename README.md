# docdiff
Tool to check that documents end up with same fields in elasticsearch

## Approach

### Setup
1. ingest documents into elasticsearch with method 1 (example elastic-agent with elasticsearch output), required that documents contain a unique field
2. ingest documents into elasticsearch again with method 2 (example elastic-agent with shipper), required that same source event will produce same unique field as in method 1

### docdiff
1. query elasticsearch with field name & unique value
2. query should return 2 results
3. flatten responses (change nesting to dot notation)
4. remove fields we expect to have changed (example, agent.id or @timestamp)
5. diff responses
