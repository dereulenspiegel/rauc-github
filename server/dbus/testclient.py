#!/usr/bin/env python3

from pydbus import SessionBus
bus = SessionBus()

update_manager = bus.get(
    "com.github.dereulenspiegel.rauc", # Bus name
    "/com/github/dereulenspiegel/rauc" # Object path
)

next_update = update_manager.NextUpdate()
print(next_update)
if next_update["name"] != "Penguin":
    exit(1)
if next_update["version"] != "1.8.2":
    exit(1)
