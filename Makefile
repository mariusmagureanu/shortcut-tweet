# Makefile
#

all:
	@$(MAKE) --directory=./streamer
	@$(MAKE) --directory=./client

clean:
	@$(MAKE) --directory=./streamer clean
	@$(MAKE) --directory=./client clean
	@docker container stop $(docker container ls -aq)
	@docker container rm $(docker container ls -aq)

docker: all
	@docker-compose up --build --scale client=5

