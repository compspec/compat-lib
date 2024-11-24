#!/bin/bash

for dirname in $(ls /cache)
do
  cp -R /cache/$dirname/* /$dirname/
done
chmod +x /usr/bin/lmp
