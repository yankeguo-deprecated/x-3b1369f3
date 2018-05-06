# minit

minit is a deamon for docker container as a sandbox

minit listens on a socket file, or any other bi-direction streams (TCP connection, etc)

minit uses `stdcopy` from `github.com/docker/docker` as stream multiplexing protocol