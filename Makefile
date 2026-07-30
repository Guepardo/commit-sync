.PHONY: complexity

complexity:
	go run github.com/fzipp/gocyclo/cmd/gocyclo -top 20 ./
