test:
	@time go test -race -parallel=10 -p=5 -coverprofile coverage.out ./...

docker-build:
	@docker build -t awesome-bttc-price:0.0.1 .

docker-run:
	@docker run --rm -it -p 3000:3000 -e TOKEN=a56afa8fab481d71649f36188a8d344154fd2270554acc18de5efbd6130999b2 awesome-bttc-price:0.0.1