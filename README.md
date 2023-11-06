# Instana Mock Agent

The idea is to provide a manager that spawns mocked agents.

Each of these mocked agents provide the following endpoints:

1. Agent ping. Eg: `/com.instana.plugin.golang.12345`
1. Discovery. Eg: `/com.instana.plugin.golang.discovery`
1. Traces. Eg: `/com.instana.plugin.golang/traces.12345`

## How to use it

We assume default port 9090, but you can change it with the env var `MOCK_AGENT_PORT`

1. Start the app
1. To spawn a new mocked agent call `http://localhost:9090/spawn`. The body response and the header `X-MOCK-AGENT-PORT` will give the port number of the spawned agent.
1. If you want to specify a port, just call `http://localhost:9090/spawn/{SPECIFIC_PORT}`
1. Run the instrumented application normally, but remember to set the agent port to the new one
1. If you want to see the spans that were sent to the agent, call `http://localhost:AGENT-PORT/dump`
1. To kill a spawned mocked agent call `http://localhost:9090/kill/AGENT-PORT`
1. If the mocked agent manager dies, all spawns die with it

### Useful cURL tests

We assume the first auto generated port `29091`, but you can specify a port by calling `http://localhost:9090/spawn/{MY_PORT}`

      $  curl -D- -X -POST -d "$(cat ./fixtures/span.json)" "http://localhost:29091/com.instana.plugin.nodejs/traces.12345"
---
      $  curl -D- -X -POST -d "$(cat ./fixtures/discovery_req.json)" "http://localhost:29091/com.instana.plugin.nodejs.discovery"
---
      $  curl -D- "http://localhost:29091/com.instana.plugin.golang.12345"


## Environment Variables

### MOCK_AGENT_PORT

Will start the agent manager with the given port.

> [!NOTE]
> This is a pet project and should be treated as one.
