FROM ubuntu:latest
LABEL authors="ankha"

ENTRYPOINT ["top", "-b"]