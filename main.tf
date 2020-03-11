provider "vsphere" {
  user                 = var.vsphere_username
  password             = var.vsphere_password
  vsphere_server       = var.vsphere_url
  allow_unverified_ssl = true
}


provider "vsphereprivate" {
  user                 = var.vsphere_username
  password             = var.vsphere_password
  vsphere_server       = var.vsphere_url
  allow_unverified_ssl = true
}

data "vsphere_datacenter" "datacenter" {
  name = var.vsphere_datacenter
}

data "vsphere_compute_cluster" "cluster" {
  name          = var.vsphere_cluster
  datacenter_id = data.vsphere_datacenter.datacenter.id
}

data "vsphere_datastore" "datastore" {
  name          = var.vsphere_datastore
  datacenter_id = data.vsphere_datacenter.datacenter.id
}

data "vsphere_network" "network" {
  name          = var.vsphere_network
  datacenter_id = data.vsphere_datacenter.datacenter.id
}

resource "vsphereprivate_import_ova" "import" {
  name       = "rhcos-44.81.202003062006-0-vmware"
  path       = "/tmp/rhcos-44.81.202003062006-0-vmware.x86_64.ova"
  cluster    = var.vsphere_cluster
  datacenter = var.vsphere_datacenter
  datastore  = var.vsphere_datastore
  network    = var.vsphere_network
  folder     = "foo"
}
