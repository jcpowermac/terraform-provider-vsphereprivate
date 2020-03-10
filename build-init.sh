#!/bin/bash
set -xe

go build -o terraform-provider-vsphereprivate
./terraform init
./terraform apply -auto-approve
