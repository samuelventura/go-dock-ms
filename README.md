# go-dock-ms

SSH dock micro service

- Reverse SOCKS proxy only
- TXT record load balancing

# helpers

```bash
#keepalive, mac whitlist
#DOCK_MAXSHIPS=1000
#DOCK_LOGS=/var/log
#DOCK_ENDPOINT=0.0.0.0:31652
#DOCK_DB_DRIVER=sqlite|postgres
#DOCK_DB_SOURCE=<driver dependant>
#DOCK_HOSTNAME=domain.tld
#DOCK_HOSTKEY=path/key.priv
#https://gorm.io/docs/connecting_to_the_database.html
#export DOCK_HOSTKEY=~/.ssh/id_rsa
ssh ship@localhost -p 31652 -N
go install && go-dock-ms
go install && go-dock-ss
#export DOCK_KEYPATH=~/.ssh/id_rsa
#export DOCK_RECORD=dock.domain.tld
(cd go-dock-sh && go install && go-dock-sh)
curl -vx socks5h://localhost:60101 http://google.com/
sqlite3 ~/go/bin/go-dock-ms.db3 ".tables"
sqlite3 ~/go/bin/go-dock-ms.db3 ".schema key_dros"
sqlite3 ~/go/bin/go-dock-ms.db3 ".schema ship_dros"
sqlite3 ~/go/bin/go-dock-ms.db3 ".schema log_dros"
sqlite3 ~/go/bin/go-dock-ms.db3 "select * from key_dros"
sqlite3 ~/go/bin/go-dock-ms.db3 "select * from ship_dros"
sqlite3 ~/go/bin/go-dock-ms.db3 "select * from log_dros"
#no key management wapi to avoid the extra endpoint
sqlite3 ~/go/bin/go-dock-ms.db3 \
    "insert into key_dros (host, name, key) values ('`hostname`', 'default', readfile('$HOME/.ssh/id_rsa.pub'))"
#for go-sqlite in linux
sudo apt install build-essentials
dig dock.domain.tld TXT
```
