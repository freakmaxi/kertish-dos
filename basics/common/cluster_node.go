package common

import "time"

const leadDuration = time.Minute * 5 // 5 minutes

// Node struct is to hold the node details of the dos cluster
type Node struct {
	Id       string    `json:"nodeId"`
	Address  string    `json:"address"`
	Master   bool      `json:"master"`
	LeadTill time.Time `json:"leadTill"`
	Quality  int64     `json:"quality"`
}

func (n *Node) LeadershipExpired() bool {
	return time.Now().UTC().After(n.LeadTill)
}

func (n *Node) SetLeadDuration() {
	if !n.Master {
		n.LeadTill = time.Now().UTC()
		return
	}
	n.LeadTill = time.Now().UTC().Add(leadDuration)
}

// NodeList is the definition of the pointer array of Node struct
type NodeList []*Node

func (n NodeList) Len() int           { return len(n) }
func (n NodeList) Less(i, _ int) bool { return n[i].Master }
func (n NodeList) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }

// PrioritizedHighQualityNodeList is the definition of the pointer array of Node struct
type PrioritizedHighQualityNodeList []*Node

func (n PrioritizedHighQualityNodeList) Len() int           { return len(n) }
func (n PrioritizedHighQualityNodeList) Less(i, j int) bool { return n[i].Quality < n[j].Quality }
func (n PrioritizedHighQualityNodeList) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }
