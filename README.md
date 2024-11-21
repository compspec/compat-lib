# Compat Lib

![PyPI - Version](https://img.shields.io/pypi/v/compatlib)

This is a library that provides tools for compatibility and understanding of filesystem events. There
are several tools packaged here:

- **fs-record**: records events (paths and timestamps) that occur when you run an application. Since this is run with fuse (in user space) it can work on a local machine (a binary directly) or via a container! This was the third project I created here that I ultimately find the most interesting.
- **compat-gen**: is a tool that records library (or generally software) loading when you run a binary, and generates a compatibility artifact.
- **spindle** and **spindle-server**: are a library discovery wrapper and server to distribute the cache across nodes, respectively. It was the original prototype that I created to emulate [spindle](https://github.com/hpc/Spindle) and I need to have discussion with the spindle developers about what they would like to do next! It was paramount for me to learn how to use fuse to write custom events (and further understand the [SOCI snapshotter](https://youtu.be/ZXM1gP4goP8?si=MokyPbLl95nN8Un9)).


## Usage

For all examples below, build first.

```bash
make
```

To make the proto (or re-generate, if necessary):

```bash
make proto
```

## Tools

### 1. Application Recorder

> **fs-record** to record filesystem events using a custom Fuse filesystem (works in a container too)!

This tool does the following:

1. Record the file access of running some HPC application (in a container or not) meaning paths and timestamps since start.
2. Record file access of a set of "the same" app over time to assess differences.

```bash
./bin/fs-record /home/vanessa/Desktop/Code/spack/opt/spack/linux-ubuntu24.04-zen4/gcc-13.2.0/xz-5.4.6-klise22d77jjaoejkucrczlkvnm6f4au/bin/xz --help
```

Here is how to customize the output file name:

```bash
./bin/fs-record --out ./example/compat/xz-libs.txt /home/vanessa/Desktop/Code/spack/opt/spack/linux-ubuntu24.04-zen4/gcc-13.2.0/xz-5.4.6-klise22d77jjaoejkucrczlkvnm6f4au/bin/xz
```

Test running in a container, and binding the binary!

```bash
# Test running lammps first
docker run -it ghcr.io/converged-computing/lammps-time:stable_29Aug2024_update1 lmp -v x 2 -v y 2 -v z 2 -in ./in.reaxff.hns -nocite

# Now record!
docker run -v $PWD/bin:/compat --security-opt apparmor:unconfined --device /dev/fuse --cap-add SYS_ADMIN -it ghcr.io/converged-computing/lammps-time-fuse:stable_29Aug2024_update1 /compat/fs-record --out /compat/lammps-run-1.out lmp -v x 2 -v y 2 -v z 2 -in ./in.reaxff.hns -nocite

# With a temporary file in the PWD
docker run -v $PWD/bin:/compat --security-opt apparmor:unconfined --device /dev/fuse --cap-add SYS_ADMIN -it ghcr.io/converged-computing/lammps-time-fuse:stable_29Aug2024_update1 /compat/fs-record --out-dir /compat lmp -v x 2 -v y 2 -v z 2 -in ./in.reaxff.hns -nocite
```

We provide functions in Python under [python/compatlib](python/compatlib) for parsing and generating models for the event files. You can see using the library [here](https://github.com/converged-computing/lammps-time/tree/main/experiments/local-kind), and early work [in the lammps-time repository](https://github.com/converged-computing/lammps-time/tree/main/fuse/analysis) to do this that has since been turned into the library here. The next stage of work for that project will use the library here.


### 2. Compatibility Wrapper

> **compat-gen** to generate compatibility artifacts

This idea was suited for the use case of compatibility, but I'm not sure it has captured enough interest yet to continue working on. The general idea is the following:

1. Generate compatibility artifacts that describe applications (or containers) of interest
2. They can live in a local cache or a registry
3. A service (like a daemon) runs on a node and can evaluate if the node is compatible with the application.

To start, I wanted to look at software. I generated an example artifact that described a binary and the libraries that are needed.
The next step was to run the service that will discover the paths provided on the host (exposed via ldd) and be able to quickly answer if this is compatible or not. Ironically, as I was exploring this space I realized it was an easy way to cache library locations based on soname, which could be used akin to a tool like [spindle](https://github.com/LLNL/spindle). I started testing that (see the first idea) but ultimately returned to the compatibility use case because I find it more interesting.

> Too long, didn't read

For this idea, I'll have one entrypoint that can generate a compatibility artifact for some binary. This will just be the .so libraries that are needed for the binary (along with the binary).
Then I'll have a grpc server / service (or could also be a database) that you run to discover the paths on the node, and you can pull the artifact in advance to check if its compatible. Let's do a dummy
case. First, generate the artifact.

```bash
./bin/compat-gen /home/vanessa/Desktop/Code/spack/opt/spack/linux-ubuntu24.04-zen4/gcc-13.2.0/xz-5.4.6-klise22d77jjaoejkucrczlkvnm6f4au/bin/xz
```
```console
⭐️ Compatibility Library Generator (clib-gen)
Preparing to find shared libraries needed for [/home/vanessa/Desktop/Code/spack/opt/spack/linux-ubuntu24.04-zen4/gcc-13.2.0/xz-5.4.6-klise22d77jjaoejkucrczlkvnm6f4au/bin/xz]
{
  "version": "0.0.1",
  "attributes": {
    "llnl.compatlib.executable-name": "xz",
    "llnl.compatlib.library-name.0": "ld-linux-x86-64.so.2",
    "llnl.compatlib.library-name.1": "liblzma.so.5",
    "llnl.compatlib.library-name.2": "libc.so.6"
  }
}
```

Now let's save that to file.

```bash
./bin/compat-gen --out ./example/compat/xz-libs.json /home/vanessa/Desktop/Code/spack/opt/spack/linux-ubuntu24.04-zen4/gcc-13.2.0/xz-5.4.6-klise22d77jjaoejkucrczlkvnm6f4au/bin/xz
```

We could now push that to a registry with ORAS, but we are first going to test with a server. The following should happen:

1. The server starts and is oriented to a mode to parse libraries on the host.
2. The client is run to request a compatibility check of the artifact against that node (comparing libraries needed)
3. If all paths can be satisfied, we get an affirmative response, otherwise nope.

To run the server:

```bash
./bin/compat-server
```
```console
2024/10/13 17:58:41 🧩 starting compatibility server: compatServer v0.0.1
2024/10/13 17:58:41 server listening: [::]:50051
```

### 3. Library Discovery Wrapper (spindle)

> **spindle** to figure out what shared libraries are needed via an open intercept, and **spindle-server** to distribute the cache across nodes.

This was the experiment to generate something akin to spindle. I think it still has feet, I just got interested in other things more. Next I need to create some kind of cache. We can:

1. Parse the ELF to get sonames needed (if we want to see them in advance, this isn't actually necessary)
2. Generate a fuse overlay where everything will be found in one spot (no searching needed)
3. Write a create function for a loopback filesystem that will intercept calls
4. Use proot (or similar, I used proot since I don't want to use root) to execute a command to our mounted filesystem
5. Then execute the binary, see the open calls.

We would want to see that the exercise of not needing to do the search speeds up that loading time. If it does, it would make sense to pre-package this metadata with the binary for some registry to use.  Here is how to run it with a binary:

```bash
./bin/spindle /home/vanessa/Desktop/Code/spack/opt/spack/linux-ubuntu24.04-zen4/gcc-13.2.0/xz-5.4.6-klise22d77jjaoejkucrczlkvnm6f4au/bin/xz --help
```

This one has a few more paths:

```bash
./bin/spindle /home/vanessa/Desktop/Code/spack/opt/spack/linux-ubuntu24.04-zen4/gcc-13.2.0/hwloc-2.11.1-zuv2etx7sgd5yn6khpblfw6qjh54lpsp/bin/hwloc-ls
```

```bash
./bin/spindle sleep 2
```

Next we would want to add some kind of cache to store file descriptors (or paths) and return something else.
This could also be used in a compatibility context to figure out what a binary needs before running it, and give it to a scheduler, but I'm not sure that use case is of interest.


## License

HPCIC DevTools is distributed under the terms of the MIT license.
All new contributions must be made under this license.

See [LICENSE](https://github.com/converged-computing/cloud-select/blob/main/LICENSE),
[COPYRIGHT](https://github.com/converged-computing/cloud-select/blob/main/COPYRIGHT), and
[NOTICE](https://github.com/converged-computing/cloud-select/blob/main/NOTICE) for details.

SPDX-License-Identifier: (MIT)

LLNL-CODE- 842614
