# Compat Lib

This is a library that prepares artifacts that describe the libraries needed (exposed via ldd) for a specific binary.
We are going to do several experiments here (as I learn about the space).

## Compatibility Wrapper

If we wrap a binary, we can:

1. Parse the ELF to get sonames needed
2. Generate a fuse overlay where everything will be found in one spot (no searching needed)
3. Then execute the binary.

We would want to see that the exercise of not needing to do the search speeds up that loading time. If it does, it would make sense to pre-package this metadata with the binary for some registry to use.


🚧️ Under Development! 🚧️

## Usage

Build the binary

```bash
make
```

Test running with a binary. Note that since the go program starts the filesystem and we use chroot (outside of it) to launch our job to that root, we have them currently
separate. If there is a clean way to orchestrate Go running the chroot and binary I'll keep looking for it.

```bash
./bin/clib-gen /home/vanessa/Desktop/Code/spack/opt/spack/linux-ubuntu24.04-zen4/gcc-13.2.0/xz-5.4.6-klise22d77jjaoejkucrczlkvnm6f4au/bin/xz --help
```

Work in progress! The above calls the custom open function, so now we can do a special case for the libraries.

TODO:
1. Write a wrapper that provides the binary here
2. Get the pid with &
3. Run the thing
4. Then exit.

The wrapper probablhy won't work.


## License

HPCIC DevTools is distributed under the terms of the MIT license.
All new contributions must be made under this license.

See [LICENSE](https://github.com/converged-computing/cloud-select/blob/main/LICENSE),
[COPYRIGHT](https://github.com/converged-computing/cloud-select/blob/main/COPYRIGHT), and
[NOTICE](https://github.com/converged-computing/cloud-select/blob/main/NOTICE) for details.

SPDX-License-Identifier: (MIT)

LLNL-CODE- 842614
