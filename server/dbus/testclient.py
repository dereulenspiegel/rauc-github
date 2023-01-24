#!/usr/bin/env python3

from pydbus import SystemBus
bus = SystemBus()

update_manager = bus.get(
    "com.github.dereulenspiegel.rauc", # Bus name
    "/com/github/dereulenspiegel/rauc" # Object path
)

next_update = update_manager.NextUpdate()
print(next_update)
