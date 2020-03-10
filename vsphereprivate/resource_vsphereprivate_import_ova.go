package vsphereprivate

import (
	"context"
	"fmt"
	"log"
	_ "strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"

	"github.com/pkg/errors"
	_ "github.com/vmware/govmomi/object"
	_ "github.com/vmware/govmomi/vim25/types"

	//_ "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
)

func resourceVSpherePrivateImportOva() *schema.Resource {
	return &schema.Resource{
		Create: resourceVSpherePrivateImportOvaCreate,
		Read:   resourceVSpherePrivateImportOvaRead,
		Update: resourceVSpherePrivateImportOvaUpdate,
		Delete: resourceVSpherePrivateImportOvaDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		SchemaVersion: 1,
		//MigrateState:  resourceVSphereFolderMigrateState,
		/*
		   type ImportOvaParams struct {
		   	ResourcePool *object.ResourcePool
		   	Datacenter   *object.Datacenter
		   	Datastore    *object.Datastore
		   	Network      *object.Network
		   	Host         *object.HostSystem
		   	Folder       *object.Folder
		   }
		*/

		Schema: map[string]*schema.Schema{
			"path": {
				Type:         schema.TypeString,
				Description:  "",
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
			},
			"datacenter": {
				Type:        schema.TypeString,
				Description: "The ID of the datacenter. Can be ignored if creating a datacenter folder, otherwise required.",
				Optional:    false,
				Required:    true,
			},
			"cluster": {
				Type:        schema.TypeString,
				Description: "The ID of the datacenter. Can be ignored if creating a datacenter folder, otherwise required.",
				Required:    true,
			},
			"network": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of a network that the virtual machine will use.",
			},
			"datastore": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the virtual machine's datastore. The virtual machine configuration is placed here, along with any virtual disks that are created without datastores.",
			},
			"folder": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the folder to locate the virtual machine in.",
			},

			// Tagging
			//vSphereTagAttributeKey: tagsSchema(),
			// Custom Attributes
			//customattribute.ConfigKey: customattribute.ConfigSchema(),
		},
	}
}

// ImportOvaParams contains the vCenter objects required to import a OVA into vSphere.
type ImportOvaParams struct {
	ResourcePool *object.ResourcePool
	Datacenter   *object.Datacenter
	Datastore    *object.Datastore
	Network      *object.Network
	Host         *object.HostSystem
	Folder       *object.Folder
}

func findImportOvaParams(client *vim25.Client, datacenter, cluster, datastore, network string) (*ImportOvaParams, error) {
	var ccrMo mo.ClusterComputeResource
	ctx := context.TODO()
	importOvaParams := &ImportOvaParams{}

	finder := find.NewFinder(client)

	// Find the object Datacenter by using its name provided by install-config
	dcObj, err := finder.Datacenter(ctx, datacenter)
	if err != nil {
		return nil, err
	}
	importOvaParams.Datacenter = dcObj

	// Find the top-level (and hidden to view) folders in the
	// datacenter
	folders, err := importOvaParams.Datacenter.Folders(ctx)
	if err != nil {
		return nil, err
	}
	// The only folder we are interested in is VmFolder
	// Which can contain our template
	importOvaParams.Folder = folders.VmFolder

	clusterPath := fmt.Sprintf("/%s/host/%s", datacenter, cluster)

	// Find the cluster object by the datacenter and cluster name to
	// generate the path e.g. /datacenter/host/cluster
	clusterComputeResource, err := finder.ClusterComputeResource(ctx, clusterPath)
	if err != nil {
		return nil, err
	}

	// Get the network properties that is defined in ClusterComputeResource
	// We need to know if the network name provided exists in the cluster that was
	// also provided.
	err = clusterComputeResource.Properties(context.TODO(), clusterComputeResource.Reference(), []string{"network"}, &ccrMo)
	if err != nil {
		return nil, err
	}

	// Find the network object using the provided network name
	for _, networkMoRef := range ccrMo.Network {
		networkObj := object.NewNetwork(client, networkMoRef)
		networkObjectName, err := networkObj.ObjectName(ctx)
		if err != nil {
			return nil, err
		}
		if network == networkObjectName {
			importOvaParams.Network = networkObj
			break
		}
	}

	// Find all the datastores that are configured under the cluster
	datastores, err := clusterComputeResource.Datastores(ctx)
	if err != nil {
		return nil, err
	}

	// Find the specific datastore by the name provided
	for _, datastoreObj := range datastores {
		datastoreObjName, err := datastoreObj.ObjectName(ctx)
		if err != nil {
			return nil, err
		}
		if datastore == datastoreObjName {
			importOvaParams.Datastore = datastoreObj
			break
		}
	}

	// Find all the HostSystem(s) under cluster
	hosts, err := clusterComputeResource.Hosts(ctx)
	if err != nil {
		return nil, err
	}
	foundDatastore := false
	foundNetwork := false
	var hostSystemManagedObject mo.HostSystem

	// Confirm that the network and datastore that was provided is
	// available for use on the HostSystem we will import the
	// OVA to.
	for _, hostObj := range hosts {
		hostObj.Properties(ctx, hostObj.Reference(), []string{"network", "datastore"}, &hostSystemManagedObject)

		if err != nil {
			return nil, err
		}
		for _, dsMoRef := range hostSystemManagedObject.Datastore {

			if importOvaParams.Datastore.Reference().Value == dsMoRef.Value {
				foundDatastore = true
				break
			}
		}
		for _, nMoRef := range hostSystemManagedObject.Network {
			if importOvaParams.Network.Reference().Value == nMoRef.Value {
				foundNetwork = true
				break
			}
		}

		if foundDatastore && foundNetwork {
			importOvaParams.Host = hostObj
			resourcePool, err := hostObj.ResourcePool(ctx)
			if err != nil {
				return nil, err
			}
			importOvaParams.ResourcePool = resourcePool
		}
	}
	if !foundDatastore {
		return nil, errors.Errorf("The hosts in the cluster do not have the datastore provided in install-config.yaml")
	}
	if !foundNetwork {
		return nil, errors.Errorf("The hosts in the cluster do not have the network provided in install-config.yaml")
	}

	return importOvaParams, nil
}

func resourceVSpherePrivateImportOvaCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] : Beginning create")

	client := meta.(*VSphereClient).vimClient.Client

	importParams, err := findImportOvaParams(client, d.Get("datacenter").(string), d.Get("cluster").(string), d.Get("datastore").(string), d.Get("network").(string))

	if err != nil {
		return err
	}
	spew.Dump(importParams)

	return resourceVSpherePrivateImportOvaRead(d, meta)
}

func resourceVSpherePrivateImportOvaRead(d *schema.ResourceData, meta interface{}) error {

	return nil
}

func resourceVSpherePrivateImportOvaUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceVSpherePrivateImportOvaDelete(d *schema.ResourceData, meta interface{}) error {

	return nil
}

func resourceVSpherePrivateImportOvaImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	return nil, nil
}
