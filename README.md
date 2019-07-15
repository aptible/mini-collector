# mega-collector

mega-collector tails log files, and ships those log files to a centralized destination.
The destinations currently supported are:

* Nowhere (aka blackhole)
* A file

### mega-collector anatomy

mega-collector has two main components: a collector and an aggregator. The collector tails
the specified log files and ships the contents of the logs files to the aggregator. The aggregator 
then ships the those logs to a single specified destination in batches. 

## Aggregator

The entry point for the aggregator is defined in `cmd/aggregator/main.go`. 
The entry point runs a server (specified in `api/api.pb.go`) that is configured to receive data that
is in a specific format (defined in `api/api.proto`). 
The code for this server is auto-generated based on the values in `api/api.proto`. 

The messages the aggregator receives are grouped into batches. There can be at most `AGGREGATOR_MAX_BATCH_SIZE`
messages in each batch. 
Once a batch has `AGGREGATOR_MAX_BATCH_SIZE` messages, or `AGGREGATOR_MINIMUM_PUBLISH_FREQUENCY` time has passed,
the batcher pushes the batch to the destination.


TODO: Add something about the ingestBuffer here

Pushing to a destination is done via an emitter. The emitter used is determined in the `getEmitter` function
in `cmd/aggregator/main.go` by the presence of a configuration environment variable.
  
(This is not implemented yet, because it's irrelevant for the text output) On failure to push to the destination,
the emitter will retry X times. After X failures, the data is discarded and a PagerDuty alert is generated.
(see https://github.com/aptible/mini-collector/blob/master/cmd/aggregator/main.go#L180-L183) 




### Aggregator environment variables

|  Variable |  Default | Format | Description |
|-----------|----------|--------|-------|
| `AGGREGATOR_NOTIFY_CONFIGURATION` | n/a | See [`AGGREGATOR_NOTIFY_CONFIGURATION`](#AGGREGATOR_NOTIFY_CONFIGURATION) | Specifies how we notify PagerDuty of a failure  |
| `AGGREGATOR_MINIMUM_PUBLISH_FREQUENCY` | 15s | should be parsable by `ParseDuration`: https://golang.org/pkg/time/#ParseDuration | | 
| `AGGREGATOR_TLS` |  0 | 0 or 1 (false/true) | Indicates whether or not the aggregator should use TLS to communicate with the collectors |
| `AGGREGATOR_TLS_CERTIFICATE` | n/a | String | |
| `AGGREGATOR_TLS_KEY` | n/a | String | |
| `AGGREGATOR_TLS_CA_CERTIFICATE` | n/a | String | |
| `AGGREGATOR_MAX_BATCH_SIZE` | 1000 | integer | The maximum number of log entries that can be emitted in a single group |
| `AGGREGATOR_TEXT_CONFIGURATION`  | n/a  | Any truthy value works | The presence of this env variable indicates that logs should be set to stdout |
|   |   |   |  |
|   |   |   |  |




## Configuration formats

### `AGGREGATOR_NOTIFY_CONFIGURATION`

```json
{
  "integration_key": "String; The PagerDuty integration key",
  "incident_key": "String; A unique identifier for this drain, e.g. stack/log_drain/ID/notify",
  "identifier": "String; A human-readable way to identify this drain, e.g. Log Drain #ID (handle)"
}
```
