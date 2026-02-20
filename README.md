# Configuration example
```toml
debug = true
interval = 120
looptest_match_suffix = "match-suffix.example.com"

[nats]
url = "nats://localhost:4222"
event_subject = "internal.events.new_qname"
observation_subject_prefix = "internal.observations"
looptest_bucket = "looptest"

[cert]
active = false
debug = false

[api]
active = false
debug = false

[libtapir]
debug = true
```
