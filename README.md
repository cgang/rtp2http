# A simple RTP/UDP to HTTP relay server

This is a very simple RTP/UDP to HTTP relay server written in pure golang,
it provides similar function and API like udpxy, such as:
```
http://<ip>:<port>/udp/[multicast_address]
http://<ip>:<port>/rtp/[multicast_address]
```

The memory footprint (Resident Set Size) is under 10MB since the IO buffer is reused.
Only MPEG-TS (UDP or RTP) is supported right now.
This program is intended to run on Linux platform only.

## Options
There are some options for this program.
- `-m <interface>` required parameter to specify the multicast interface.
- `-a <address>` optional parameter to specify HTTP listen address, default to 127.0.0.1
- `-p <port>` optional parameter to specify HTTP listen port, default to 4022

## TODOs
- Support for other stream type
- Support the status page
- Support for IPv6
- Support for non Linux platform
- Daemonize this program? maybe, since we can use systemd

## Non Goals
- Support for HTTPS, since this is not need at home

## License
This program is released under MIT license, see LICENSE for detail.
