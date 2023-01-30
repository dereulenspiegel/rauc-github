#!/usr/bin/env python3

from pydbus import SessionBus
from gi.repository import GLib
import time

loop = GLib.MainLoop()

bus = SessionBus()

update_manager = bus.get(
    "com.github.dereulenspiegel.rauc", # Bus name
    "/com/github/dereulenspiegel/rauc" # Object path
)

signal_received = False

def signal_callback(signal):
    global signal_received
    signal_received = True
    print(signal)

update_manager.UpdateAvailable.connect(signal_callback)
loop.run()
time.sleep(1)

if signal_received:
    exit(0)
else:
    exit(1)
