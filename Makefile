include .envrc

.PHONY: run
run:
	go run main.go -port=${API_PORT} -redisAddr=${REDIS_HOST}:${REDIS_PORT}
	
