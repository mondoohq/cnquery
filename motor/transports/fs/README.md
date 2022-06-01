# Virtual Filesystem

This transport reads a directory and treats it as its own platform. This is useful if you want to do a static analysis of a mounted operating system, where you wan to ensure nothing is running.

## Testing

If you need to test a remote linux system on macos, it is possible to spin up the machine and mount the whole filesystem to your local machine.

```bash
# NOTE: make sure you do NOT run the following as sudo

# create local mount directory
mkdir -p ~/mnt/minikube
sshfs -o allow_other,default_permissions root@192.168.99.103:/  ~/mnt/minikube

cat ~/mnt/minikube/etc/os-release 
NAME=Buildroot
VERSION=2020.02.10
ID=buildroot
VERSION_ID=2020.02.10
PRETTY_NAME="Buildroot 2020.02.10"
```

References
- https://www.digitalocean.com/community/tutorials/how-to-use-sshfs-to-mount-remote-file-systems-over-ssh
- https://osxfuse.github.io/
- https://github.com/osxfuse/sshfs/releases