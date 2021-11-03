require (
	github.com/Microsoft/go-winio v0.4.14 // indirect
	github.com/Nvveen/Gotty v0.0.0-20120604004816-cd527374f1e5 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/go-chi/chi v4.0.2+incompatible // indirect
	github.com/go-chi/jwtauth v4.0.4+incompatible
	github.com/golang-migrate/migrate/v4 v4.11.0
	github.com/google/uuid v1.1.1
	github.com/gorilla/websocket v1.4.1 // indirect
	github.com/jmoiron/sqlx v1.2.0
	github.com/lib/pq v1.5.2
	github.com/mattn/go-sqlite3 v1.11.0 // indirect
	github.com/mongodb/mongo-go-driver v0.3.0 // indirect
	github.com/sirupsen/logrus v1.4.2 // indirect
	github.com/skip2/go-qrcode v0.0.0-20191027152451-9434209cb086
	github.com/stretchr/testify v1.4.0 // indirect
	golang.org/x/crypto v0.0.0-20200510223506-06a226fb4e37
	golang.org/x/time v0.0.0-20190921001708-c4c64cad1fd0 // indirect
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gotest.tools v2.2.0+incompatible // indirect	
	github.com/Rhymen/go-whatsapp v0.0.0
)

replace github.com/sufficit/sufficit-quepasa-fork/models => ./

replace github.com/Rhymen/go-whatsapp => github.com/sufficit/sufficit-go-whatsapp v0.1.12
// replace github.com/Rhymen/go-whatsapp => Z:\Desenvolvimento\sufficit-go-whatsapp

go 1.14
