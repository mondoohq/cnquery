# Proc Filesystem for Linux

This package provides various helper to parse information for the proc filesystem. The best description is its [specification](https://www.kernel.org/doc/Documentation/filesystems/proc.txt). This guided the implementation of the parser.

## Why are you reinventing the wheel? There is a sysctl, top etc already available

The aim of this implementation is not to replace those tools, instead its main focus is to provided structured information for uses. Therefore this implementation is mainly targeted to machine readers instead of human readers.

The main difference compared to other versions is that this implementation does not expect to have access to the system directly. It is intended to work via a remote connection, too.

## Notable other implementations

- https://github.com/shirou/gopsutil
- https://github.com/c9s/goprocinfo
- https://github.com/prometheus/procfs
- https://github.com/bcicen/ctop
