package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type addNode struct {
	clusterId string
	addresses []string
}

func (a *addNode) String() string {
	if a == nil {
		return ""
	}
	return fmt.Sprintf("%s=%s", a.clusterId, strings.Join(a.addresses, ","))
}

func (a *addNode) Set(value string) error {
	eqIdx := strings.Index(value, "=")
	if eqIdx == -1 {
		return fmt.Errorf("input is not suitable")
	}

	ana := strings.Split(value[eqIdx+1:], ",")
	if len(ana) > 0 && len(ana[0]) == 0 {
		ana = []string{}
	}

	a.clusterId = value[:eqIdx]
	a.addresses = ana

	return nil
}

type flagContainer struct {
	managerAddress     string
	createCluster      []string
	deleteCluster      string
	moveCluster        []string
	balanceClusters    []string
	balanceAllClusters bool
	repairConsistency  string
	addNode            addNode
	removeNode         string
	createSnapshot     string
	deleteSnapshot     string
	restoreSnapshot    string
	changeState        []string
	changeStateAll     bool
	stateOnline        bool
	stateReadonly      bool
	stateOffline       bool
	syncCluster        string
	syncClusters       bool
	clustersReport     bool
	getCluster         string
	getClusters        bool
	help               bool
	version            bool

	active string
}

func (f *flagContainer) Define(v string) int {
	if f.help {
		fmt.Printf("Kertish-dos Admin (v%s) usage: \n", v)
		fmt.Println()

		return 1
	}

	if f.version {
		f.active = "version"
		return 0
	}

	activeCount := 0
	if len(f.createCluster) != 0 {
		activeCount++
		f.active = "createCluster"
	}

	if len(f.deleteCluster) != 0 {
		activeCount++
		f.active = "deleteCluster"
	}

	if len(f.moveCluster) != 0 {
		if len(f.moveCluster) != 2 {
			fmt.Println("you should define source and target cluster ids")
			fmt.Println()
			return 1
		}
		activeCount++
		f.active = "moveCluster"
	}

	if len(f.balanceClusters) != 0 || f.balanceAllClusters {
		if len(f.balanceClusters) == 1 {
			fmt.Println("you should define at least two cluster id or leave empty for all")
			fmt.Println()
			return 1
		}
		activeCount++
		f.active = "balanceClusters"
	}

	if len(f.repairConsistency) != 0 {
		activeCount++
		f.active = "repairConsistency"
	}

	if len(f.addNode.clusterId) > 0 && len(f.addNode.addresses) > 0 {
		activeCount++
		f.active = "addNode"
	}

	if len(f.removeNode) != 0 {
		activeCount++
		f.active = "removeNode"
	}

	if len(f.createSnapshot) != 0 {
		activeCount++
		f.active = "createSnapshot"
	}

	if len(f.deleteSnapshot) != 0 {
		paramTest := f.deleteSnapshot

		eqIdx := strings.Index(paramTest, "=")
		if eqIdx == -1 {
			fmt.Println("you should define the snapshot index for the cluster")
			fmt.Println()
			return 1
		}

		clusterId := paramTest[:eqIdx]
		if len(clusterId) == 0 {
			fmt.Println("you should define the target cluster id")
			fmt.Println()
			return 1
		}

		_, err := strconv.ParseUint(paramTest[eqIdx+1:], 10, 64)
		if err != nil {
			fmt.Println("snapshot index should be 0 or positive numeric value")
			fmt.Println()
			return 1
		}

		activeCount++
		f.active = "deleteSnapshot"
	}

	if len(f.restoreSnapshot) != 0 {
		paramTest := f.restoreSnapshot

		eqIdx := strings.Index(paramTest, "=")
		if eqIdx == -1 {
			fmt.Println("you should define the snapshot index for the cluster")
			fmt.Println()
			return 1
		}

		clusterId := paramTest[:eqIdx]
		if len(clusterId) == 0 {
			fmt.Println("you should define the target cluster id")
			fmt.Println()
			return 1
		}

		_, err := strconv.ParseUint(paramTest[eqIdx+1:], 10, 64)
		if err != nil {
			fmt.Println("snapshot index should be 0 or positive numeric value")
			fmt.Println()
			return 1
		}

		activeCount++
		f.active = "restoreSnapshot"
	}

	if len(f.changeState) != 0 || f.changeStateAll {
		activeCount++
		f.active = "changeState"
	}

	if len(f.syncCluster) > 0 {
		activeCount++
		f.active = "syncClusters"
	}

	if f.syncClusters {
		activeCount++
		f.active = "syncClusters"
	}

	if f.clustersReport {
		activeCount++
		f.active = "clustersReport"
	}

	if len(f.getCluster) > 0 {
		activeCount++
		f.active = "getCluster"
	}

	if f.getClusters {
		activeCount++
		f.active = "getClusters"
	}

	if activeCount == 0 {
		fmt.Printf("Kertish-dos Admin (v%s) usage: \n", v)
		fmt.Println()

		return 1
	}

	if activeCount > 1 {
		fmt.Println("accepts only one operation request at a time")

		return 2
	}

	return 0
}

func defineFlags(v string) *flagContainer {
	set := flag.NewFlagSet("dos", flag.ContinueOnError)

	var managerAddress string
	set.StringVar(&managerAddress, `manager-address`, "localhost:9400", `(DEPRECATED) The end point of manager to work with.`)

	var targetAddress string
	set.StringVar(&targetAddress, `target`, "localhost:9400", `The end point of manager to work with.`)

	var t string
	set.StringVar(&t, `t`, "localhost:9400", `The end point of manager to work with.`)

	var createCluster string
	set.StringVar(&createCluster, `create-cluster`, "", `Creates data nodes cluster. Provide data node binding addresses to create cluster. Node Manager will decide which data node will be master and which others are slave.
Ex: 192.168.0.1:9430,192.168.0.2:9430`)

	var deleteCluster string
	set.StringVar(&deleteCluster, `delete-cluster`, "", `Deletes data nodes cluster. Provide cluster id to delete.`)

	var moveCluster string
	set.StringVar(&moveCluster, `move-cluster`, "", `Moves cluster data between clusters. Provide cluster source and target ids to move cluster.
Ex: sourceClusterId,targetClusterId`)

	var balanceClusters string
	set.StringVar(&balanceClusters, `balance-clusters`, "", `Balance data weight between clusters. Provide at least two cluster ids to balance the data between or leave empty to apply all clusters in the setup.
Ex: clusterId,clusterId`)

	var getCluster string
	set.StringVar(&getCluster, `get-cluster`, "", `Gets and prints cluster information.`)

	set.Bool(`get-clusters`, false, `Gets and prints all clusters information.`)

	addNode := addNode{}
	set.Var(&addNode, `add-node`, `Adds more nodes to the existent cluster. Node Manager will decide for the priority of data nodes.
Ex: clusterId=192.168.0.1:9430,192.168.0.2:9430`)

	var removeNode string
	set.StringVar(&removeNode, `remove-node`, "", `Removes the node from its cluster.`)

	var repairConsistency string
	set.StringVar(&repairConsistency, `repair-consistency`, "", `Repair file chunk node distribution consistency in metadata and data nodes and mark as zombie for the broken ones. Provide repair model for consistency repairing operation or leave empty to run full repair. Possible repair models (full, structure, structure+integrity, integrity, integrity+checksum, checksum, checksum+rebuild)`)

	var createSnapshot string
	set.StringVar(&createSnapshot, `create-snapshot`, "", `Creates snapshot on a cluster. Provide cluster id to create snapshot.`)

	var deleteSnapshot string
	set.StringVar(&deleteSnapshot, `delete-snapshot`, "", `Deletes a snapshot on a cluster. Provide cluster id with snapshot index to be deleted.
Ex: clusterId=snapshotIndex`)

	var restoreSnapshot string
	set.StringVar(&restoreSnapshot, `restore-snapshot`, "", `Restores a snapshot in the cluster. Provide cluster id with snapshot index to be restored.
Ex: clusterId=snapshotIndex`)

	var changeState string
	set.StringVar(&changeState, `change-state`, "", `Change the state of the cluster. Provide at least one cluster id to change the state or leave empty to apply all clusters in the setup.
Ex: clusterId,clusterId`)

	set.Bool(`online`, false, `Change the state of the cluster to ONLINE. (Can only be used with -change-state argument)`)
	set.Bool(`readonly`, false, `Change the state of the cluster to READONLY. (Can only be used with -change-state argument)`)
	set.Bool(`offline`, false, `Change the state of the cluster to OFFLINE. (Can only be used with -change-state argument)`)

	var syncCluster string
	set.StringVar(&syncCluster, `sync-cluster`, "", `Synchronise selected cluster and their nodes for data consistency.`)

	set.Bool(`sync-clusters`, false, `Synchronise all clusters and their nodes for data consistency.`)
	set.Bool(`clusters-report`, false, `Gets clusters health report.`)
	set.Bool(`help`, false, `Print this usage documentation`)
	set.Bool(`h`, false, `Print this usage documentation`)
	set.Bool(`version`, false, `Print release version`)
	set.Bool(`v`, false, `Print release version`)

	args := os.Args[1:]
	for i, arg := range args {
		idx := strings.Index(arg, "-balance-clusters")
		if idx == -1 {
			continue
		}
		if len(args) > i+1 && !strings.HasPrefix(args[i+1], "-") {
			break
		}
		args = insert(args, i, "*")
		break
	}

	for i, arg := range args {
		idx := strings.Index(arg, "-repair-consistency")
		if idx == -1 {
			continue
		}
		if len(args) > i+1 && !strings.HasPrefix(args[i+1], "-") {
			break
		}
		args = insert(args, i, "full")
		break
	}

	for i, arg := range args {
		idx := strings.Index(arg, "-change-state")
		if idx == -1 {
			continue
		}
		if len(args) > i+1 && !strings.HasPrefix(args[i+1], "-") {
			break
		}
		args = insert(args, i, "*")
		break
	}

	_ = set.Parse(args)

	if strings.Compare(managerAddress, "localhost:9400") == 0 {
		managerAddress = targetAddress
	}

	if strings.Compare(managerAddress, "localhost:9400") == 0 {
		managerAddress = t
	}

	cc := strings.Split(createCluster, ",")
	if len(cc) > 0 && len(cc[0]) == 0 {
		cc = []string{}
	}

	mc := strings.Split(moveCluster, ",")
	if len(mc) != 2 || len(mc) == 2 && len(mc[0]) == 0 && len(mc[1]) == 0 {
		mc = []string{}
	}

	bac := false
	bc := strings.Split(balanceClusters, ",")
	if len(bc) > 0 && len(bc[0]) == 0 || strings.Compare(bc[0], "*") == 0 {
		bac = strings.Compare(bc[0], "*") == 0
		bc = []string{}
	}

	csa := false
	cs := strings.Split(changeState, ",")
	if len(cs) > 0 && len(cs[0]) == 0 || strings.Compare(cs[0], "*") == 0 {
		csa = strings.Compare(cs[0], "*") == 0
		cs = []string{}
	}

	joinedArgs := strings.Join(os.Args, " ")

	fc := &flagContainer{
		managerAddress:     managerAddress,
		createCluster:      cc,
		deleteCluster:      deleteCluster,
		moveCluster:        mc,
		balanceClusters:    bc,
		balanceAllClusters: bac,
		repairConsistency:  repairConsistency,
		addNode:            addNode,
		removeNode:         removeNode,
		createSnapshot:     createSnapshot,
		deleteSnapshot:     deleteSnapshot,
		restoreSnapshot:    restoreSnapshot,
		changeState:        cs,
		changeStateAll:     csa,
		stateOnline:        strings.Contains(joinedArgs, "online"),
		stateReadonly:      strings.Contains(joinedArgs, "readonly"),
		stateOffline:       strings.Contains(joinedArgs, "offline"),
		syncCluster:        syncCluster,
		syncClusters:       strings.Contains(joinedArgs, "sync-clusters"),
		clustersReport:     strings.Contains(joinedArgs, "clusters-report"),
		getCluster:         getCluster,
		getClusters:        strings.Contains(joinedArgs, "get-clusters"),
		help:               strings.Contains(joinedArgs, "-help") || strings.Contains(joinedArgs, "-h"),
		version:            strings.Contains(joinedArgs, "-version") || strings.Contains(joinedArgs, "-v"),
	}

	switch fc.Define(v) {
	case 1:
		set.PrintDefaults()
		os.Exit(0)
	case 2:
		os.Exit(2)
	}

	return fc
}

func insert(target []string, index int, item string) []string {
	temp := make([]string, index+1)
	copy(temp, target[:index+1])
	temp = append(temp, item)
	return append(temp, target[index+1:]...)
}
