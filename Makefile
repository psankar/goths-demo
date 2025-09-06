all:
	cd templ && templ generate
	cd sqlc && sqlc generate
	go run main.go
