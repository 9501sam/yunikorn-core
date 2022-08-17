package scheduler

import (
	"sort"

	"github.com/apache/yunikorn-core/pkg/common/resources"
	"github.com/apache/yunikorn-core/pkg/log"
	"github.com/apache/yunikorn-core/pkg/scheduler/objects"
)

const (
	eta   = 3
	alpha = 2
)

var numUser int
var numNode int

// var A *allocationDecision
var R *resourceRequirement
var C *resourceCapacity
var Cj []float64
var phi [][]float64
var d [][]float64
var lamda float64

type resourceCapacity map[string]*resources.Resource
type resourceRequirement map[string]*resources.Resource
type nodeDecision map[int]int
type allocationDecision map[int]*nodeDecision

// combination of a node
type combinations struct {
	coms [eta]*nodeDecision // coms[1 ~ eta][1 ~ s]
	min  float64
}

func initGame(apps []*objects.Application, nodes []*objects.Node) {
	// numUser
	// numNode

	// R
	// C
	// Cj
}

func v(A *allocationDecision, m int) float64 {
	return 0
}

func u() float64 {
	return 0.0
}

func skew(A *allocationDecision, m int) float64 {
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
	return sgn(1-alpha)*v(A, m) - skew(A, m)
}

func getCombinations(apps []*objects.Application, node *objects.Node) *combinations {
	return nil
}

func G(apps []*objects.Application, nodes []*objects.Node) *allocationDecision {
	// s := len(apps)
	// k := 2
	p := len(nodes)

	// Step 2: Strategies Set for Each Player
	o := make([]*combinations, p) // o[1 ~ p]
	for i, p := range nodes {
		o[i] = getCombinations(apps, p)
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
		tmp[i].index = i
		tmp[i].min = p.min
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
	var alloc allocationDecision

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
			// A := Ax + Ay // TODO
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
		log.Logger().Info("haha")
	}
}
