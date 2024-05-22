package k3s

import (
	"errors"
	"math"

	sanitization "polyqube/pkg/utils"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	DefaultControlPlaneCount = uint8(3)
	DefaultWorkerCount       = uint8(2)
)

var (
	ErrEmptyName    = errors.New("name cannot be empty")
	ErrEmptyimageId = errors.New("imageId cannot be empty")
)

type ClusterOpts struct {
	name                 string
	numControlPlaneNodes uint8
	numWorkerNodes       uint8
	imageId              string
}

type clusterOptsBuilder struct {
	name                 string
	numControlPlaneNodes uint8
	numWorkerNodes       uint8
	imageId              string
}

// Create a set of options for defining a cluster
//
// Note that the cluster name must be equal to the stack name
func NewClusterOpts() *clusterOptsBuilder {
	opts := &clusterOptsBuilder{
		name:                 "",
		numControlPlaneNodes: DefaultControlPlaneCount,
		numWorkerNodes:       DefaultWorkerCount,
		imageId:              "",
	}
	return opts
}

// Name of the cluster. Must be exactly the same as the stack name for the cluster
func (b *clusterOptsBuilder) Name(val string) *clusterOptsBuilder {
	b.name = val
	return b
}

// Set number of control plane nodes to deploy
func (b *clusterOptsBuilder) ControlPlaneNodes(val uint8) *clusterOptsBuilder {
	val = sanitization.Clamp(val, 3, math.MaxUint8)
	b.numControlPlaneNodes = val
	return b
}

// Set number of cloud worker nodes to deploy
func (b *clusterOptsBuilder) WorkerNodes(val uint8) *clusterOptsBuilder {
	val = sanitization.Clamp(val, 2, math.MaxUint8)
	b.numWorkerNodes = val
	return b
}

// Set imageId for ControlPlane and Worker nodes to use
func (b *clusterOptsBuilder) ImageId(val string) *clusterOptsBuilder {
	b.imageId = val
	return b
}

// Create ClusterOpts that defines all configurations for a new cluster
func (b *clusterOptsBuilder) Build() (*ClusterOpts, error) {
	if b.name == "" {
		return nil, ErrEmptyName
	}
	if b.imageId == "" {
		return nil, ErrEmptyimageId
	}

	opts := &ClusterOpts{
		name:                 b.name,
		imageId:              b.imageId,
		numControlPlaneNodes: b.numControlPlaneNodes,
		numWorkerNodes:       b.numWorkerNodes,
	}
	return opts, nil
}

// TODO: Create a general interface for building cloud specific clusters
type Cluster interface {
	Create(pulumi.Context) error
}
