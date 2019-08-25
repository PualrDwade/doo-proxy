# doo-proxy
![](https://img.shields.io/badge/language-go-blue.svg)  ![](https://img.shields.io/badge/proxy-http-gren.svg)

doo-proxy is a simple http/https proxy implement by golang

## quick start

``` bash
go get github.com/PualrDwade/doo-proxy

go install github.com/PualrDwade/doo-proxy

./doo-proxy -credential "aksldjhlkasdj" (your credential)

```

## key funciton

all thing to understand is a code snippet bellow:
```golang
func (proxy *dooProxy) tunnel(clientConn net.Conn, remoteConn net.Conn) {
	group := &sync.WaitGroup{}
	group.Add(2)
	go func() {
		defer group.Done()
		_, err := io.Copy(remoteConn, clientConn)
		if err != nil {
			log.Error(err.Error())
		}
	}()
	go func() {
		defer group.Done()
		_, err := io.Copy(clientConn, remoteConn)
		if err != nil {
			log.Error(err.Error())
		}
	}()
	group.Wait()
}
```