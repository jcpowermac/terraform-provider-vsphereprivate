#!/bin/bash
set -xe

export TF_LOG=TRACE
export TF_LOG_PATH=$(pwd)/terraform.tf.log


go build -o terraform-provider-vsphereprivate
./terraform init
./terraform destroy -auto-approve
