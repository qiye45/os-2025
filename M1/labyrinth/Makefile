NAME := $(shell basename $(PWD))
export MODULE := M1
all: $(NAME)

include ../.shadow/oslabs.mk

go-run:
	go run labyrinth.go

go-build:
	go build -o labyrinth labyrinth.go