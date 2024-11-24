# Slim

This is an experiment to build a container with only the assets that are needed.

Running the example:

```bash
cd /opt/lammps/examples/reaxff/HNS
slim lmp -v x 1 -v y 1 -v z 1 -in ./in.reaxff.hns -nocite

# Mount to a persistent location and keep the cache
slim --keep --mount-path /home/sochat1_llnl_gov/compat-lib/example/slim/lammps lmp -v x 1 -v y 1 -v z 1 -in ./in.reaxff.hns -nocite
```

```bash
docker build -t test .
docker run --workdir /opt/lammps/examples/reaxff/HNS --entrypoint /usr/bin/lmp -it test -v x 1 -v y 1 -v z 1 -in ./in.reaxff.hns -nocite
```
