# playerpath

[![ci](https://img.shields.io/github/actions/workflow/status/cetteup/playerpath/ci.yaml?label=ci)](https://github.com/cetteup/playerpath/actions?query=workflow%3Aci)
[![Go Report Card](https://goreportcard.com/badge/github.com/cetteup/playerpath)](https://goreportcard.com/report/github.com/cetteup/playerpath)
[![License](https://img.shields.io/github/license/cetteup/playerpath)](/LICENSE)
[![Last commit](https://img.shields.io/github/last-commit/cetteup/playerpath)](https://github.com/cetteup/playerpath/commits/main)

Proxy that forwards Battlefield 2 stats requests, choosing the correct provider on a per-player basis. Evolved from [unlockproxy](https://github.com/cetteup/unlockproxy).

## Usage

### Running playerpath

You can run playerpath on Windows or Linux, either directly or via Docker. Please note that playerpath usually cannot run on the same host as the game server.

On any Linux using systemd, you can run playerpath as a simple service.

```ini
[Unit]
Description=Battlefield 2 aspx proxy
After=network-online.target

[Install]
WantedBy=multi-user.target

[Service]
Type=simple

WorkingDirectory=/opt/playerpath
ExecStart=/opt/playerpath/playerpath

Restart=always
RestartSec=1m

StandardOutput=syslog
StandardError=syslog
SyslogIdentifier=%n

User=playerpath
```

### Using a reverse proxy

When running behind a reverse proxy such as NGiNX, the proxy needs to be configured to ignore the client closing the connection. Else certain endpoints such as BF2Hub's `getrankstatus.aspx` will not work correctly.

For NGiNX, simply add the following to the relevant http/server/location configuration.

```
proxy_ignore_client_abort on;
```

### Redirecting ASPX HTTP traffic to playerpath

To make your Battlefield 2 server use playerpath, you need to redirect all HTTP traffic for the game's ASPX endpoints to playerpath. On Linux, you can use `iptables` to achieve this. Using the IP addresses behind `servers.bf2hub.com` as an example, you could use (as root/with `sudo`):

```sh
iptables -t nat -A OUTPUT -p tcp -d 116.203.14.166 --dport 80 -j DNAT --to-destination 192.168.100.50:8080
```

This rule will redirect any HTTP traffic designated to BF2Hub's servers to an instance of playerpath running on `192.168.100.50`. Make sure you have a setup to persist `iptables` rules, otherwise they will lost on reboot.

## How it works

When a player joins a Battlefield 2 server, the server makes several HTTP request to the statistics backend to determine the player's rank, unlocked weapons and awards. This works fine if player and server use the same statistics backend. However, the requests usually won't return any data if e.g. a PlayBF2 player joins a BF2Hub server. Since the BF2Bub backend does not know the PlayBF2 player, it returns no usable information and the player is treated as a private with no unlocks.

playerpath solves this by dynamically forwarding the requests to the player's provider. The respective provider is determined based on data from [bf2opendata](https://github.com/art567/bf2opendata), which contains player information from all major Battlefield 2 providers (currently BF2Hub, PlayBF2, OpenSpy and B2BF2). Thanks to this additional information, the requests which would have been sent to BF2Hub are sent to PlayBF2 instead, which is able to provide the required details for the player.

While playerpath enables servers to _retrieve_ player information, it does **not** support sending post-round statistics snapshots to multiple providers. Snapshots are only forwarded to the configured default provider.

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://github.com/user-attachments/assets/d2a01453-55d7-4679-b7f5-cb4c31be74f5">
  <source media="(prefers-color-scheme: light)" srcset="https://github.com/user-attachments/assets/8d25686d-d986-41f7-a965-51cdb4330187">
  <img src="https://github.com/user-attachments/assets/8d25686d-d986-41f7-a965-51cdb4330187">
</picture>


