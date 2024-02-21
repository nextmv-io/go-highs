#!/usr/bin/env bash
set -e

###### README ######
# This script is meant to be executed once after a fresh clone of the repo.
# It will setup some prerequisites.

# >>> Git LFS
git lfs checkout
git lfs pull
