package vsphereprivate

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"

	"github.com/pkg/errors"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/nfc"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/ovf"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
)

func resourceVSpherePrivateImportOva() *schema.Resource {
	return &schema.Resource{
		Create:        resourceVSpherePrivateImportOvaCreate,
		Read:          resourceVSpherePrivateImportOvaRead,
		Update:        resourceVSpherePrivateImportOvaUpdate,
		Delete:        resourceVSpherePrivateImportOvaDelete,
		SchemaVersion: 1,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Description:  "",
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
			},
			"filename": {
				Type:         schema.TypeString,
				Description:  "",
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
			},
			"datacenter": {
				Type:        schema.TypeString,
				Description: "The ID of the datacenter. Can be ignored if creating a datacenter folder, otherwise required.",
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
		return nil, errors.Errorf("failed to find a host in the cluster that contains the provided datastore")
	}
	if !foundNetwork {
		return nil, errors.Errorf("failed to find a host in the cluster that contains the provided network")
	}

	return importOvaParams, nil
}

// Used govc/importx/ovf.go as an example to implement
// resourceVspherePrivateImportOvaCreate and upload functions
// See: https://github.com/vmware/govmomi/blob/master/govc/importx/ovf.go#L196-L324

func upload(ctx context.Context, archive *ArchiveFlag, lease *nfc.Lease, item nfc.FileItem) error {
	file := item.Path

	f, size, err := archive.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	opts := soap.Upload{
		ContentLength: size,
	}

	return lease.Upload(ctx, item, f, opts)
}

func resourceVSpherePrivateImportOvaCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: Beginning import ova create", d.Get("filename").(string))

	ctx := context.TODO()
	client := meta.(*VSphereClient).vimClient.Client
	ovaTapeArchive := &TapeArchive{Path: d.Get("filename").(string)}
	archive := &ArchiveFlag{}
	archive.Archive = ovaTapeArchive

	importOvaParams, err := findImportOvaParams(client,
		d.Get("datacenter").(string),
		d.Get("cluster").(string),
		d.Get("datastore").(string),
		d.Get("network").(string))
	if err != nil {
		return err
	}

	ovfDescriptor, err := archive.ReadOvf("*.ovf")
	if err != nil {
		return errors.Errorf("failed to read ovf: %s", err)
	}

	ovfEnvelope, err := archive.ReadEnvelope(ovfDescriptor)
	if err != nil {
		return errors.Errorf("failed to parse ovf: %s", err)
	}

	// The RHCOS OVA only has one network defined by default
	// The OVF envelope defines this.  We need a 1:1 mapping
	// between networks with the OVF and the host
	if len(ovfEnvelope.Network.Networks) != 1 {
		return errors.Errorf("Expected the OVA to only have a single network adapter")
	}
	// Create mapping between OVF and the network object
	// found by Name
	networkMappings := []types.OvfNetworkMapping{{
		Name:    ovfEnvelope.Network.Networks[0].Name,
		Network: importOvaParams.Network.Reference(),
	}}
	// This is a very minimal spec for importing
	// an OVF.
	cisp := types.OvfCreateImportSpecParams{
		EntityName:     d.Get("name").(string),
		NetworkMapping: networkMappings,
	}

	m := ovf.NewManager(client)
	spec, err := m.CreateImportSpec(ctx,
		string(ovfDescriptor),
		importOvaParams.ResourcePool.Reference(),
		importOvaParams.Datastore.Reference(),
		cisp)

	if err != nil {
		return errors.Errorf("failed to create import spec: %s", err)
	}
	if spec.Error != nil {
		return errors.New(spec.Error[0].LocalizedMessage)
	}

	//Creates a new entity in this resource pool.
	//See VMware vCenter API documentation: Managed Object - ResourcePool - ImportVApp
	lease, err := importOvaParams.ResourcePool.ImportVApp(ctx,
		spec.ImportSpec,
		importOvaParams.Folder,
		importOvaParams.Host)

	if err != nil {
		return errors.Errorf("failed to import vapp: %s", err)
	}

	info, err := lease.Wait(ctx, spec.FileItem)
	if err != nil {
		return errors.Errorf("failed to lease wait: %s", err)
	}

	u := lease.StartUpdater(ctx, info)
	defer u.Done()

	for _, i := range info.Items {
		// upload the vmdk to which ever host that was first
		// available with the required network and datastore.
		err = upload(ctx, archive, lease, i)
		if err != nil {
			return errors.Errorf("failed to upload: %s", err)
		}
	}

	err = lease.Complete(ctx)
	if err != nil {
		return errors.Errorf("failed to lease copmlete: %s", err)
	}

	d.SetId(info.Entity.Value)
	log.Printf("[DEBUG] %s: ova import complete", d.Get("name").(string))

	return resourceVSpherePrivateImportOvaRead(d, meta)
}

func resourceVSpherePrivateImportOvaRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*VSphereClient).vimClient.Client
	moRef := types.ManagedObjectReference{
		Value: d.Id(),
		Type:  "VirtualMachine",
	}

	vm := object.NewVirtualMachine(client, moRef)
	if vm == nil {
		return fmt.Errorf("VirtualMachine not found")
	}

	return nil
}

func resourceVSpherePrivateImportOvaUpdate(d *schema.ResourceData, meta interface{}) error {
	return resourceVSpherePrivateImportOvaRead(d, meta)
}

func resourceVSpherePrivateImportOvaDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: Beginning delete", d.Get("name").(string))
	ctx := context.TODO()

	client := meta.(*VSphereClient).vimClient.Client
	moRef := types.ManagedObjectReference{
		Value: d.Id(),
		Type:  "VirtualMachine",
	}

	vm := object.NewVirtualMachine(client, moRef)
	if vm == nil {
		return errors.Errorf("VirtualMachine not found")
	}

	task, err := vm.Destroy(ctx)
	if err != nil {
		return errors.Errorf("failed to destroy virtual machine %s", err)
	}

	err = task.Wait(ctx)
	if err != nil {
		return errors.Errorf("failed to destroy virtual machine %s", err)
	}

	d.SetId("")

	log.Printf("[DEBUG] %s: Delete complete", d.Get("name").(string))

	return nil
}
