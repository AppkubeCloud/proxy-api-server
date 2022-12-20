# proxy-api-server
proxy api server that talks with multiple api server


1. go mod init proxy-api-server

2. Install packages
go get github.com/gorilla/mux
go get github.com/gosimple/slug
go get github.com/gosimple/unidecode
go get github.com/grafana-tools/sdk
go get github.com/layer5io/meshkit
go get github.com/pkg/errors
go get github.com/rainycape/unidecode
go get github.com/sirupsen/logrus
go get golang.org/x/sys
go get golang.org/x/text

3. go run main.go

APIS:

http://localhost:10000/grafana-ds?grafanaUrl=http://grafana.synectiks.net&apiKey=admin:password

http://localhost:10000/grafana-ds/query-range?ds=Prometheus&query=sum(istio_build%7Bcomponent%3D%22pilot%22%7D)%20by%20(tag)&start=1671095526&end=1671095826&step=5&url=http%3A%2F%2Fgrafana.synectiks.net%3A80&api-key=admin:password