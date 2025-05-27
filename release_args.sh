#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail
# This script is automatically called by the automatic git flow release process. It is responsible to change the
# version of the sonarcarp
makefile=sonarcarp/Makefile

update_versions_modify_files() {
  local newReleaseVersion="${1}"

  if [ -f "${makefile}" ]; then
    echo "Updating version in sonarcarp-Makefile..."
    sed -i "s/\(^VERSION=\)\(.*\)$/\1${newReleaseVersion}/" "${makefile}"
  fi
}

update_versions_stage_modified_files() {
  git add "${makefile}"
}
