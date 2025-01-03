package manager

import (
	"fmt"
	"os"

	"github.com/freakmaxi/kertish-dos/basics/common"
	"github.com/freakmaxi/kertish-dos/basics/errors"
	cluster2 "github.com/freakmaxi/kertish-dos/manager-node/cluster"
	"github.com/freakmaxi/kertish-dos/manager-node/data"
	"go.uber.org/zap"
)

// Cluster interface contains functions to handle the cluster administration in the dos farm
type Cluster interface {
	Register(nodeAddresses []string) (*common.Cluster, error)
	RegisterNodesTo(clusterId string, nodeAddresses []string) error

	UnRegisterCluster(clusterId string) error
	UnRegisterNode(nodeId string) error

	Handshake() error

	GetClusters() (common.Clusters, error)
	GetCluster(clusterId string) (*common.Cluster, error)

	Reserve(size uint64) (*common.ReservationMap, error)
	Commit(reservationId string, clusterMap map[string]uint64) error
	Discard(reservationId string) error

	MoveCluster(sourceClusterId string, targetClusterId string) error
	BalanceClusters(clusterIds []string) error
	ChangeState(clusterId string, state common.States) error
	ChangeStateAll(state common.States) error

	CreateSnapshot(clusterId string) error
	DeleteSnapshot(clusterId string, snapshotIndex uint64) error
	RestoreSnapshot(clusterId string, snapshotIndex uint64) error

	Map(sha512HexList []string, mapType common.MapType) (map[string][]string, error)
	Find(sha512Hex string, mapType common.MapType) (string, []string, error)
}

type cluster struct {
	clusters    data.Clusters
	index       data.Index
	synchronize Synchronize
	logger      *zap.Logger
}

// NewCluster creates the instance for cluster administration of the dos farm
func NewCluster(clusters data.Clusters, index data.Index, synchronize Synchronize, logger *zap.Logger) (Cluster, error) {
	return &cluster{
		clusters:    clusters,
		index:       index,
		synchronize: synchronize,
		logger:      logger,
	}, nil
}

func (c *cluster) Register(nodeAddresses []string) (*common.Cluster, error) {
	cluster := common.NewCluster(newClusterId())

	nodes, clusterSize, err := c.prepareNodes(nodeAddresses, 0)
	if err != nil {
		return nil, err
	}
	cluster.Size = clusterSize
	cluster.Nodes = append(cluster.Nodes, nodes...)

	masterAddress := ""
	for i, node := range cluster.Nodes {
		mA := masterAddress

		if i == 0 {
			node.Master = true
			masterAddress = node.Address
		}

		dn, err := cluster2.NewDataNode(node.Address)
		if err != nil {
			return nil, err
		}
		if !dn.Join(cluster.Id, node.Id, mA) {
			return nil, errors.ErrMode
		}
	}

	if err := c.clusters.RegisterCluster(cluster); err != nil {
		return nil, err
	}

	return cluster, nil
}

func (c *cluster) RegisterNodesTo(clusterId string, nodeAddresses []string) error {
	if err := c.clusters.Save(clusterId, func(cluster *common.Cluster) error {
		if cluster.Maintain {
			return errors.ErrMaintain
		}

		masterNode := cluster.Master()

		nodes, _, err := c.prepareNodes(nodeAddresses, cluster.Size)
		if err != nil {
			return err
		}
		cluster.Nodes = append(cluster.Nodes, nodes...)

		for _, node := range nodes {
			dn, err := cluster2.NewDataNode(node.Address)
			if err != nil {
				return err
			}

			if !dn.Join(clusterId, node.Id, masterNode.Address) {
				return errors.ErrJoin
			}
		}

		cluster.Maintain = true

		return nil
	}); err != nil {
		return err
	}

	c.synchronize.QueueCluster(clusterId, true, false)

	return nil
}

func (c *cluster) prepareNodes(nodeAddresses []string, clusterSize uint64) (common.NodeList, uint64, error) {
	nodeMap := make(map[string]*common.Node)
	for _, nodeAddress := range nodeAddresses {
		if _, has := nodeMap[nodeAddress]; has {
			return nil, 0, fmt.Errorf("node address entered twice")
		}

		node, err := cluster2.NewDataNode(nodeAddress)
		if err != nil {
			return nil, 0, err
		}

		if node.Ping() == -1 {
			return nil, 0, errors.ErrPing
		}

		size, err := node.Size()
		if err != nil {
			return nil, 0, err
		}

		if clusterSize > 0 && size != clusterSize {
			return nil, 0, fmt.Errorf("inconsistent size between master and slave")
		}
		clusterSize = size

		hardwareId, err := node.HardwareId()
		if err != nil {
			return nil, 0, err
		}

		nodeId := newNodeId(hardwareId, nodeAddress, clusterSize)
		if _, err := c.clusters.GetByNodeId(nodeId); err == nil || err != errors.ErrNotFound {
			if err == nil {
				err = errors.ErrRegistered
			}
			return nil, 0, err
		}

		nodeMap[nodeAddress] = &common.Node{
			Id:      nodeId,
			Address: nodeAddress,
			Master:  false,
		}
	}

	r := make(common.NodeList, 0)
	for _, v := range nodeMap {
		r = append(r, v)
	}

	return r, clusterSize, nil
}

func (c *cluster) UnRegisterCluster(clusterId string) error {
	if err := c.clusters.Save(clusterId, func(cluster *common.Cluster) error {
		if cluster.Maintain {
			return errors.ErrMaintain
		}

		cluster.State = common.StateOffline
		cluster.Maintain = true

		return nil
	}); err != nil {
		return err
	}

	return c.clusters.UnregisterCluster(clusterId, func(cluster *common.Cluster) error {
		for _, node := range cluster.Nodes {
			dn, err := cluster2.NewDataNode(node.Address)
			if err != nil {
				continue
			}
			dn.Wipe()
		}
		return nil
	})
}

func (c *cluster) UnRegisterNode(nodeId string) error {
	return c.clusters.UnregisterNode(
		nodeId,
		func(cluster *common.Cluster) error {
			return c.synchronize.Cluster(cluster.Id, true, false, false)
		},
		func(deletingNode *common.Node) error {
			dn, err := cluster2.NewDataNode(deletingNode.Address)
			if err != nil || !dn.Leave() {
				return errors.ErrMode
			}
			dn.Wipe()
			return nil
		},
		func(newMaster *common.Node) error {
			dn, err := cluster2.NewDataNode(newMaster.Address)
			if err != nil || !dn.Mode(true) {
				return errors.ErrMode
			}
			return nil
		})
}

func (c *cluster) Handshake() error {
	clusters, err := c.clusters.GetAll()
	if err != nil {
		return err
	}

	hasJoinError := false

	for _, cluster := range clusters {
		masterNode := cluster.Master()

		mdn, err := cluster2.NewDataNode(masterNode.Address)
		if err != nil || !mdn.Join(cluster.Id, masterNode.Id, "") {
			c.logger.Error(
				"Syncing error: master node is not accessible",
				zap.String("clusterId", cluster.Id),
				zap.String("nodeId", masterNode.Id),
				zap.String("nodeAddress", masterNode.Address),
				zap.Error(errors.ErrJoin),
			)
			hasJoinError = true
			continue
		}

		slaveNodes := cluster.Slaves()

		if len(slaveNodes) == 0 {
			continue
		}

		for _, slaveNode := range slaveNodes {
			sdn, err := cluster2.NewDataNode(slaveNode.Address)
			if err != nil || !sdn.Join(cluster.Id, slaveNode.Id, masterNode.Address) {
				c.logger.Error(
					"Syncing error: slave node is not accessible",
					zap.String("clusterId", cluster.Id),
					zap.String("nodeId", slaveNode.Id),
					zap.String("nodeAddress", slaveNode.Address),
					zap.Error(errors.ErrJoin),
				)
				hasJoinError = true
				continue
			}
		}
	}

	if hasJoinError {
		return errors.ErrJoin
	}
	return nil
}

func (c *cluster) GetClusters() (common.Clusters, error) {
	return c.clusters.GetAll()
}

func (c *cluster) GetCluster(clusterId string) (*common.Cluster, error) {
	return c.clusters.Get(clusterId)
}

func (c *cluster) Reserve(size uint64) (*common.ReservationMap, error) {
	var reservationMap *common.ReservationMap

	if err := c.clusters.SaveAll(func(clusters common.Clusters) error {
		var err error
		reservationMap, err = c.createReservationMap(size, clusters)

		return err
	}); err != nil {
		return nil, err
	}

	return reservationMap, nil
}

func (c *cluster) Commit(reservationId string, clusterMap map[string]uint64) error {
	return c.clusters.SaveAll(func(clusters common.Clusters) error {
		for _, cluster := range clusters {
			v, has := clusterMap[cluster.Id]
			if !has {
				v = 0
			}
			cluster.Commit(reservationId, v)
		}
		return nil
	})
}

func (c *cluster) Discard(reservationId string) error {
	return c.clusters.SaveAll(func(clusters common.Clusters) error {
		for _, cluster := range clusters {
			cluster.Discard(reservationId)
		}
		return nil
	})
}

func (c *cluster) MoveCluster(sourceClusterId string, targetClusterId string) error {
	move := newMove(c.clusters, c.index, c.synchronize, c.logger)
	return move.Move(sourceClusterId, targetClusterId)
}

func (c *cluster) BalanceClusters(clusterIds []string) error {
	balance := newBalance(c.clusters, c.index, c.synchronize, c.logger)
	return balance.Balance(clusterIds)
}

func (c *cluster) ChangeState(clusterId string, state common.States) error {
	return c.clusters.Save(clusterId, func(cluster *common.Cluster) error {
		if cluster.Maintain {
			return errors.ErrMaintain
		}
		cluster.State = state
		return nil
	})
}

func (c *cluster) ChangeStateAll(state common.States) error {
	return c.clusters.SaveAll(func(clusters common.Clusters) error {
		for _, cluster := range clusters {
			if cluster.Maintain {
				return errors.ErrMaintain
			}
			cluster.State = state
		}
		return nil
	})
}

func (c *cluster) CreateSnapshot(clusterId string) error {
	cluster, err := c.clusters.Get(clusterId)
	if err != nil {
		return err
	}

	if cluster.Maintain {
		return errors.ErrMaintain
	}
	if err := c.clusters.UpdateMaintain(cluster.Id, true, common.TopicCreateSnapshot); err != nil {
		return err
	}

	masterNode := cluster.Master()
	dn, err := cluster2.NewDataNode(masterNode.Address)
	if err != nil {
		return err
	}

	if !dn.SnapshotCreate() {
		return errors.ErrSnapshot
	}

	return c.synchronize.Cluster(cluster.Id, true, false, false)
}

func (c *cluster) DeleteSnapshot(clusterId string, snapshotIndex uint64) error {
	cluster, err := c.clusters.Get(clusterId)
	if err != nil {
		return err
	}

	if cluster.Maintain {
		return errors.ErrMaintain
	}
	if err := c.clusters.UpdateMaintain(cluster.Id, true, common.TopicDeleteSnapshot); err != nil {
		return err
	}

	masterNode := cluster.Master()
	dn, err := cluster2.NewDataNode(masterNode.Address)
	if err != nil {
		var _ = c.clusters.UpdateMaintain(cluster.Id, false, common.TopicNone)
		return err
	}

	if !dn.SnapshotDelete(snapshotIndex) {
		var _ = c.clusters.UpdateMaintain(cluster.Id, false, common.TopicNone)
		return errors.ErrSnapshot
	}

	return c.synchronize.Cluster(cluster.Id, true, false, false)
}

func (c *cluster) RestoreSnapshot(clusterId string, snapshotIndex uint64) error {
	cluster, err := c.clusters.Get(clusterId)
	if err != nil {
		return err
	}

	if cluster.Maintain {
		return errors.ErrMaintain
	}
	if err := c.clusters.UpdateMaintain(cluster.Id, true, common.TopicRestoreSnapshot); err != nil {
		return err
	}

	masterNode := cluster.Master()
	dn, err := cluster2.NewDataNode(masterNode.Address)
	if err != nil {
		return err
	}

	if !dn.SnapshotRestore(snapshotIndex) {
		return errors.ErrSnapshot
	}

	return c.synchronize.Cluster(cluster.Id, true, false, false)
}

func (c *cluster) Map(sha512HexList []string, mapType common.MapType) (map[string][]string, error) {
	clusterMapping := make(map[string][]string)
	for _, sha512Hex := range sha512HexList {
		_, addresses, err := c.Find(sha512Hex, mapType)
		if err != nil {
			if err == os.ErrNotExist && mapType == common.MTDelete {
				continue
			}
			return nil, err
		}
		clusterMapping[sha512Hex] = addresses
	}
	return clusterMapping, nil
}

func (c *cluster) Find(sha512Hex string, mapType common.MapType) (string, []string, error) {
	cacheFileItem, err := c.index.Get(sha512Hex)
	if err != nil {
		return "", nil, err
	}

	cluster, err := c.clusters.Get(cacheFileItem.ClusterId)
	if err != nil {
		return "", nil, err
	}

	// if it is a read request, try other nodes even the cluster is paralyzed.
	// maybe there is no master candidate but other nodes can contain the
	// requested file chunk to provide
	if cluster.State == common.StateOffline {
		return "", nil, errors.ErrNoAvailableActionNode
	}
	if !cluster.CanSchedule() && mapType != common.MTRead {
		return "", nil, errors.ErrNoAvailableActionNode
	}

	// addresses should always contain node address, if it will be empty or nil
	// just return the error
	addresses := make([]string, 0)

	switch mapType {
	case common.MTRead:
		nodes := cluster.PrioritizedHighQualityNodes(cacheFileItem.ExistsIn)
		if nodes == nil {
			return "", nil, errors.ErrNoAvailableActionNode
		}
		for _, n := range nodes {
			addresses = append(addresses, n.Address)
		}
	default:
		node := cluster.Master()
		if node == nil {
			return "", nil, errors.ErrNoAvailableActionNode
		}
		addresses = []string{node.Address}
	}

	return cluster.Id, addresses, nil
}

var _ Cluster = &cluster{}
