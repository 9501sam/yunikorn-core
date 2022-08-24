package scheduler

import (
	"fmt"
	"math"

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

var Cj map[string]float64
var lamda float64

type combination map[int]int64
type allocationDecision map[int]*combination

type strategiesSet struct {
	coms     []*combination // coms[1 ~ eta][1 ~ s]
	min      float64
	minIndex int
	num      int
}

func u(com *combination, restype string, m int) float64 {
	node := allNodes[m]
	x := 0.0
	for i, num := range *com {
		res := allRequests[i]
		x += float64(res.Resources[restype]) * float64(num)
	}
	remres := node.GetAvailableResource().Resources
	initres := node.GetCapacity().Resources
	return 1.0 - ((float64(remres[restype]) - x) / float64(initres[restype]))
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
	log.Logger().Info(fmt.Sprintf("fuga: enter push()"))
	log.Logger().Info(fmt.Sprintf("fuga: com = %+v, minu(com, m) = %f", com, minu(com, m)))

	// clone com
	newCom := make(combination)
	for k, v := range *com {
		newCom[k] = v
	}
	if ss.num < eta {
		ss.coms[ss.num] = &newCom
		ss.num++
	} else {
		ss.coms[ss.minIndex] = &newCom
	}
	var min float64
	var minIndex int
	for i := 0; i < eta && i < ss.num; i++ {
		if i == 0 || minu(ss.coms[i], m) < min { // TODO: should be better
			minIndex = i
			min = minu(ss.coms[i], m)
		}
	}
	ss.min = min
	ss.minIndex = minIndex

	// log.Logger().Info(fmt.Sprintf("fuga: ss.coms = %+v", ss.coms))
	for i := 0; i < eta; i++ {
		com := ss.coms[i]
		if com == nil {
			break
		}
		log.Logger().Info(fmt.Sprintf("fuga: %dth combination: com = %+v, minu(com, m) = %f", i, com, minu(com, m)))
	}
	log.Logger().Info(fmt.Sprintf("fuga: min = %f, minIndex = %d", min, minIndex))
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

	// Cj
	Cj = make(map[string]float64)
	for _, node := range allNodes {
		res := node.GetAvailableResource()
		for s, q := range res.Resources {
			if _, ok := Cj[s]; ok {
				Cj[s] += float64(q)
			} else {
				Cj[s] = float64(q)
			}
		}
	}

	// lamda

	return nil
}

func check(node *objects.Node, com combination) bool {
	r := resources.NewResource()
	for appID, num := range com {
		// app := allApps[appID]
		// asks := allRequests[appID]
		// var req *resources.Resource
		// for _, ask := range asks {
		// 	req = ask.GetAllocatedResource()
		// 	break
		// }
		req := allRequests[appID]
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
	log.Logger().Info(fmt.Sprintf("fuga: enter find(), com = %+v, l = %d, m = %d, minu(com, m) = %f",
		com, l, m, minu(&com, m)))

	if ss.shouldAdd(&com, m) { // TODO
		ss.push(&com, m)
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
		minIndex: 0,
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

func v(A *allocationDecision) float64 {
	// TA := make(map[int]*resources.Resource)
	// for _, com := range *A {
	// 	if com == nil {
	// 		log.Logger().Info("fuga: com is nil")
	// 	}
	// 	for i, num := range *com {
	// 		if _, ok := TA[i]; ok {
	// 			TA[i] = resources.Add(TA[i], resources.Multiply(allRequests[i], num))
	// 		} else {
	// 			TA[i] = resources.Multiply(allRequests[i], num)
	// 		}
	// 	}
	// }
	// Z := 0.0
	// for i, res := range TA {
	// 	for s, q := range res.Resources {
	// 		var z float64
	// 		z = float64(q) / Cj[s] // TODO
	// 		z = math.Abs(z - (lamda * userList[i].d[s]))
	// 		z = math.Pow(z, alpha-1)
	// 		Z += z
	// 	}
	// }
	// return Z
	return 0.0
}

func skew(com *combination, m int) float64 {
	temp1u := u(com, resources.VCORE, m)
	temp2u := u(com, resources.MEMORY, m)
	ua := (temp1u + temp2u) / 2
	sk := 0.0
	sk += math.Pow(temp1u/ua-1, 2)
	sk += math.Pow(temp2u/ua-1, 2)
	sk = math.Pow(sk, 0.5)
	return sk
	return 0.0
}

func sgn(x float64) float64 {
	if x > 0 {
		return 1.0
	} else if x < 0 {
		return -1.0
	}
	return 0.0
}

func U(A *allocationDecision, m int) float64 {
	com := (*A)[m]
	log.Logger().Info(fmt.Sprintf("fuga: com[%d] = %+v", m, com))
	if com == nil {
		return 0.0
	}
	return sgn(1-alpha)*v(A) - skew(com, m)
}

func G() (*allocationDecision, error) {
	var err error

	// Step 2: Strategies Set for Each Player
	log.Logger().Info(fmt.Sprintf("fuga: Step 2: Strategies Set for Each Player"))
	o := make([]*strategiesSet, numNodes)
	for m := 0; m < numNodes; m++ {
		log.Logger().Info(fmt.Sprintf("fuga: node %d(%s) getStrategies()", m, allNodes[m].NodeID))
		o[m], err = getStrategies(m)
		if err != nil {
			return nil, err
		}
	}

	// TODO
	log.Logger().Info(fmt.Sprintf("fuga: o[numNodes]*strategiesSet:"))
	for m := 0; m < numNodes; m++ {
		log.Logger().Info(fmt.Sprintf("fuga: node %d(%s), o[%d]", m, allNodes[m].NodeID, m))

		tmpSs := o[m]
		if tmpSs == nil {
			break
		}

		for i := 0; i < eta; i++ {
			com := tmpSs.coms[i]
			if com == nil {
				break
			}
			log.Logger().Info(fmt.Sprintf("fuga: %dth combination: com = %+v, minu(com, m) = %f",
				i, com, minu(com, m)))
		}
	}
	// TODO

	// // Step 3: Generate the Extension-form Game Tree
	// log.Logger().Info(fmt.Sprintf("fuga: Step 3: Generate the Extension-form Game Tree"))
	// // for indices
	// indices := make([]int, numNodes)
	// type pair struct {
	// 	index int
	// 	min   float64
	// }
	// tmp := make([]*pair, numNodes)
	// for i, p := range o {
	// 	tmpPair := &pair{
	// 		index: i,
	// 		min:   p.min,
	// 	}
	// 	tmp[i] = tmpPair
	// }
	// sort.SliceStable(tmp, func(i, j int) bool {
	// 	l := tmp[i]
	// 	r := tmp[j]
	// 	return l.min < r.min
	// })
	// for i, t := range tmp {
	// 	indices[i] = t.index
	// }

	// // Step 4: Find the SPNE for a game G
	// log.Logger().Info(fmt.Sprintf("fuga: Step 4: Find the SPNE for a game G"))
	// // var selection [numNodes]int
	// selection := make([]int, numNodes)
	// // var alloc allocationDecision
	// alloc := make(allocationDecision)

	// // numNodes and numNodes-1
	// theLast := o[indices[numNodes-1]].coms
	// theSecondLast := o[indices[numNodes-2]].coms
	// var tableX [eta][eta]float64
	// var tableY [eta][eta]float64
	// var max [eta]int
	// var x, y int
	// for x = 0; x < eta; x++ {
	// 	var maxU float64
	// 	for y = 0; y < eta; y++ {
	// 		alloc[indices[numNodes-1]] = theLast[y]
	// 		alloc[indices[numNodes-2]] = theSecondLast[x]

	// 		tableX[x][y] = U(&alloc, x)
	// 		tableY[x][y] = U(&alloc, y)
	// 		if y == 0 || tableY[x][y] > maxU {
	// 			maxU = tableY[x][y]
	// 			max[x] = y
	// 		}
	// 	}
	// }

	// var maxU float64
	// var maxX int
	// var maxY int
	// for x = 0; x < eta; x++ {
	// 	y = max[x]
	// 	if x == 0 || tableX[x][y] > maxU {
	// 		maxU = tableX[x][y]
	// 		maxX = x
	// 		maxY = y
	// 	}
	// }
	// selection[indices[numNodes-2]] = maxX
	// selection[indices[numNodes-1]] = maxY

	// alloc[indices[numNodes-1]] = o[indices[numNodes-1]].coms[maxY]
	// alloc[indices[numNodes-2]] = o[indices[numNodes-2]].coms[maxX]

	// var i int
	// for i = numNodes - 2; i >= 0; i-- {
	// 	m := indices[i]
	// 	coms := o[m].coms
	// 	var maxU float64
	// 	var maxX int
	// 	for x = 0; x < eta; x++ {
	// 		com := coms[x]
	// 		alloc[m] = com
	// 		tmpU := U(&alloc, m)
	// 		if x == 0 || tmpU > maxU {
	// 			maxU = tmpU
	// 			maxX = x
	// 		}
	// 	}
	// 	com := coms[maxX]
	// 	alloc[m] = com
	// }

	// return &alloc, nil
	return nil, nil
}

func fuga(apps []*objects.Application, nodes []*objects.Node) error {
	log.Logger().Info("fuga: enter fuga()------------------------")

	// Step 1: Pre-combination Phase
	log.Logger().Info("fuga: Step 1: Pre-combination Phase")
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

	// TODO
	log.Logger().Info("fuga: game informations:")
	log.Logger().Info("fuga: allNodes:")
	for m, node := range allNodes {
		availible := node.GetAvailableResource().Resources
		capacity := node.GetCapacity().Resources
		log.Logger().Info(fmt.Sprintf("fuga: node %d(%s), capacity(mem:%d, vcore:%d), availible(mem:%d, vcore:%d)",
			m, node.NodeID, capacity[resources.MEMORY], capacity[resources.VCORE], availible[resources.MEMORY], availible[resources.VCORE]))
	}
	log.Logger().Info("fuga: allRequests:")
	for i, app := range allApps {
		req := allRequests[i].Resources
		log.Logger().Info(fmt.Sprintf("fuga: app %d(%s), req(mem:%d, vcore:%d)",
			i, app.ApplicationID, req[resources.MEMORY], req[resources.VCORE]))
	}
	// TODO

	ad, err := G()
	if ad != nil {
		log.Logger().Info(fmt.Sprintf("fuga: ad: %+v\n", ad))
		for m, com := range *ad {
			log.Logger().Info(fmt.Sprintf("fuga: com[%d] = %+v", m, com))
		}
		log.Logger().Info("fuga: haha\n")
	} else {
		log.Logger().Info("fuga: baba\n")
	}
	return nil
}
