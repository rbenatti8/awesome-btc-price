test:
	@time go test -race -parallel=10 -p=5 -coverprofile coverage.out ./...

docker-build:
	@docker build -t awesome-bttc-price:0.0.1 .

docker-run:
	@docker run --rm -it -p 3000:3000 --env-file .env awesome-bttc-price:0.0.1