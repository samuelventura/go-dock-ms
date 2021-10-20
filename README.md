# go-dock-ms

SSH dock micro service

- Reverse SOCKS proxy only
- TXT record load balancing

## development

```bash
#dock service
ln -sf ~/github/go-dock-ms/id_rsa.key ~/go/bin/go-dock-ms.key
sqlite3 ~/go/bin/go-dock-ms.db3 "delete from key_dros"
sqlite3 ~/go/bin/go-dock-ms.db3 "insert into key_dros (host, name, key) values ('`hostname`', 'default', readfile('$HOME/github/go-dock-ms/id_rsa.pub'))"
sqlite3 ~/go/bin/go-dock-ms.db3 "select * from key_dros"
go install && go-dock-ms
#ship instance
ln -sf ~/github/go-dock-ms/id_rsa.key ~/go/bin/go-dock-sh.key
(cd go-dock-sh && go install && go-dock-sh)
#use the proxy
(cd go-dock-to && go install && go-dock-to 127.0.0.1:$PORT localhost:31699)
GET /
#kill and dump stacktrace to test timeout
killall go-dock-to
ps -A | grep go-
kill -ABRT <pid>
```

## helpers

```bash
#FIXME keepalive, mac whitelist, ssh nop log, sock5 nop log, replace connection by mac
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
curl -vx socks5h://127.0.0.1:PORT http://google.com/
sqlite3 ~/go/bin/go-dock-ms.db3 ".tables"
sqlite3 ~/go/bin/go-dock-ms.db3 ".schema key_dros"
sqlite3 ~/go/bin/go-dock-ms.db3 ".schema ship_dros"
sqlite3 ~/go/bin/go-dock-ms.db3 ".schema log_dros"
sqlite3 ~/go/bin/go-dock-ms.db3 "select * from key_dros"
sqlite3 ~/go/bin/go-dock-ms.db3 "select * from ship_dros"
sqlite3 ~/go/bin/go-dock-ms.db3 "select * from log_dros"
#no key management wapi to avoid the extra endpoint
#sqlite3 ~/go/bin/go-dock-ms.db3 "insert into key_dros (host, name, key) values ('`hostname`' 'default', readfile('$HOME/github/go-dock-ms/id_rsa.pub'))"
#sudo sqlite3 /usr/local/bin/go-dock-ms.db3 "insert into key_dros (host, name, key) values ('`hostname`', 'default', readfile('$HOME/local/go-dock-ms/id_rsa.pub'))"
#sudo sqlite3 /usr/local/bin/go-dock-ms.db3 "select * from key_dros"
#sudo ln -sf ~/local/go-dock-ms/id_rsa.key /usr/local/bin/go-dock-ms.key
#ln -sf ~/local/go-dock-ms/id_rsa.key ~/go/bin/go-dock-sh.key
#sudo tail -f /usr/local/bin/go-dock-ms.out.log 
#export DOCK_POOL=127.0.0.1:31652
#for go-sqlite in linux
sudo apt install build-essentials
dig dock.domain.tld TXT
```


