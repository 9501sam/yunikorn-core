package scheduler

import (
	"fmt"
	"math"
	"sort"

	"github.com/apache/yunikorn-core/pkg/common/resources"
	"github.com/apache/yunikorn-core/pkg/log"
	"github.com/apache/yunikorn-core/pkg/scheduler/objects"
)

const (
	eta   = 3
	alpha = 2
)

type Player struct {
	node       *objects.Node
	combinList [eta]nodeDecision
}

type User struct {
	app *objects.Application
	req *resources.Resource
	d   map[string]float64
}

var numUser int
var numNode int

var playerList []Player
var userList []User

var applications []*objects.Application

// var A *allocationDecision
var Cj map[string]float64
var lamda float64

type resourceCapacity map[string]*resources.Resource
type resourceRequirement map[string]*resources.Resource
type nodeDecision map[int]int64
type allocationDecision map[int]*nodeDecision

// combination of a node
type combinations struct {
	coms []*nodeDecision // coms[1 ~ eta][1 ~ s]
	min  float64
}

func initGame(apps []*objects.Application, nodes []*objects.Node) {
	numUser = len(apps)
	numNode = len(nodes)

	pList := make([]Player, numNode)
	var nD nodeDecision
	var cl [eta]nodeDecision
	for i := 0; i < eta; i++ {
		cl[i] = nD
	}
	for i, n := range nodes {
		p := Player{
			node:       n,
			combinList: cl,
		}
		pList[i] = p
	}
	playerList = pList

	applications = apps
	uList := make([]User, numUser)
	for i, a := range apps {
		asks := a.GetAllRequests()
		res := resources.NewResource()
		for _, ask := range asks {
			res = ask.GetAllocatedResource()
			break
		}
		dom := make(map[string]float64)
		uer := User{
			app: a,
			req: res,
			d:   dom,
		}
		uList[i] = uer
	}
	userList = uList

	// R
	// C
	// Cj
	// Cj
	Cj = make(map[string]float64)
	for _, p := range playerList {
		n := p.node
		res := n.GetAvailableResource()
		for s, q := range res.Resources {
			if _, ok := Cj[s]; ok {
				Cj[s] += float64(q)
			} else {
				Cj[s] = float64(q)
			}
		}
	}

}

func dominant() {
	largest := 0.0
	for _, user := range userList {
		for _, q := range user.req.Resources {
			if float64(q) >= largest {
				largest = float64(q)
			}
		}
		for s, q := range user.req.Resources {
			user.d[s] = float64(q) / largest
		}
	}
	L := 0.0

	sum1 := 0.0
	sum2 := 0.0

	for _, uer := range userList {
		sum1 += uer.d["vcore"]
		sum2 += uer.d["memory"]
	}
	if sum1 >= sum2 {
		L = sum1
	} else {
		L = sum2
	}
	lamda = 1 / L
}

func v(A *allocationDecision) float64 {
	TA := make(map[int]*resources.Resource)
	for _, nd := range *A {
		if nd == nil {
			log.Logger().Info("nD is nil")
		}
		for i, num := range *nd {
			if _, ok := TA[i]; ok {
				TA[i] = resources.Add(TA[i], resources.Multiply(userList[i].req, num))
			} else {
				TA[i] = resources.Multiply(userList[i].req, num)
			}
		}
	}
	Z := 0.0
	for i, res := range TA {
		for s, q := range res.Resources {
			var z float64
			z = float64(q) / Cj[s]
			z = math.Abs(z - (lamda * userList[i].d[s]))
			z = math.Pow(z, alpha-1)
			Z += z
		}
	}
	return Z

}

func u(nD nodeDecision, restype string, m int) float64 {
	n := playerList[m].node
	x := 0.0
	for i, num := range nD {
		res := userList[i].req
		x += float64(res.Resources[restype]) * float64(num)
	}
	/*for i := 0; i < numNode; i++ {
		res := userList[i].req
		x += float64(res.Resources[restype]) * float64((*nD)[i])
	}*/
	remres := n.GetAvailableResource().Resources
	initres := n.GetCapacity().Resources
	return (float64(remres[restype]) - x) / float64(initres[restype])
}

func findminu(nD *nodeDecision, m int) float64 {
	if nD == nil {
		return 0.0
	}
	temp1 := u(*nD, "vcore", m)
	temp2 := u(*nD, "memory", m)
	if temp1 >= temp2 {
		return temp1
	} else {
		return temp2
	}
}

func skew(com *nodeDecision, m int) float64 {
	temp1u := u(*com, "vcore", m)
	temp2u := u(*com, "memory", m)
	ua := (temp1u + temp2u) / 2
	sk := 0.0
	sk += math.Pow(temp1u/ua-1, 2)
	sk += math.Pow(temp2u/ua-1, 2)
	sk = math.Pow(sk, 0.5)
	return sk
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
	c := (*A)[m]
	log.Logger().Info(fmt.Sprintf("com[%d] = %+v", m, c))
	if c == nil {
		return 0.0
	}
	return sgn(1-alpha)*v(A) - skew(c, m)
}

var allCombinations []nodeDecision

func check(n *objects.Node, com nodeDecision) bool {
	r := resources.NewResource()
	for appID, num := range com {
		app := applications[appID]
		asks := app.GetAllRequests()
		var req *resources.Resource
		for _, ask := range asks {
			req = ask.GetAllocatedResource()
			break
		}
		r = resources.Add(r, resources.Multiply(req, num))
	}
	if n.GetAvailableResource().FitInMaxUndef(r) {
		return true
	} else {
		return false
	}
}

var Om *combinations

func find(com nodeDecision, l int, m int) {
	n := playerList[m].node
	if check(n, com) {
		if findminu(&com, m) > Om.min {
			log.Logger().Info("findminu > Om.min")
			var mini int
			minu := 1.0
			for i := 0; i < eta; i++ {
				c := Om.coms[i]
				if findminu(c, m) == Om.min {
					mini = i
					break
				}
			}
			log.Logger().Info(fmt.Sprintf("Om.coms[%d] change", mini))
			Om.coms[mini] = &com
			for i := 0; i < eta; i++ {
				c := Om.coms[i]
				if findminu(c, m) < minu {
					minu = findminu(c, m)
				}
			}
			Om.min = minu
		}
		/*newCom := com
		allCombinations = append(allCombinations, newCom)*/
		var i int
		for i = l; i < numUser; i++ {
			com[i]++
			if com[i] > int64(len(userList[i].app.GetAllRequests())) {
				return
			}
			find(com, i, m)
			com[i]--
		}
	} else {
		//log.Logger().Info(fmt.Sprintf("l = %d", l))
		return
	}
}

func getCombinations(apps []*objects.Application, m int) {
	var l int
	// var Om *combinations
	Om = &combinations{
		coms: make([]*nodeDecision, eta),
		min:  0.0,
	}
	// var com nodeDecision
	com := make(nodeDecision)

	l = 0
	var i int
	for i, _ = range apps {
		com[i] = 0
	}

	//allCombinations = make([]nodeDecision)
	find(com, l, m)

	log.Logger().Info(fmt.Sprintf("m = %d", m))
	log.Logger().Info(fmt.Sprintf("len = %d\n", len(allCombinations)))
	// for i, nd := range allCombinations {
	// 	var j int
	// 	for j = 0; j < numUser; j++ {
	// 		log.Logger().Info(fmt.Sprintf("app: %+v, num: %d\n", j, nd[i]))
	// 	}
	// }

	/*for i := 0; i < 2; i++ {
		log.Logger().Info(fmt.Sprintf("com%d", i))
		com := allCombinations[i]
		for j := 0; j < numUser; j++ {
			log.Logger().Info(fmt.Sprintf("app: %+v, num: %d", j, (com)[j]))
		}
	}*/

	/*if len(allCombinations) < 10 {
		return nil
	}*/

	/*for i = 0; i < eta; i++ {
		maxu := 0.0
		var maxj int
		for j, c := range allCombinations {
			// com := c
			if findminu(&c, m) > maxu {
				maxj = j
				maxu = findminu(&c, m)
			}
		}

		log.Logger().Info(fmt.Sprintf("com%d", i))
		log.Logger().Info(fmt.Sprintf("maxj: %d", maxj))
		log.Logger().Info(fmt.Sprintf("maxu: %f", maxu))
		Om.coms[i] = &allCombinations[maxj]
		c := Om.coms[i]
		for j := 0; j < numUser; j++ {
			log.Logger().Info(fmt.Sprintf("app: %+v, num: %d", j, (*c)[j]))
		}
		log.Logger().Info(fmt.Sprintf("before remove allCombinations[%d]: %+v", maxj, allCombinations[maxj]))
		allCombinations = append(allCombinations[:maxj], allCombinations[maxj+1:]...)
		allCombinations = allCombinations[:len(allCombinations)-1]
	}

	var minu float64
	for i = 0; i < eta; i++ {
		utilization := findminu(Om.coms[i], m)
		if i == 0 || utilization < minu {
			minu = utilization
		}
	}*/

	log.Logger().Info(fmt.Sprintf("m = %d", m))
	coms := Om.coms

	log.Logger().Info(fmt.Sprintf("Om.com: %+v", coms))
	log.Logger().Info(fmt.Sprintf("Om.min: %+v", Om.min))
	for i := 0; i < 1; i++ {
		com := coms[i]
		log.Logger().Info(fmt.Sprintf("com%d", i))
		for j := 0; j < numUser; j++ {
			log.Logger().Info(fmt.Sprintf("app: %+v, num: %d", j, (*com)[j]))
		}
	}

	//Om.min = minu

	//return Om
}

func G(apps []*objects.Application, nodes []*objects.Node) *allocationDecision {
	p := numNode

	// Step 2: Strategies Set for Each Player
	o := make([]*combinations, numNode) // o[1 ~ p]
	for m, _ := range nodes {
		getCombinations(apps, m)
		com := Om
		if com != nil {
			o[m] = com
		} else {
			return nil
		}
	}

	// Step 3: Generate the Extension-form Game Tree
	// for indices
	indices := make([]int, p)
	type pair struct {
		index int
		min   float64
	}
	tmp := make([]*pair, p)
	for i, p := range o {
		tmpPair := &pair{
			index: i,
			min:   p.min,
		}
		tmp[i] = tmpPair
	}
	sort.SliceStable(tmp, func(i, j int) bool {
		l := tmp[i]
		r := tmp[j]
		return l.min < r.min
	})
	for i, t := range tmp {
		indices[i] = t.index
	}

	// Step 4: Find the SPNE for a game G
	// var selection [p]int
	selection := make([]int, p)
	// var alloc allocationDecision
	alloc := make(allocationDecision)

	// p and p-1
	theLast := o[indices[p-1]].coms
	theSecondLast := o[indices[p-2]].coms
	var tableX [eta][eta]float64
	var tableY [eta][eta]float64
	var max [eta]int
	var x, y int
	for x = 0; x < eta; x++ {
		var maxU float64
		for y = 0; y < eta; y++ {
			alloc[indices[p-1]] = theLast[y]
			alloc[indices[p-2]] = theSecondLast[x]

			tableX[x][y] = U(&alloc, x)
			tableY[x][y] = U(&alloc, y)
			if y == 0 || tableY[x][y] > maxU {
				maxU = tableY[x][y]
				max[x] = y
			}
		}
	}

	var maxU float64
	var maxX int
	var maxY int
	for x = 0; x < eta; x++ {
		y = max[x]
		if x == 0 || tableX[x][y] > maxU {
			maxU = tableX[x][y]
			maxX = x
			maxY = y
		}
	}
	selection[indices[p-2]] = maxX
	selection[indices[p-1]] = maxY

	alloc[indices[p-1]] = o[indices[p-1]].coms[maxY]
	alloc[indices[p-2]] = o[indices[p-2]].coms[maxX]

	var i int
	for i = p - 2; i >= 0; i-- {
		m := indices[i]
		coms := o[m].coms
		var maxU float64
		var maxX int
		for x = 0; x < eta; x++ {
			com := coms[x]
			alloc[m] = com
			tmpU := U(&alloc, m)
			if x == 0 || tmpU > maxU {
				maxU = tmpU
				maxX = x
			}
		}
		com := coms[maxX]
		alloc[m] = com
	}

	return &alloc
}

func fuga(apps []*objects.Application, nodes []*objects.Node) {
	log.Logger().Info("enter fuga()")

	// Step 1: Pre-combination Phase
	var players []*objects.Node
	for _, n := range nodes {
		if resources.StrictlyGreaterThanZero(n.GetAvailableResource()) {
			players = append(players, n)
		}
	}
	initGame(apps, players)

	ad := G(apps, players)
	if ad != nil {
		log.Logger().Info(fmt.Sprintf("ad: %+v\n", ad))
		var i int
		var j int
		for i = 0; i < numNode; i++ {
			log.Logger().Info(fmt.Sprintf("node: %+v\n", i))
			for j = 0; j < numUser; j++ {
				nd := (*ad)[i]
				log.Logger().Info(fmt.Sprintf("app: %+v, num: %d\n", j, (*nd)[j]))
			}
		}
		log.Logger().Info("haha")
	}
}
