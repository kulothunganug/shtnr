# shtnr

a simple url shortener service

## why?

- to try google cloud
- to learn how to test go codebase
- to integrate swagger ui in go app

## tech stack

- go
- sqlite
- sqlc
- swaggo

## api

- `POST /shorten` with `{"url": "https://example.com"}`
- `GET /{shortCode}` redirects to original url

## try it

[swagger ui](https://test-61983252144.europe-west1.run.app/swagger/index.html)
