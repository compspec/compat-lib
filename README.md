# Compat Lib

This is a library that provides a compatibility service. The basic idea is that:

1. Generate compatibility artifacts that describe applications (or containers) of interest
2. They can live in a local cache or a registry
3. A service (like a daemon) runs on a node and can evaluate if the node is compatible with the application.

To start, we will look at software. We will generate artifacts that describe binaries and the libraries that are needed.
We will then run the service that will discover the paths provided on the host (exposed via ldd) and be able to quickly answer
if this is compatible or not. Ironically, as I was exploring this space I realized it was an easy way to cache library locations
based on soname, which could be used akin to a tool like [spindle](https://github.com/LLNL/spindle). I started testing
that (see the first idea) but ultimately returned to the compatibility use case because I find it more interesting.

## Usage

For all examples below, build first.

```bash
make
```

To make the proto (or re-generate, if necessary):

```bash
make proto
```

## Ideas

### 1. Library Discovery Wrapper

> Figure out what shared libraries are needed via an open intercept

This was the experiment to generate something akin to spindle. I think it still has feet, I just got interested in other things more. Next I need to create some kind of cache. We can:

1. Parse the ELF to get sonames needed (if we want to see them in advance, this isn't actually necessary)
2. Generate a fuse overlay where everything will be found in one spot (no searching needed)
3. Write a create function for a loopback filesystem that will intercept calls
4. Use proot (or similar, I used proot since I don't want to use root) to execute a command to our mounted filesystem
5. Then execute the binary, see the open calls.

We would want to see that the exercise of not needing to do the search speeds up that loading time. If it does, it would make sense to pre-package this metadata with the binary for some registry to use. Here is how to run it with a binary:

```bash
./bin/fs-gen /home/vanessa/Desktop/Code/spack/opt/spack/linux-ubuntu24.04-zen4/gcc-13.2.0/xz-5.4.6-klise22d77jjaoejkucrczlkvnm6f4au/bin/xz --help
```

This one has a few more paths:

```bash
./bin/fs-gen /home/vanessa/Desktop/Code/spack/opt/spack/linux-ubuntu24.04-zen4/gcc-13.2.0/hwloc-2.11.1-zuv2etx7sgd5yn6khpblfw6qjh54lpsp/bin/hwloc-ls
```

Next we would want to add some kind of cache to store file descriptors (or paths) and return something else.
This could also be used in a compatibility context to figure out what a binary needs before running it, and give it to a scheduler,
but I'm not sure that use case is of interest.

### 2. Compatibility Wrapper

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

🚧️ Under Development! 🚧️


## License

HPCIC DevTools is distributed under the terms of the MIT license.
All new contributions must be made under this license.

See [LICENSE](https://github.com/converged-computing/cloud-select/blob/main/LICENSE),
[COPYRIGHT](https://github.com/converged-computing/cloud-select/blob/main/COPYRIGHT), and
[NOTICE](https://github.com/converged-computing/cloud-select/blob/main/NOTICE) for details.

SPDX-License-Identifier: (MIT)

LLNL-CODE- 842614
