provider "vsphereprivate" {
  user                 = var.vsphere_username
  password             = var.vsphere_password
  vsphere_server       = var.vsphere_url
  allow_unverified_ssl = true
}

resource "vsphereprivate_import_ova" "import" {
  name       = "rhcos-44.81.202003062006-0-vmware"
  path       = "/var/home/jcallen/Downloads/rhcos-44.81.202003062006-0-vmware.x86_64.ova"
  cluster    = var.vsphere_cluster
  datacenter = var.vsphere_datacenter
  datastore  = var.vsphere_datastore
  network    = var.vsphere_network
  folder     = var.vsphere_folder
}
