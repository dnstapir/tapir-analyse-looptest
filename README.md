# Configuration example
```toml
debug = true
interval = 120
looptest_match_suffix = "match-suffix.example.com"

[nats]
debug = true
url = "nats://nats:4222"
event_subject = "internal.events.new_qname"
observation_subject_prefix = "internal.observations"

[[nats.observation_buckets]]
name = "looptest"
observation = "looptest"
ttl = 10
create = false # A "false" setting requires bucket to be pre-provisioned

[api]
active = false
```
