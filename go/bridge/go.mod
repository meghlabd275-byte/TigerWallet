module github.com/tigerwallet/bridge

go 1.23

require github.com/tigerwallet/sharedauth v0.0.0

require github.com/golang-jwt/jwt/v5 v5.2.1 // indirect

replace github.com/tigerwallet/sharedauth => ../sharedauth
