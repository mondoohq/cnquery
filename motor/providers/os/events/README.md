# Motor Events

Motor events implement a watcher on file and commands, so that the user of the library has the latest information available.

It implements that by two different mechanism:
- polling
- pushing

By default, we assume polling since it works across all motor transports, while pushing is a special optimization for specific transports. Motor Events abstracts that mechanism away, therefore users do not need to take care about.

               ┌────────────────┐
               │    Watcher     │
               └────────────────┘
                        │
        ┌───────────────┴──────────────┐
        ▼                              ▼
┌───────────────┐              ┌───────────────┐
│  eg. iNotify  │              │    polling    │
└───────────────┘              └───────────────┘
        │                              │
        │                              ▼
        │                      ┌───────────────┐
        │                      │   runnable    │
        │                      └───────────────┘
        │                              │
        │                              ▼
        │                      ┌───────────────┐
        │                      │     diff      │
        │                      └───────────────┘
        │                              │
        │      ┌────────────────┐      │
        └─────▶│    update()    │◀─────┘
               └────────────────┘