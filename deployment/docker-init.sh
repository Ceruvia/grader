#!/bin/sh
# Image entrypoint. systemd as PID 1 points its stdio at /dev/null, so nothing in the
# container could reach `docker logs` any more. The sleeper keeps Docker's stdout pipe open
# and /dev/docker-log names it for grader-log-forward.service (/dev, not /run: systemd
# mounts a fresh tmpfs on /run).
# No `set -e`: a failed link only costs docker logs (the forwarder retries, the journal keeps
# every line), so systemd and grading must boot anyway.
sleep infinity &
ln -sf "/proc/$!/fd/1" /dev/docker-log
exec /sbin/init "$@"
