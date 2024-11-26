#!/usr/bin/env bash

set -e

function install_kubevirt_job {
    _kubectl apply -f "./_out/manifests/release/kubevirt-job.yaml"
}

function delete_kubevirt_job {
  if [ -f "./_out/manifests/release/kubevirt-job.yaml" ]; then
    _kubectl delete --ignore-not-found -f "./_out/manifests/release/kubevirt-job.yaml"
  else
    echo "File ./_out/manifests/release/kubevirt-job.yaml does not exist."
  fi
}