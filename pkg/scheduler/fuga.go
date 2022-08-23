package scheduler

import (
	"fmt"

	"github.com/apache/yunikorn-core/pkg/common/resources"
	"github.com/apache/yunikorn-core/pkg/log"
	"github.com/apache/yunikorn-core/pkg/scheduler/objects"
)

const (
	eta   = 3
	alpha = 2
)

var numApps int
var numNodes int

var allApps []*objects.Application
var allRequests []*resources.Resource
var allNodes []*objects.Node

type combination map[int]int64
type allocationDecision map[int]*combination

type strategiesSet struct {
	coms     []*combination // coms[1 ~ eta][1 ~ s]
	min      float64
	minIndex int
	num      int
}

func u(com *combination, restype string, m int) float64 {
	// node := allNodes[m]
	// x := 0.0
	// for i, num := range com {
	// 	res := allRequests[i]
	// 	x += float64(res.Resources[restype]) * float64(num)
	// }
	// remres := n.GetAvailableResource().Resources
	// initres := n.GetCapacity().Resources
	// return (float64(remres[restype]) - x) / float64(initres[restype])
	return 0.0
}

func minu(com *combination, m int) float64 {
	temp1 := u(com, resources.MEMORY, m)
	temp2 := u(com, resources.VCORE, m)
	if temp1 > temp2 {
		return temp1
	}
	return temp2
}

func (ss *strategiesSet) shouldAdd(com *combination, m int) bool {
	if ss.num < eta {
		return true
	}
	if ss.min < minu(com, m) {
		return true
	}
	return false
}

func (ss *strategiesSet) push(com *combination, m int) {
	if ss.num < eta {
		ss.coms[ss.num] = com
		ss.num++
	} else {
		ss.coms[ss.minIndex] = com
		var min float64
		var minIndex int
		for i := 0; i < eta; i++ {
			if i == 0 || minu(ss.coms[i], m) > min { // TODO: should be better
				minIndex = i
				min = minu(ss.coms[i], m)
			}
		}
		ss.min = min
		ss.minIndex = minIndex
	}
}

func initGame(apps []*objects.Application, nodes []*objects.Node) error {
	numApps = len(apps)
	numNodes = len(nodes)
	if numApps < 1 {
		return fmt.Errorf("numApps < 1")
	}
	if numNodes < 1 {
		return fmt.Errorf("numNodes < 1")
	}
	allApps = apps
	allNodes = nodes
	allRequests = make([]*resources.Resource, numApps)
	for i, app := range allApps {
		asks := app.GetAllRequests()
		for _, ask := range asks {
			if ask != nil {
				allRequests[i] = ask.GetAllocatedResource()
				break
			}
		}
		if allRequests[i] == nil {
			return fmt.Errorf("allRequests[i] == nil")
		}
	}
	return nil
}

func check(node *objects.Node, com combination) bool {
	r := resources.NewResource()
	for appID, num := range com {
		app := allApps[appID]
		asks := app.GetAllRequests()
		var req *resources.Resource
		for _, ask := range asks {
			req = ask.GetAllocatedResource()
			break
		}
		r = resources.Add(r, resources.Multiply(req, num))
	}
	if node.GetAvailableResource().FitInMaxUndef(r) {
		return true
	} else {
		return false
	}
}

var ss *strategiesSet

func find(com combination, l int, m int) {
	node := allNodes[m]
	if !check(node, com) {
		return
	}

	if ss.shouldAdd(&com, m) { // TODO
		// addToSs
		newCom := com
		ss.push(&newCom, m)
	}

	for i := l; i < numApps; i++ {
		com[i]++
		if com[i] > int64(len(allApps[i].GetAllRequests())) {
			com[i]--
			continue
		}
		find(com, i, m)
		com[i]--
	}
}

func getStrategies(m int) (*strategiesSet, error) {
	ss = &strategiesSet{
		coms:     make([]*combination, eta),
		min:      0.0,
		minIndex: -1,
		num:      0,
	}
	com := make(combination)
	for i := 0; i < numApps; i++ {
		com[i] = 0
	}
	l := 0
	find(com, l, m)

	ret := ss
	return ret, nil
}

func G() (*allocationDecision, error) {
	var err error

	// Step 2: Strategies Set for Each Player
	o := make([]*strategiesSet, numNodes)
	for m := 0; m < numNodes; m++ {
		o[m], err = getStrategies(m)
		if err != nil {
			return nil, err
		}
	}

	// Step 3: Generate the Extension-form Game Tree
	// Step 4: Find the SPNE for a game G

	return nil, nil
}

func fuga(apps []*objects.Application, nodes []*objects.Node) error {
	log.Logger().Info("enter fuga()")

	// Step 1: Pre-combination Phase
	var players []*objects.Node
	for _, n := range nodes {
		if resources.StrictlyGreaterThanZero(n.GetAvailableResource()) {
			players = append(players, n)
		}
	}
	err := initGame(apps, players)
	if err != nil {
		return err
	}

	ad, err := G()
	if ad != nil {
		log.Logger().Info(fmt.Sprintf("ad: %+v\n", ad))
		log.Logger().Info("haha")
	}
	return nil
}
