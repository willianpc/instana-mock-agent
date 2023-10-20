# Instana Mock Agent

The idea is to provide a mocked agent that offers the following endpoints:

1. Discovery. Eg: `/com.instana.plugin.golang.discovery`
1. Traces. Eg: `/com.instana.plugin.golang/traces.12345`

### Useful cURL tests

      $  curl -D- -X -POST -d "$(cat ./fixtures/span.json)" "http://localhost:9090/com.instana.plugin.nodejs/traces.12345"
---
      $  curl -D- -X -POST -d "$(cat ./fixtures/discovery_req.json)" "http://localhost:9090/com.instana.plugin.nodejs.discovery"


## Environment Variables

### MOCK_AGENT_PORT

Default is `42698`. Real Agent default port is `42699`



