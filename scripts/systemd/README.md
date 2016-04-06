# systemd service


## Installation

Following steps let you set up Request Baskets as systemd service:

 * copy `request-baskets` executable file into `/usr/local/bin/` folder
 * copy `rbaskets.service` unit file into `/etc/systemd/system/` folder
 * create `rbaskets` system user:
 ```
 ~$ sudo adduser --system --no-create-home --group rbaskets
 ```
 * create folder `/var/lib/rbaskets` and assign it to `rbaskets` user:
 ```
 ~$ sudo mkdir -p /var/lib/rbaskets
 ~$ sudo chown rbaskets:rbaskets /var/lib/rbaskets
 ```


## Usage

Start Request Baskets service:
```
~$ sudo systemctl start rbaskets
```

Stop service:
```
~$ sudo systemctl start rbaskets
```

Check service current status:
```
~$ sudo systemctl status rbaskets
```

Display service log:
```
~$ sudo journalctl -u rbaskets
```

Start service on every system restart:
```
~$ sudo systemctl enable rbaskets
```
