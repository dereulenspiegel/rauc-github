#!/usr/bin/env python3

import json

import requests_unixsocket

session = requests_unixsocket.Session()

r = session.get('http+unix://update.socket/update/status')
status = r.json()
print(json.dumps(status, indent=4))

assert status['status'] == "idle"

