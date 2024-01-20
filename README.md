# A simple RTP/UDP to HTTP relay server

This is a very simple RTP/UDP to HTTP relay server written in pure golang,
it provides similar function and API like udpxy, such as:
```
http://<ip>:<port>/udp/[multicast_address]
http://<ip>:<port>/rtp/[multicast_address]
```

Initially only MPEG-TS (UDP or RTP) are supported.

## Options
There are some options for this program.
- `-m <interface>` required parameter to specify the multicast interface.
- `-a <address>` optional parameter to specify HTTP listen address, default to 127.0.0.1
- `-p <port>` optional parameter to specify HTTP listen port, default to 4022

## TODOs
- Daemonize this program
- Support for other stream type
- Support the status page? maybe

## Non Goals
- Support for HTTPS, since this is not need at home

## License
This program is released under MIT license, see LICENSE for detail.
