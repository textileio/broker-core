package store

//go:generate go-bindata -pkg migrations -prefix migrations/ -o migrations/migrations.go -ignore=migrations.go migrations/
//go:generate sqlc generate
