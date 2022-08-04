package scheduler

import (
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
type nodeDecision map[string]int
type allocationDecision map[string]*nodeDecision

func initGame(apps []*objects.Application, nodes []*objects.Node) {
	// numUser
	// numNode

	// R
	// C
	// Cj
}

func v() float64 {
}

func u() float64 {
}

func skew(A *allocationDecision, int m) float64 {
}

func sgn() float64 {
}

func U(A *allocationDecision, int m) float64 {
	return sgn(1-alpha)*v(A) - ske(A, m)
}

func G() *allocationDecision {
}

func fuga(apps []*objects.Application, nodes []*objects.Node) {
	log.Logger().Info("enter fuga()")
	initGame(apps, nodes)
	alloc := G()
}
