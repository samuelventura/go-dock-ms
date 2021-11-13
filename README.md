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

## API

```bash
#key management
curl -X GET http://127.0.0.1:31623/api/key/list
curl -X GET http://127.0.0.1:31623/api/key/info/:name
curl -X POST http://127.0.0.1:31623/api/key/delete/:name
curl -X POST http://127.0.0.1:31623/api/key/enable/:name
curl -X POST http://127.0.0.1:31623/api/key/disable/:name
curl -X POST http://127.0.0.1:31623/api/key/add/:name -F "file=@filepath"
#ship management
curl -X GET http://127.0.0.1:31623/api/ship/count
curl -X GET http://127.0.0.1:31623/api/ship/info/:name
curl -X POST http://127.0.0.1:31623/api/ship/add/:name
curl -X POST http://127.0.0.1:31623/api/ship/port/:name/:port
curl -X POST http://127.0.0.1:31623/api/ship/remove/:name
curl -X POST http://127.0.0.1:31623/api/ship/enable/:name
curl -X POST http://127.0.0.1:31623/api/ship/disable/:name
curl -X POST http://127.0.0.1:31623/api/ship/stop/:name
curl -X GET http://127.0.0.1:31623/api/ship/status/:name
```

## Test Drive

```bash
curl -X GET http://127.0.0.1:31623/api/key/list
curl -X GET http://127.0.0.1:31623/api/key/info/default
curl -X POST http://127.0.0.1:31623/api/key/delete/default
curl -X POST http://127.0.0.1:31623/api/key/enable/default
curl -X POST http://127.0.0.1:31623/api/key/disable/default
curl -X POST http://127.0.0.1:31623/api/key/add/default -F "file=@id_rsa.pub"
curl -X GET http://127.0.0.1:31623/api/ship/count
curl -X GET http://127.0.0.1:31623/api/ship/count/enabled
curl -X GET http://127.0.0.1:31623/api/ship/count/disabled
curl -X GET http://127.0.0.1:31623/api/ship/info/sample
curl -X GET http://127.0.0.1:31623/api/ship/status/sample
curl -X POST http://127.0.0.1:31623/api/ship/add/sample
curl -X POST http://127.0.0.1:31623/api/ship/port/sample/4000
curl -X POST http://127.0.0.1:31623/api/ship/enable/sample
curl -X POST http://127.0.0.1:31623/api/ship/disable/sample
curl -X POST http://127.0.0.1:31623/api/ship/close/sample
```

## Development

```bash
#run locally
ln -sf ~/github/go-dock-ms/id_rsa.key ~/go/bin/go-dock-ms.key
sqlite3 ~/go/bin/go-dock-ms.db3 "delete from key_dros"
sqlite3 ~/go/bin/go-dock-ms.db3 "insert into key_dros (enabled, name, key) values (true, 'default', readfile('./id_rsa.pub'))"
sqlite3 ~/go/bin/go-dock-ms.db3 "insert into key_dros (enabled, name, key) values (true, 'user', readfile('$HOME/.ssh/id_rsa.pub'))"
sqlite3 ~/go/bin/go-dock-ms.db3 "select * from key_dros"
go install && ~/go/bin/go-dock-ms
#kill and dump stacktrace to test keepalive timeout
killall go-dock-to
ps -A | grep go-
kill -ABRT <pid>
#manually check DNS records
dig dock.domain.tld TXT
```
