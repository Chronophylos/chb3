# CHB3 - ChronophylosBot 3

## Installation

```
git clone https://github.com/Chronophylos/chb3
cd chb3
sudo make install
sudo systemctl enable --now chb3
```

## Configuration

Config should be in `/etc/chb3/config.toml`

Example config:

```toml
[twitch]
username = "your twitch username"
token = "oauth:the token"

[imgur]
clientid = "the client id for imgur"
```
