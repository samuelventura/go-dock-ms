# go-dock-ms

SSH dock micro service

- Reverse SOCKS proxy only
- Single text line dialing
- TXT record load balancing (client side)
- DB based data exchange with public facing proxy

Next Steps

- White list check ship name for partitioning
- Check against white list before ssh handling
- Key handling RESTish API
- Ship state RESTish API

## Development

```bash
#run locally
ln -sf ~/github/go-dock-ms/id_rsa.key ~/go/bin/go-dock-ms.key
sqlite3 ~/go/bin/go-dock-ms.db3 "delete from key_dros"
sqlite3 ~/go/bin/go-dock-ms.db3 "insert into key_dros (name, key) values ('default', readfile('./id_rsa.pub'))"
sqlite3 ~/go/bin/go-dock-ms.db3 "insert into key_dros (name, key) values ('user', readfile('$HOME/.ssh/id_rsa.pub'))"
sqlite3 ~/go/bin/go-dock-ms.db3 "select * from key_dros"
go install && ~/go/bin/go-dock-ms
#kill and dump stacktrace to test keepalive timeout
killall go-dock-to
ps -A | grep go-
kill -ABRT <pid>
#manually check DNS records
dig dock.domain.tld TXT
```
